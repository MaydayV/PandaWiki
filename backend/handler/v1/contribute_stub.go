package v1

import "github.com/labstack/echo/v4"

type contributeListResp struct {
	List  []map[string]any `json:"list"`
	Total int64            `json:"total"`
}

type contributeAuditResp struct {
	Message string `json:"message"`
}

// GetContributeList keeps admin contribution page functional even when
// enterprise contribute workflow is not wired in this repo.
func (h *KnowledgeBaseHandler) GetContributeList(c echo.Context) error {
	return h.NewResponseWithData(c, contributeListResp{
		List:  make([]map[string]any, 0),
		Total: 0,
	})
}

func (h *KnowledgeBaseHandler) GetContributeDetail(c echo.Context) error {
	return h.NewResponseWithData(c, map[string]any{
		"id":    c.QueryParam("id"),
		"kb_id": c.QueryParam("kb_id"),
		"meta": map[string]any{
			"content_type": "md",
		},
	})
}

func (h *KnowledgeBaseHandler) AuditContribute(c echo.Context) error {
	return h.NewResponseWithData(c, contributeAuditResp{
		Message: "success",
	})
}
