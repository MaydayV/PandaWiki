package domain

import (
	"context"
	"time"
)

const (
	SettingKeySystemPrompt = "system_prompt"
	SettingBlockWords      = "block_words"
	SettingCopyrightInfo   = "本网站由 PandaWiki 提供技术支持"
	SettingDefaultLanguage = "en-US"
)

// table: settings
type Setting struct {
	ID          int       `json:"id" gorm:"primary_key"`
	KBID        string    `json:"kb_id"`
	Key         string    `json:"key"`
	Value       []byte    `json:"value" gorm:"type:jsonb"` // JSON string
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SettingRepo interface {
	CreateSetting(ctx context.Context, setting *Setting) error
	GetSetting(ctx context.Context, kbID, key string) (*Setting, error)
	UpdateSetting(ctx context.Context, kbID, key, value string) error
}
