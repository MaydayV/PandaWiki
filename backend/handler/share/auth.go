package share

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/share/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/pkg/ratelimit"
	storecache "github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/usecase"
)

type ShareAuthHandler struct {
	*handler.BaseHandler
	logger      *log.Logger
	rateLimiter *ratelimit.RateLimiter
	kbUsecase   *usecase.KnowledgeBaseUsecase
	authUsecase *usecase.AuthUsecase
}

func NewShareAuthHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	cache *storecache.Cache,
	kbUsecase *usecase.KnowledgeBaseUsecase,
	authUsecase *usecase.AuthUsecase,
) *ShareAuthHandler {
	handlerLogger := logger.WithModule("handler.share.auth")
	h := &ShareAuthHandler{
		BaseHandler: baseHandler,
		logger:      handlerLogger,
		rateLimiter: ratelimit.NewRateLimiter(handlerLogger, cache),
		kbUsecase:   kbUsecase,
		authUsecase: authUsecase,
	}

	shareAuthMiddleware := middleware.NewShareAuthMiddleware(logger, kbUsecase, authUsecase)

	share := e.Group("share/v1/auth", shareAuthMiddleware.CheckForbidden)
	share.GET("/get", h.AuthGet)
	share.POST("/login/simple", h.AuthLoginSimple)
	share.POST("/github", h.AuthGitHub)
	return h
}

// AuthGet auth获取
//
//	@Tags			share_auth
//	@Summary		AuthGet
//	@Description	AuthGet
//	@ID				v1-AuthGet
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string			true	"kb_id"
//	@Param			param	query		v1.AuthGetReq	true	"para"
//	@Success		200		{object}	domain.PWResponse{data=v1.AuthGetResp}
//	@Router			/share/v1/auth/get [get]
func (h *ShareAuthHandler) AuthGet(c echo.Context) error {
	ctx := c.Request().Context()

	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	kb, err := h.kbUsecase.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get knowledge base detail", err)
	}

	resp := &v1.AuthGetResp{
		AuthType:       kb.AccessSettings.GetAuthType(),
		SourceType:     kb.AccessSettings.SourceType,
		LicenseEdition: consts.GetLicenseEdition(c),
	}
	return h.NewResponseWithData(c, resp)
}

// AuthLoginSimple 简单口令登录
//
//	@Tags			share_auth
//	@Summary		AuthLoginSimple
//	@Description	AuthLoginSimple
//	@ID				v1-AuthLoginSimple
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string					true	"kb_id"
//	@Param			param	body		v1.AuthLoginSimpleReq	true	"para"
//	@Success		200		{object}	domain.Response
//	@Router			/share/v1/auth/login/simple [post]
func (h *ShareAuthHandler) AuthLoginSimple(c echo.Context) error {
	ctx := c.Request().Context()

	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	// 匿名端点加 IP 锁定限速（与 /api/v1/user/login 同模式），防在线爆破共享口令
	ip := c.RealIP()
	if locked, remaining := h.rateLimiter.CheckIPLocked(ctx, ip); locked {
		return h.NewResponseWithError(c, fmt.Sprintf("too many attempts, please try again in %d seconds", int(remaining.Seconds())), nil)
	}

	var req v1.AuthLoginSimpleReq
	if err := c.Bind(&req); err != nil {
		h.logger.Error("parse request failed", log.Error(err))
		return h.NewResponseWithError(c, "AuthGet bind failed", nil)
	}

	kb, err := h.kbUsecase.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get knowledge base detail", err)
	}

	if !kb.AccessSettings.SimpleAuth.Enabled {
		return h.NewResponseWithError(c, "simple auth is not enabled", nil)
	}

	// 常量时间比较，避免公开端点被时序侧信道逐字节爆破共享口令
	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(kb.AccessSettings.SimpleAuth.Password)) != 1 {
		h.rateLimiter.LockAttempt(ctx, ip)
		return h.NewResponseWithError(c, "simple auth password is incorrect", nil)
	}
	_ = h.rateLimiter.ResetLoginAttempts(ctx, ip)

	s := c.Get(domain.SessionCacheKey)
	if s == nil {
		return h.NewResponseWithError(c, "get session cache key failed", nil)
	}
	store := s.(sessions.Store)

	newSess := sessions.NewSession(store, domain.SessionName)
	newSess.IsNew = true

	newSess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
	}

	newSess.Values["kb_id"] = kb.ID

	if err := newSess.Save(c.Request(), c.Response()); err != nil {
		return h.NewResponseWithError(c, "save session failed", nil)
	}

	return h.NewResponseWithData(c, nil)
}

// AuthGitHub GitHub登录
//
//	@Tags			ShareAuth
//	@Summary		GitHub登录
//	@Description	GitHub登录
//	@ID				v1-AuthGitHub
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string				true	"kb id"
//	@Param			param	body		v1.AuthGitHubReq	true	"para"
//	@Success		200		{object}	domain.PWResponse{data=v1.AuthGitHubResp}
//	@Router			/share/v1/auth/github [post]
func (h *ShareAuthHandler) AuthGitHub(c echo.Context) error {
	ctx := context.WithValue(c.Request().Context(), consts.ContextKeyEdition, consts.GetLicenseEdition(c))

	var req v1.AuthGitHubReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	req.KbID = kbID

	valid, err := h.authUsecase.ValidateRedirectUrl(ctx, req.KbID, req.RedirectUrl)
	if err != nil || !valid {
		return h.NewResponseWithError(c, "invalid redirect url", err)
	}

	url, err := h.authUsecase.GenerateGitHubAuthUrl(ctx, req)
	if err != nil {
		return h.NewResponseWithError(c, "GenerateGitHubAuthUrl failed", err)
	}

	return h.NewResponseWithData(c, v1.AuthGitHubResp{
		Url: url,
	})
}
