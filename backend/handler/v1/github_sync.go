package v1

import (
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type GitHubSyncHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.GitHubSyncUsecase
	auth    middleware.AuthMiddleware
}

func NewGitHubSyncHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	usecase *usecase.GitHubSyncUsecase,
	auth middleware.AuthMiddleware,
) *GitHubSyncHandler {
	h := &GitHubSyncHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.github_sync"),
		usecase:     usecase,
		auth:        auth,
	}

	group := e.Group("/api/v1/github_sync", h.auth.Authorize, h.auth.ValidateUserRole(consts.UserRoleAdmin))
	group.GET("/config", h.GetConfig)
	group.PUT("/config", h.SaveConfig)
	group.POST("/sync", h.TriggerSync)
	return h
}

// GetConfig
//
//	@Summary		Get GitHub sync config
//	@Description	Get GitHub sync config for the KB
//	@Tags			github_sync
//	@Produce		json
//	@Param			kb_id	query		string	true	"Knowledge Base ID"
//	@Security		bearerAuth
//	@Success		200	{object}	domain.PWResponse{data=domain.GitHubSyncSetting}
//	@Router			/api/v1/github_sync/config [get]
func (h *GitHubSyncHandler) GetConfig(c echo.Context) error {
	kbID := c.QueryParam("kb_id")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	cfg, err := h.usecase.GetConfig(c.Request().Context(), kbID)
	if err != nil {
		return h.NewResponseWithError(c, "get github sync config failed", err)
	}
	return h.NewResponseWithData(c, cfg)
}

// SaveConfig
//
//	@Summary		Save GitHub sync config
//	@Description	Save GitHub sync config for the KB
//	@Tags			github_sync
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.GitHubSyncSetting	true	"GitHub sync config"
//	@Security		bearerAuth
//	@Success		200	{object}	domain.PWResponse
//	@Router			/api/v1/github_sync/config [put]
func (h *GitHubSyncHandler) SaveConfig(c echo.Context) error {
	kbID := c.QueryParam("kb_id")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	req := domain.GitHubSyncSetting{}
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := h.usecase.SaveConfig(c.Request().Context(), kbID, &req); err != nil {
		return h.NewResponseWithError(c, "save github sync config failed", err)
	}
	return h.NewResponseWithData(c, "success")
}

// TriggerSync
//
//	@Summary		Trigger GitHub sync
//	@Description	Trigger a manual GitHub sync (runs synchronously)
//	@Tags			github_sync
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.GitHubSyncTriggerReq	true	"trigger request"
//	@Security		bearerAuth
//	@Success		200	{object}	domain.PWResponse{data=domain.GitHubSyncResult}
//	@Router			/api/v1/github_sync/sync [post]
func (h *GitHubSyncHandler) TriggerSync(c echo.Context) error {
	req := domain.GitHubSyncTriggerReq{}
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	userID := ""
	if authInfo := domain.GetAuthInfoFromCtx(c.Request().Context()); authInfo != nil {
		userID = authInfo.UserId
	}
	result, err := h.usecase.TriggerSync(c.Request().Context(), req.KBID, userID)
	if err != nil {
		return h.NewResponseWithError(c, "github sync failed", err)
	}
	return h.NewResponseWithData(c, result)
}
