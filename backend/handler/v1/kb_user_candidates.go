package v1

import (
	"strings"

	"github.com/labstack/echo/v4"

	_ "github.com/chaitin/panda-wiki/api/user/v1"
)

// KBUserCandidates
//
//	@Summary		KBUserCandidates
//	@Description	List candidate users that can be invited to the knowledge base
//	@Tags			knowledge_base
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			kb_id	query		string	true	"Knowledge Base ID"
//	@Success		200		{object}	domain.PWResponse{data=v1.UserListResp}
//	@Router			/api/v1/knowledge_base/user/candidates [get]
func (h *KnowledgeBaseHandler) KBUserCandidates(c echo.Context) error {
	if strings.TrimSpace(c.QueryParam("kb_id")) == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}

	resp, err := h.usecase.GetKBUserCandidates(c.Request().Context())
	if err != nil {
		return h.NewResponseWithError(c, "get kb user candidates failed", err)
	}

	return h.NewResponseWithData(c, resp)
}
