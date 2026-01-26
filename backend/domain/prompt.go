package domain

type Prompt struct {
	Content        string `json:"content"`
	SummaryContent string `json:"summary_content"`
}

type UpdatePromptReq struct {
	KBID           string  `json:"kb_id" validate:"required"`
	Content        *string `json:"content,omitempty"`
	SummaryContent *string `json:"summary_content,omitempty"`
}
