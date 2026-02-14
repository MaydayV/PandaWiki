package v1

import (
	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/stat/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type StatHandler struct {
	*handler.BaseHandler
	usecase *usecase.StatUseCase
	auth    middleware.AuthMiddleware
	logger  *log.Logger
}

func NewStatHandler(baseHandler *handler.BaseHandler, echo *echo.Echo, usecase *usecase.StatUseCase, logger *log.Logger, auth middleware.AuthMiddleware) *StatHandler {
	h := &StatHandler{
		BaseHandler: baseHandler,
		usecase:     usecase,
		auth:        auth,
		logger:      logger.WithModule("handler.v1.stat"),
	}

	group := echo.Group("/api/v1/stat", h.auth.Authorize, auth.ValidateKBUserPerm(consts.UserKBPermissionDataOperate))

	// 实时
	group.GET("/instant_count", h.GetInstantCount) // instant count (30min, every 1min)
	group.GET("/instant_pages", h.GetInstantPages) // instant pages (latest 10 pages)

	// 周期统计
	group.GET("/count", h.StatCount)
	group.GET("/geo_count", h.StatGeoCountReq)                              // geo (24h)
	group.GET("/conversation_distribution", h.StatConversationDistribution) // conversation (24h)
	group.GET("/hot_pages", h.StatHotPages)
	group.GET("/referer_hosts", h.StatRefererHosts)
	group.GET("/browsers", h.StatBrowsers)
	group.GET("/funnel", h.StatFunnel)
	return h
}

// StatCount 全局统计
//
//	@Summary		全局统计
//	@Description	全局统计
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatCountReq	true	"para"
//	@Success		200		{object}	domain.PWResponse{data=v1.StatCountResp}
//	@Router			/api/v1/stat/count [get]
func (h *StatHandler) StatCount(c echo.Context) error {
	var req v1.StatCountReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	if err := h.usecase.ValidateStatDay(req.Day, consts.GetLicenseEdition(c)); err != nil {
		h.logger.Error("validate stat day failed")
		return h.NewResponseWithErrCode(c, domain.ErrCodePermissionDenied)
	}

	count, err := h.usecase.GetStatCount(c.Request().Context(), req.KbID, req.Day)
	if err != nil {
		return h.NewResponseWithError(c, "get count failed", err)
	}
	return h.NewResponseWithData(c, count)
}

// StatGeoCountReq 用户地理分布
//
//	@Summary		用户地理分布
//	@Description	用户地理分布
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatGeoCountReq	true	"para"
//	@Success		200		{object}	domain.Response
//	@Router			/api/v1/stat/geo_count [get]
func (h *StatHandler) StatGeoCountReq(c echo.Context) error {
	var req v1.StatGeoCountReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	if err := h.usecase.ValidateStatDay(req.Day, consts.GetLicenseEdition(c)); err != nil {
		h.logger.Error("validate stat day failed")
		return h.NewResponseWithErrCode(c, domain.ErrCodePermissionDenied)
	}

	geoCount, err := h.usecase.GetGeoCount(c.Request().Context(), req.KbID, req.Day)
	if err != nil {
		return h.NewResponseWithError(c, "get geo count failed", err)
	}
	return h.NewResponseWithData(c, geoCount)
}

// StatConversationDistribution
//
//	@Summary		问答来源
//	@Description	问答来源
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatConversationDistributionReq	true	"para"
//	@Success		200		{object}	domain.Response{data=[]v1.StatConversationDistributionResp}
//	@Router			/api/v1/stat/conversation_distribution [get]
func (h *StatHandler) StatConversationDistribution(c echo.Context) error {
	var req v1.StatConversationDistributionReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	if err := h.usecase.ValidateStatDay(req.Day, consts.GetLicenseEdition(c)); err != nil {
		h.logger.Error("validate stat day failed")
		return h.NewResponseWithErrCode(c, domain.ErrCodePermissionDenied)
	}

	distribution, err := h.usecase.GetConversationDistribution(c.Request().Context(), req.KbID, req.Day)
	if err != nil {
		return h.NewResponseWithError(c, "get conversation distribution failed", err)
	}
	return h.NewResponseWithData(c, distribution)
}

// StatHotPages 热门文档
//
//	@Summary		热门文档
//	@Description	热门文档
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatHotPagesReq	true	"para"
//	@Success		200		{object}	domain.Response{data=[]domain.HotPage}
//	@Router			/api/v1/stat/hot_pages [get]
func (h *StatHandler) StatHotPages(c echo.Context) error {
	var req v1.StatHotPagesReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	if err := h.usecase.ValidateStatDay(req.Day, consts.GetLicenseEdition(c)); err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}

	hotPages, err := h.usecase.GetHotPages(c.Request().Context(), req.KbID, req.Day)
	if err != nil {
		return h.NewResponseWithError(c, "get hot pages failed", err)
	}
	return h.NewResponseWithData(c, hotPages)
}

// StatRefererHosts 来源域名
//
//	@Summary		来源域名
//	@Description	来源域名
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatRefererHostsReq	true	"para"
//	@Success		200		{object}	domain.Response{data=[]domain.HotRefererHost}
//	@Router			/api/v1/stat/referer_hosts [get]
func (h *StatHandler) StatRefererHosts(c echo.Context) error {
	var req v1.StatRefererHostsReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	if err := h.usecase.ValidateStatDay(req.Day, consts.GetLicenseEdition(c)); err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}

	refererHosts, err := h.usecase.GetHotRefererHosts(c.Request().Context(), req.KbID, req.Day)
	if err != nil {
		return h.NewResponseWithError(c, "get hot referer hosts failed", err)
	}
	return h.NewResponseWithData(c, refererHosts)
}

// StatBrowsers 客户端统计
//
//	@Summary		客户端统计
//	@Description	客户端统计
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatBrowsersReq	true	"para"
//	@Success		200		{object}	domain.Response{data=domain.HotBrowser}
//	@Router			/api/v1/stat/browsers [get]
func (h *StatHandler) StatBrowsers(c echo.Context) error {
	var req v1.StatBrowsersReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	if err := h.usecase.ValidateStatDay(req.Day, consts.GetLicenseEdition(c)); err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}

	hotBrowsers, err := h.usecase.GetHotBrowsers(c.Request().Context(), req.KbID, req.Day)
	if err != nil {
		return h.NewResponseWithError(c, "get hot browsers failed", err)
	}
	return h.NewResponseWithData(c, hotBrowsers)
}

// StatFunnel 漏斗和来源转化统计
//
//	@Summary		漏斗和来源转化统计
//	@Description	漏斗和来源转化统计
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatFunnelReq	true	"para"
//	@Success		200		{object}	domain.Response{data=v1.StatFunnelResp}
//	@Router			/api/v1/stat/funnel [get]
func (h *StatHandler) StatFunnel(c echo.Context) error {
	var req v1.StatFunnelReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	if err := h.usecase.ValidateStatDay(req.Day, consts.GetLicenseEdition(c)); err != nil {
		h.logger.Error("validate stat day failed")
		return h.NewResponseWithErrCode(c, domain.ErrCodePermissionDenied)
	}

	resp, err := h.usecase.GetStatFunnel(c.Request().Context(), req.KbID, req.Day)
	if err != nil {
		return h.NewResponseWithError(c, "get stat funnel failed", err)
	}
	return h.NewResponseWithData(c, resp)
}

// GetInstantCount get instant count
//
//	@Summary		GetInstantCount
//	@Description	GetInstantCount
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatInstantCountReq	true	"para"
//	@Success		200		{object}	domain.Response{data=[]domain.InstantCountResp}
//	@Router			/api/v1/stat/instant_count [get]
func (h *StatHandler) GetInstantCount(c echo.Context) error {
	var req v1.StatInstantCountReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	count, err := h.usecase.GetInstantCount(c.Request().Context(), req.KbID)
	if err != nil {
		return h.NewResponseWithError(c, "get instant count failed", err)
	}
	return h.NewResponseWithData(c, count)
}

// GetInstantPages get instant pages
//
//	@Summary		GetInstantPages
//	@Description	GetInstantPages
//	@Tags			stat
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			para	query		v1.StatInstantPagesReq	true	"para"
//	@Success		200		{object}	domain.Response{data=[]domain.InstantPageResp}
//	@Router			/api/v1/stat/instant_pages [get]
func (h *StatHandler) GetInstantPages(c echo.Context) error {
	var req v1.StatInstantPagesReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validation failed", err)
	}

	pages, err := h.usecase.GetInstantPages(c.Request().Context(), req.KbID)
	if err != nil {
		return h.NewResponseWithError(c, "get instant pages failed", err)
	}
	return h.NewResponseWithData(c, pages)
}
