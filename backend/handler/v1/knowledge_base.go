package v1

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type KnowledgeBaseHandler struct {
	*handler.BaseHandler
	usecase    *usecase.KnowledgeBaseUsecase
	llmUsecase *usecase.LLMUsecase
	logger     *log.Logger
	auth       middleware.AuthMiddleware
}

func NewKnowledgeBaseHandler(
	baseHandler *handler.BaseHandler,
	echo *echo.Echo,
	usecase *usecase.KnowledgeBaseUsecase,
	llmUsecase *usecase.LLMUsecase,
	auth middleware.AuthMiddleware,
	logger *log.Logger,
) *KnowledgeBaseHandler {
	h := &KnowledgeBaseHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.knowledge_base"),
		usecase:     usecase,
		llmUsecase:  llmUsecase,
		auth:        auth,
	}

	group := echo.Group("/api/v1/knowledge_base", h.auth.Authorize)
	group.POST("", h.CreateKnowledgeBase, h.auth.ValidateUserRole(consts.UserRoleAdmin))
	group.GET("/list", h.GetKnowledgeBaseList)
	group.GET("/detail", h.GetKnowledgeBaseDetail, h.auth.ValidateKBUserPerm(consts.UserKBPermissionNotNull))
	group.PUT("/detail", h.UpdateKnowledgeBase, h.auth.ValidateKBUserPerm(consts.UserKBPermissionFullControl))
	group.DELETE("/detail", h.DeleteKnowledgeBase, h.auth.ValidateUserRole(consts.UserRoleAdmin))

	// user management
	userGroup := group.Group("/user", h.auth.ValidateKBUserPerm(consts.UserKBPermissionFullControl))
	userGroup.GET("/list", h.KBUserList)
	userGroup.POST("/invite", h.KBUserInvite)
	userGroup.PATCH("/update", h.KBUserUpdate)
	userGroup.DELETE("/delete", h.KBUserDelete)

	// release
	releaseGroup := group.Group("/release", h.auth.ValidateKBUserPerm(consts.UserKBPermissionDocManage))
	releaseGroup.POST("", h.CreateKBRelease)
	releaseGroup.GET("/list", h.GetKBReleaseList)

	promptGroup := echo.Group("/api/pro/v1/prompt", h.auth.Authorize, h.auth.ValidateKBUserPerm(consts.UserKBPermissionFullControl))
	promptGroup.GET("", h.GetPromptSettings)
	promptGroup.POST("", h.UpdatePromptSettings)
	promptGroup.GET("/version/list", h.GetPromptVersionList)
	promptGroup.GET("/version/detail", h.GetPromptVersionDetail)
	promptGroup.POST("/version/rollback", h.RollbackPromptVersion)

	tokenGroup := echo.Group("/api/pro/v1/token", h.auth.Authorize, h.auth.ValidateKBUserPerm(consts.UserKBPermissionFullControl))
	tokenGroup.POST("/create", h.CreateAPIToken)
	tokenGroup.GET("/list", h.GetAPITokenList)
	tokenGroup.PATCH("/update", h.UpdateAPIToken)
	tokenGroup.DELETE("/delete", h.DeleteAPIToken)

	return h
}

// CreateKnowledgeBase
//
//	@Summary		CreateKnowledgeBase
//	@Description	CreateKnowledgeBase
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Param			body	body		domain.CreateKnowledgeBaseReq	true	"CreateKnowledgeBase Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/knowledge_base [post]
func (h *KnowledgeBaseHandler) CreateKnowledgeBase(c echo.Context) error {

	var req domain.CreateKnowledgeBaseReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	req.Hosts = lo.Uniq(req.Hosts)
	req.Ports = lo.Uniq(req.Ports)
	req.SSLPorts = lo.Uniq(req.SSLPorts)

	if len(req.Hosts) == 0 {
		return h.NewResponseWithError(c, "hosts is required", nil)
	}
	if len(req.Ports)+len(req.SSLPorts) == 0 {
		return h.NewResponseWithError(c, "ports is required", nil)
	}

	req.MaxKB = domain.GetBaseEditionLimitation(c.Request().Context()).MaxKb

	did, err := h.usecase.CreateKnowledgeBase(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrPortHostAlreadyExists) {
			return h.NewResponseWithError(c, "端口或域名已被其他知识库占用", nil)
		}
		if errors.Is(err, domain.ErrSyncCaddyConfigFailed) {
			return h.NewResponseWithError(c, "端口可能已被其他程序占用，请检查", nil)
		}
		return h.NewResponseWithError(c, "failed to create knowledge base", err)
	}

	return h.NewResponseWithData(c, map[string]string{
		"id": did,
	})
}

// GetKnowledgeBaseList
//
//	@Summary		GetKnowledgeBaseList
//	@Description	GetKnowledgeBaseList
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	domain.PWResponse{data=[]domain.KnowledgeBaseListItem}
//	@Router			/api/v1/knowledge_base/list [get]
func (h *KnowledgeBaseHandler) GetKnowledgeBaseList(c echo.Context) error {

	knowledgeBases, err := h.usecase.GetKnowledgeBaseListByUserId(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "failed to get knowledge base list", err)
	}

	return h.NewResponseWithData(c, knowledgeBases)
}

// UpdateKnowledgeBase
//
//	@Summary		UpdateKnowledgeBase
//	@Description	UpdateKnowledgeBase
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Param			body	body		domain.UpdateKnowledgeBaseReq	true	"UpdateKnowledgeBase Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/knowledge_base/detail [put]
func (h *KnowledgeBaseHandler) UpdateKnowledgeBase(c echo.Context) error {
	var req domain.UpdateKnowledgeBaseReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	err := h.usecase.UpdateKnowledgeBase(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrPortHostAlreadyExists) {
			return h.NewResponseWithError(c, "端口或域名已被其他知识库占用", nil)
		}
		if errors.Is(err, domain.ErrSyncCaddyConfigFailed) {
			return h.NewResponseWithError(c, "端口可能已被其他程序占用，请检查", nil)
		}
		return h.NewResponseWithError(c, "failed to update knowledge base", err)
	}

	return h.NewResponseWithData(c, nil)
}

// GetKnowledgeBaseDetail
//
//	@Summary		GetKnowledgeBaseDetail
//	@Description	GetKnowledgeBaseDetail
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			id	query		string	true	"Knowledge Base ID"
//	@Success		200	{object}	domain.PWResponse{data=domain.KnowledgeBaseDetail}
//	@Router			/api/v1/knowledge_base/detail [get]
func (h *KnowledgeBaseHandler) GetKnowledgeBaseDetail(c echo.Context) error {
	kbID := c.QueryParam("id")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb id is required", nil)
	}

	kb, err := h.usecase.GetKnowledgeBase(c.Request().Context(), kbID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get knowledge base detail", err)
	}

	perm, err := h.usecase.GetKnowledgeBasePerm(c.Request().Context(), kbID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get knowledge base permission", err)
	}

	if perm != consts.UserKBPermissionFullControl {
		kb.AccessSettings.PrivateKey = ""
		kb.AccessSettings.PublicKey = ""
	}

	return h.NewResponseWithData(c, &domain.KnowledgeBaseDetail{
		ID:             kb.ID,
		Name:           kb.Name,
		DatasetID:      kb.DatasetID,
		Perm:           perm,
		AccessSettings: kb.AccessSettings,
		CreatedAt:      kb.CreatedAt,
		UpdatedAt:      kb.UpdatedAt,
	})
}

// GetPromptSettings
//
//	@Summary		get prompt settings
//	@Description	get prompt settings
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			kb_id	query		string	true	"kb id"
//	@Success		200		{object}	domain.PWResponse{data=domain.Prompt}
//	@Router			/api/pro/v1/prompt [get]
func (h *KnowledgeBaseHandler) GetPromptSettings(c echo.Context) error {
	kbID := c.QueryParam("kb_id")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb id is required", nil)
	}

	promptSetting, err := h.llmUsecase.GetPromptSettings(c.Request().Context(), kbID)
	if err != nil {
		return h.NewResponseWithError(c, "get prompt settings failed", err)
	}

	return h.NewResponseWithData(c, promptSetting)
}

// UpdatePromptSettings
//
//	@Summary		update prompt settings
//	@Description	update prompt settings
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			prompt	body		domain.UpdatePromptReq	true	"prompt settings"
//	@Success		200		{object}	domain.PWResponse{data=domain.Prompt}
//	@Router			/api/pro/v1/prompt [post]
func (h *KnowledgeBaseHandler) UpdatePromptSettings(c echo.Context) error {
	ctx := c.Request().Context()
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return h.NewResponseWithError(c, "authInfo not found in context", nil)
	}

	var req domain.UpdatePromptReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if req.Content == nil && req.SummaryContent == nil {
		return h.NewResponseWithError(c, "content or summary_content is required", nil)
	}

	promptSetting, err := h.llmUsecase.UpdatePromptSettings(ctx, &req, authInfo.UserId)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPromptTemplateVariable) {
			return h.NewResponseWithError(c, err.Error(), nil)
		}
		return h.NewResponseWithError(c, "update prompt settings failed", err)
	}

	return h.NewResponseWithData(c, promptSetting)
}

// GetPromptVersionList
//
//	@Summary		get prompt version list
//	@Description	get prompt version list
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			kb_id	query		string	true	"kb id"
//	@Success		200		{object}	domain.PWResponse{data=[]domain.PromptVersionListItem}
//	@Router			/api/pro/v1/prompt/version/list [get]
func (h *KnowledgeBaseHandler) GetPromptVersionList(c echo.Context) error {
	var req domain.PromptVersionListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	items, err := h.llmUsecase.GetPromptVersionList(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, "get prompt version list failed", err)
	}

	return h.NewResponseWithData(c, items)
}

// GetPromptVersionDetail
//
//	@Summary		get prompt version detail
//	@Description	get prompt version detail
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			kb_id	query		string	true	"kb id"
//	@Param			version	query		int		true	"version"
//	@Success		200		{object}	domain.PWResponse{data=domain.PromptVersionDetail}
//	@Router			/api/pro/v1/prompt/version/detail [get]
func (h *KnowledgeBaseHandler) GetPromptVersionDetail(c echo.Context) error {
	var req domain.PromptVersionDetailReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	item, err := h.llmUsecase.GetPromptVersionDetail(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, domain.ErrPromptVersionNotFound) {
			return h.NewResponseWithError(c, "prompt version not found", nil)
		}
		return h.NewResponseWithError(c, "get prompt version detail failed", err)
	}

	return h.NewResponseWithData(c, item)
}

// RollbackPromptVersion
//
//	@Summary		rollback prompt version
//	@Description	rollback prompt version
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			prompt	body		domain.RollbackPromptVersionReq	true	"rollback prompt version request"
//	@Success		200		{object}	domain.PWResponse{data=domain.RollbackPromptVersionResp}
//	@Router			/api/pro/v1/prompt/version/rollback [post]
func (h *KnowledgeBaseHandler) RollbackPromptVersion(c echo.Context) error {
	ctx := c.Request().Context()
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return h.NewResponseWithError(c, "authInfo not found in context", nil)
	}

	var req domain.RollbackPromptVersionReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	resp, err := h.llmUsecase.RollbackPromptVersion(ctx, &req, authInfo.UserId)
	if err != nil {
		if errors.Is(err, domain.ErrPromptVersionNotFound) {
			return h.NewResponseWithError(c, "prompt version not found", nil)
		}
		return h.NewResponseWithError(c, "rollback prompt version failed", err)
	}

	return h.NewResponseWithData(c, resp)
}

// CreateAPIToken
//
//	@Summary		创建 APIToken
//	@Description	创建 APIToken
//	@Tags			ApiToken
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			param	body		domain.CreateAPITokenReq	true	"Create API Token Request"
//	@Success		200		{object}	domain.PWResponse{data=domain.APITokenListItem}
//	@Router			/api/pro/v1/token/create [post]
func (h *KnowledgeBaseHandler) CreateAPIToken(c echo.Context) error {
	var req domain.CreateAPITokenReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	apiToken, err := h.usecase.CreateAPIToken(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, "create api token failed", err)
	}

	return h.NewResponseWithData(c, apiToken)
}

// GetAPITokenList
//
//	@Summary		获取API Token列表
//	@Description	获取当前用户的所有API Token列表，需要full_control权限
//	@Tags			ApiToken
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			param	query		domain.APITokenListReq	true	"List API Token Request"
//	@Success		200		{object}	domain.PWResponse{data=[]domain.APITokenListItem}
//	@Router			/api/pro/v1/token/list [get]
func (h *KnowledgeBaseHandler) GetAPITokenList(c echo.Context) error {
	var req domain.APITokenListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	apiTokenList, err := h.usecase.ListAPIToken(c.Request().Context(), req)
	if err != nil {
		return h.NewResponseWithError(c, "get api token list failed", err)
	}

	return h.NewResponseWithData(c, apiTokenList)
}

// UpdateAPIToken
//
//	@Summary		更新API Token
//	@Description	更新API Token的名称和权限，需要full_control权限
//	@Tags			ApiToken
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			param	body		domain.UpdateAPITokenReq	true	"Update API Token Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/pro/v1/token/update [patch]
func (h *KnowledgeBaseHandler) UpdateAPIToken(c echo.Context) error {
	var req domain.UpdateAPITokenReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if !req.HasUpdates() {
		return h.NewResponseWithError(c, "name, permission, rate_limit_per_minute or daily_quota is required", nil)
	}

	if err := h.usecase.UpdateAPIToken(c.Request().Context(), &req); err != nil {
		return h.NewResponseWithError(c, "update api token failed", err)
	}

	return h.NewResponseWithData(c, nil)
}

// DeleteAPIToken
//
//	@Summary		删除API Token
//	@Description	删除指定的API Token，需要full_control权限
//	@Tags			ApiToken
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			param	query		domain.DeleteAPITokenReq	true	"Delete API Token Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/pro/v1/token/delete [delete]
func (h *KnowledgeBaseHandler) DeleteAPIToken(c echo.Context) error {
	var req domain.DeleteAPITokenReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	if err := h.usecase.DeleteAPIToken(c.Request().Context(), req); err != nil {
		return h.NewResponseWithError(c, "delete api token failed", err)
	}

	return h.NewResponseWithData(c, nil)
}

// DeleteKnowledgeBase
//
//	@Summary		DeleteKnowledgeBase
//	@Description	DeleteKnowledgeBase
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Param			id	query		string	true	"Knowledge Base ID"
//	@Success		200	{object}	domain.Response
//	@Router			/api/v1/knowledge_base/detail [delete]
func (h *KnowledgeBaseHandler) DeleteKnowledgeBase(c echo.Context) error {
	kbID := c.QueryParam("id")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb id is required", nil)
	}

	err := h.usecase.DeleteKnowledgeBase(c.Request().Context(), kbID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to delete knowledge base", err)
	}

	return h.NewResponseWithData(c, nil)
}

// CreateKBRelease
//
//	@Summary		CreateKBRelease
//	@Description	CreateKBRelease
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Param			body	body		domain.CreateKBReleaseReq	true	"CreateKBRelease Request"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/knowledge_base/release [post]
func (h *KnowledgeBaseHandler) CreateKBRelease(c echo.Context) error {
	ctx := c.Request().Context()
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return h.NewResponseWithError(c, "authInfo not found in context", nil)
	}

	req := &domain.CreateKBReleaseReq{}
	if err := c.Bind(req); err != nil {
		return h.NewResponseWithError(c, "request body is invalid", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "validate request body failed", err)
	}

	id, err := h.usecase.CreateKBRelease(ctx, req, authInfo.UserId)
	if err != nil {
		return h.NewResponseWithError(c, "create kb release failed", err)
	}

	return h.NewResponseWithData(c, map[string]any{
		"id": id,
	})
}

// GetKBReleaseList
//
//	@Summary		GetKBReleaseList
//	@Description	GetKBReleaseList
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Param			kb_id	query		string	true	"Knowledge Base ID"
//	@Success		200		{object}	domain.PWResponse{data=domain.GetKBReleaseListResp}
//	@Router			/api/v1/knowledge_base/release/list [get]
func (h *KnowledgeBaseHandler) GetKBReleaseList(c echo.Context) error {
	var req domain.GetKBReleaseListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "request params is invalid", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "validate request params failed", err)
	}

	resp, err := h.usecase.GetKBReleaseList(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, "get kb release list failed", err)
	}

	return h.NewResponseWithData(c, resp)
}
