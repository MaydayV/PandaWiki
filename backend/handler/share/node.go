package share

import (
	"net/http"

	"github.com/labstack/echo/v4"

	shareV1 "github.com/chaitin/panda-wiki/api/share/v1"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/usecase"
)

type ShareNodeHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.NodeUsecase
}

func NewShareNodeHandler(
	baseHandler *handler.BaseHandler,
	e *echo.Echo,
	usecase *usecase.NodeUsecase,
	logger *log.Logger,
) *ShareNodeHandler {
	h := &ShareNodeHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.share.node"),
		usecase:     usecase,
	}

	group := e.Group("share/v1/node",
		h.ShareAuthMiddleware.Authorize,
	)
	group.GET("/list", h.GetNodeList)
	group.GET("/detail", h.GetNodeDetail)

	contributeGroup := e.Group("share/pro/v1/contribute",
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Access-Control-Allow-Origin", "*")
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept")
				if c.Request().Method == "OPTIONS" {
					return c.NoContent(http.StatusOK)
				}
				return next(c)
			}
		},
		h.ShareAuthMiddleware.Authorize,
	)
	contributeGroup.POST("/submit", h.SubmitContribute)

	return h
}

// GetNodeList
//
//	@Summary		GetNodeList
//	@Description	GetNodeList
//	@Tags			share_node
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string	true	"kb id"
//	@Success		200		{object}	domain.Response
//	@Router			/share/v1/node/list [get]
func (h *ShareNodeHandler) GetNodeList(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	nodes, err := h.usecase.GetNodeReleaseListByKBID(c.Request().Context(), kbID, domain.GetAuthID(c))
	if err != nil {
		return h.NewResponseWithError(c, "failed to get node list", err)
	}

	return h.NewResponseWithData(c, nodes)
}

// GetNodeDetail
//
//	@Summary		GetNodeDetail
//	@Description	GetNodeDetail
//	@Tags			share_node
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string	true	"kb id"
//	@Param			id		query		string	true	"node id"
//	@Param			format	query		string	true	"format"
//	@Success		200		{object}	domain.Response{data=v1.ShareNodeDetailResp}
//	@Router			/share/v1/node/detail [get]
func (h *ShareNodeHandler) GetNodeDetail(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	req := &shareV1.GetShareNodeDetailReq{}
	if err := c.Bind(req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}

	errCode := h.usecase.ValidateNodePerm(c.Request().Context(), kbID, req.ID, domain.GetAuthID(c))
	if errCode != nil {
		return h.NewResponseWithErrCode(c, *errCode)
	}

	node, err := h.usecase.GetNodeReleaseDetailByKBIDAndIDWithLanguage(
		c.Request().Context(),
		kbID,
		req.ID,
		req.Format,
		req.Lang,
		req.Language,
		c.Request().Header.Get("Accept-Language"),
	)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get node detail", err)
	}

	// If the node is a folder, return the list of child nodes
	if node.Type == domain.NodeTypeFolder {
		childNodes, err := h.usecase.GetNodeReleaseListByParentID(c.Request().Context(), kbID, req.ID, domain.GetAuthID(c))
		if err != nil {
			return h.NewResponseWithError(c, "failed to get child nodes", err)
		}
		node.List = childNodes
	}

	return h.NewResponseWithData(c, node)
}

func (h *ShareNodeHandler) SubmitContribute(c echo.Context) error {
	ctx := c.Request().Context()

	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	var req domain.SubmitContributeReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}
	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "validate request body failed", err)
	}

	if !h.Captcha.ValidateToken(ctx, req.CaptchaToken) {
		return h.NewResponseWithError(c, "failed to validate captcha token", nil)
	}

	contributeID, err := h.usecase.SubmitContribute(ctx, kbID, c.RealIP(), domain.GetAuthID(c), &req)
	if err != nil {
		return h.NewResponseWithError(c, "submit contribute failed", err)
	}

	return h.NewResponseWithData(c, domain.SubmitContributeResp{
		ID: contributeID,
	})
}
