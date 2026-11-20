package domain

import (
	"context"
	"time"
)

const (
	SettingKeySystemPrompt = "system_prompt"
	SettingBlockWords      = "block_words"
	SettingGitHubSync      = "github_sync"
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
	CreateOrUpdateSetting(ctx context.Context, setting *Setting) error
	GetSetting(ctx context.Context, kbID, key string) (*Setting, error)
	UpdateSetting(ctx context.Context, kbID, key, value string) error
}

// GitHubSyncSetting git 同步模式的配置（settings 表，kb_id 隔离）。
// Token 经 EncryptSecret 加密落库（未配置 ENCRYPTION_KEY 时明文，兼容既有行为）。
type GitHubSyncSetting struct {
	KBID    string `json:"kb_id"`
	Enabled bool   `json:"enabled"`
	Repo    string `json:"repo"`   // owner/repo
	Branch  string `json:"branch"` // 默认 main
	// Token 加密存储，API 响应不回传明文；HasToken 告知前端是否已配置
	Token           string     `json:"token"`
	HasToken        bool       `json:"has_token"`
	IntervalMinutes int        `json:"interval_minutes"` // 0 = 仅手动同步
	DeleteOrphans   bool       `json:"delete_orphans"`   // 仓库删除的文件是否同步删除节点（默认 false）
	AutoPublish     bool       `json:"auto_publish"`     // 同步后自动发布（触发 RAG 向量更新）
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	LastSyncResult  string     `json:"last_sync_result,omitempty"` // 最近一次同步摘要/错误
}

// GitHubSyncResult 是一次 git 同步的执行结果
type GitHubSyncResult struct {
	Created  int      `json:"created"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Deleted  int      `json:"deleted"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
	Duration string   `json:"duration"`
}

// GitHubSyncTriggerReq 手动触发同步请求
type GitHubSyncTriggerReq struct {
	KBID string `json:"kb_id" validate:"required"`
}
