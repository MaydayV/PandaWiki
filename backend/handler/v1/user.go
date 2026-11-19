package v1

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/user/v1"
	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/pkg/ratelimit"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/usecase"
)

type UserHandler struct {
	*handler.BaseHandler
	usecase     *usecase.UserUsecase
	logger      *log.Logger
	config      *config.Config
	auth        middleware.AuthMiddleware
	rateLimiter *ratelimit.RateLimiter
}

func NewUserHandler(e *echo.Echo, baseHandler *handler.BaseHandler, logger *log.Logger, usecase *usecase.UserUsecase, auth middleware.AuthMiddleware, config *config.Config, cache *cache.Cache) *UserHandler {
	handlerLogger := logger.WithModule("handler.v1.user")
	h := &UserHandler{
		BaseHandler: baseHandler,
		logger:      handlerLogger,
		usecase:     usecase,
		auth:        auth,
		config:      config,
		rateLimiter: ratelimit.NewRateLimiter(handlerLogger, cache),
	}
	group := e.Group("/api/v1/user")
	group.POST("/login", h.Login)

	group.GET("", h.GetUserInfo, h.auth.Authorize)
	// 全量用户列表（含角色）仅管理员可见，普通协作者可枚举账号属信息泄露
	group.GET("/list", h.ListUsers, h.auth.Authorize, h.auth.ValidateUserRole(consts.UserRoleAdmin))
	group.POST("/create", h.CreateUser, h.auth.Authorize, h.auth.ValidateUserRole(consts.UserRoleAdmin))
	group.PUT("/reset_password", h.ResetPassword, h.auth.Authorize, h.auth.ValidateUserRole(consts.UserRoleAdmin))
	group.DELETE("/delete", h.DeleteUser, h.auth.Authorize, h.auth.ValidateUserRole(consts.UserRoleAdmin))

	return h
}

// CreateUser
//
//	@Summary		CreateUser
//	@Description	CreateUser
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			body	body		v1.CreateUserReq	true	"CreateUser Request"
//	@Success		200		{object}	domain.Response{data=v1.CreateUserResp}
//	@Router			/api/v1/user/create [post]
func (h *UserHandler) CreateUser(c echo.Context) error {
	var req v1.CreateUserReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	uid := uuid.New().String()
	err := h.usecase.CreateUser(c.Request().Context(), &domain.User{
		ID:       uid,
		Account:  req.Account,
		Password: req.Password,
		Role:     req.Role,
	}, consts.GetLicenseEdition(c))
	if err != nil {
		return h.NewResponseWithError(c, "failed to create user", err)
	}

	return h.NewResponseWithData(c, v1.CreateUserResp{ID: uid})
}

// Login
//
//	@Summary		Login
//	@Description	Login
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			body	body		v1.LoginReq	true	"Login Request"
//	@Success		200		{object}	v1.LoginResp
//	@Router			/api/v1/user/login [post]
func (h *UserHandler) Login(c echo.Context) error {
	var req v1.LoginReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	ctx := c.Request().Context()
	ip := c.RealIP()
	locked, remaining := h.rateLimiter.CheckIPLocked(ctx, ip)
	if locked {
		h.logger.Warn("IP is locked", "ip", ip, "remaining", remaining)
		return h.NewResponseWithError(c, fmt.Sprintf("账号已被锁定，请 %s 后重试", remaining.String()), nil)
	}

	token, err := h.usecase.VerifyUserAndGenerateToken(ctx, req)
	if err != nil {
		h.rateLimiter.LockAttempt(ctx, ip)
		return h.NewResponseWithError(c, "用户名或密码错误", err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("reset login attempts panicked", "panic", r)
			}
		}()
		if err := h.rateLimiter.ResetLoginAttempts(context.Background(), ip); err != nil {
			h.logger.Error("failed to reset login attempts", "error", err, "ip", ip)
		}
	}()

	return h.NewResponseWithData(c, v1.LoginResp{Token: token})
}

// GetUserInfo
//
//	@Summary		GetUser
//	@Description	GetUser
//	@Tags			user
//	@Accept			json
//	@Success		200	{object}	v1.UserInfoResp
//	@Router			/api/v1/user [get]
func (h *UserHandler) GetUserInfo(c echo.Context) error {
	ctx := c.Request().Context()
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return h.NewResponseWithError(c, "authInfo not found in context", nil)
	}

	user, err := h.usecase.GetUser(c.Request().Context(), authInfo.UserId)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get user", err)
	}

	userInfo := &v1.UserInfoResp{
		ID:         user.ID,
		Account:    user.Account,
		Role:       user.Role,
		IsToken:    authInfo.IsToken,
		LastAccess: &user.LastAccess,
		CreatedAt:  user.CreatedAt,
	}

	return h.NewResponseWithData(c, userInfo)
}

// ListUsers
//
//	@Summary		ListUsers
//	@Description	ListUsers
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	domain.PWResponse{data=v1.UserListResp}
//	@Router			/api/v1/user/list [get]
func (h *UserHandler) ListUsers(c echo.Context) error {
	var req v1.UserListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	users, err := h.usecase.ListUsers(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "failed to list users", err)
	}
	return h.NewResponseWithData(c, users)
}

// ResetPassword
//
//	@Summary		ResetPassword
//	@Description	ResetPassword
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			body	body		v1.ResetPasswordReq	true	"ResetPassword Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/user/reset_password [put]
func (h *UserHandler) ResetPassword(c echo.Context) error {
	ctx := c.Request().Context()
	var req v1.ResetPasswordReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return h.NewResponseWithError(c, "authInfo not found in context", nil)
	}

	if authInfo.IsToken {
		return h.NewResponseWithError(c, "this api not support token call", nil)
	}

	user, err := h.usecase.GetUser(ctx, authInfo.UserId)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get user", err)
	}

	if user.Account == "admin" {
		// admin 改不了自己的密码
		if authInfo.UserId == req.ID {
			return h.NewResponseWithError(c, "请修改安装目录下 .env 文件中的 ADMIN_PASSWORD，并重启 panda-wiki-api 容器使更改生效。", nil)
		}
	} else {
		targetUser, err := h.usecase.GetUser(ctx, req.ID)
		if err != nil {
			return h.NewResponseWithError(c, "failed to get target user", err)
		}

		// 超级管理员不能改其他超级管理员密码
		if targetUser.Role == consts.UserRoleAdmin && targetUser.ID != authInfo.UserId {
			return h.NewResponseWithError(c, "无法修改其他超级管理员密码", nil)
		}
	}

	err = h.usecase.ResetPassword(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, "failed to reset password", err)
	}

	return h.NewResponseWithData(c, nil)
}

// DeleteUser
//
//	@Summary		DeleteUser
//	@Description	DeleteUser
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			params	query		v1.DeleteUserReq	true	"DeleteUser Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/user/delete [delete]
func (h *UserHandler) DeleteUser(c echo.Context) error {
	ctx := c.Request().Context()

	var req v1.DeleteUserReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return h.NewResponseWithError(c, "authInfo not found in context", nil)
	}

	if authInfo.IsToken {
		return h.NewResponseWithError(c, "this api not support token call", nil)
	}

	if authInfo.UserId == req.UserID {
		return h.NewResponseWithError(c, "cannot delete yourself", nil)
	}

	user, err := h.usecase.GetUser(ctx, authInfo.UserId)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get user", err)
	}

	targetUser, err := h.usecase.GetUser(ctx, req.UserID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get target user", err)
	}

	if targetUser.Account == "admin" {
		return h.NewResponseWithError(c, "cannot delete admin user", nil)
	}

	// 非admin账号的管理员只能删除普通用户的账户
	if user.Account != "admin" {
		if targetUser.Role != consts.UserRoleUser {
			return h.NewResponseWithError(c, "cannot delete other admin users", nil)
		}
	}

	err = h.usecase.DeleteUser(ctx, req.UserID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to delete user", err)
	}

	return h.NewResponseWithData(c, nil)
}
