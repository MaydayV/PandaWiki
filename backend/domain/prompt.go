package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidPromptTemplateVariable = errors.New("invalid prompt template variable")
	ErrPromptVersionNotFound         = errors.New("prompt version not found")
)

var PromptTemplateVariableWhitelist = []string{
	".CurrentDate",
	".Question",
	".Documents",
}

type Prompt struct {
	Content        string `json:"content"`
	SummaryContent string `json:"summary_content"`
}

type UpdatePromptReq struct {
	KBID           string  `json:"kb_id" validate:"required"`
	Content        *string `json:"content,omitempty"`
	SummaryContent *string `json:"summary_content,omitempty"`
}

type PromptVersion struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	KBID           string    `json:"kb_id" gorm:"column:kb_id;type:text;not null"`
	Version        int       `json:"version" gorm:"column:version;not null"`
	Content        string    `json:"content" gorm:"column:content;type:text;not null"`
	SummaryContent string    `json:"summary_content" gorm:"column:summary_content;type:text;not null"`
	OperatorUserID string    `json:"operator_user_id" gorm:"column:operator_user_id;type:text;not null"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:created_at"`
}

func (PromptVersion) TableName() string {
	return "prompt_versions"
}

type PromptVersionListReq struct {
	KBID string `json:"kb_id" query:"kb_id" validate:"required"`
}

type PromptVersionListItem struct {
	ID             int64     `json:"id"`
	Version        int       `json:"version"`
	OperatorUserID string    `json:"operator_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type PromptVersionDetailReq struct {
	KBID    string `json:"kb_id" query:"kb_id" validate:"required"`
	Version int    `json:"version" query:"version" validate:"required,gt=0"`
}

type PromptVersionDetail struct {
	ID             int64     `json:"id"`
	Version        int       `json:"version"`
	Content        string    `json:"content"`
	SummaryContent string    `json:"summary_content"`
	OperatorUserID string    `json:"operator_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type RollbackPromptVersionReq struct {
	KBID    string `json:"kb_id" validate:"required"`
	Version int    `json:"version" validate:"required,gt=0"`
}

type RollbackPromptVersionResp struct {
	Version int     `json:"version"`
	Prompt  *Prompt `json:"prompt"`
}
