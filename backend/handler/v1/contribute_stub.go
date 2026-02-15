package v1

import (
	"errors"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
)

func (h *KnowledgeBaseHandler) GetContributeList(c echo.Context) error {
	var req domain.ContributeListReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	resp, err := h.usecase.GetContributeList(c.Request().Context(), &req)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get contribute list", err)
	}

	return h.NewResponseWithData(c, resp)
}

func (h *KnowledgeBaseHandler) GetContributeDetail(c echo.Context) error {
	var req domain.ContributeDetailReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	detail, err := h.usecase.GetContributeDetail(c.Request().Context(), &req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return h.NewResponseWithError(c, "contribute not found", nil)
		}
		return h.NewResponseWithError(c, "failed to get contribute detail", err)
	}
	return h.NewResponseWithData(c, detail)
}

func (h *KnowledgeBaseHandler) AuditContribute(c echo.Context) error {
	ctx := c.Request().Context()
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return h.NewResponseWithError(c, "authInfo not found in context", nil)
	}

	var req domain.ContributeAuditReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	resp, err := h.usecase.AuditContribute(ctx, &req, authInfo.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return h.NewResponseWithError(c, "contribute not found", nil)
		}
		return h.NewResponseWithError(c, "failed to audit contribute", err)
	}
	return h.NewResponseWithData(c, resp)
}
