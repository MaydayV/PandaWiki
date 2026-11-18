package domain

import (
	"time"
)

const (
	RagJobStatusPending    = "pending"
	RagJobStatusProcessing = "processing"
	RagJobStatusDone       = "done"
	RagJobStatusFailed     = "failed"
)

type RagJob struct {
	// default 让 GORM 在零值时跳过该列，由数据库 gen_random_uuid() 生成，
	// 否则零值 string 主键会被显式插入为 ''，撞 UUID 类型报错
	ID          string     `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	Payload     []byte     `gorm:"column:payload;type:jsonb"`
	Status      string     `gorm:"column:status"`
	Attempts    int        `gorm:"column:attempts"`
	MaxAttempts int        `gorm:"column:max_attempts"`
	LastError   string     `gorm:"column:last_error"`
	LockedAt    *time.Time `gorm:"column:locked_at"`
	LockedBy    string     `gorm:"column:locked_by"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (RagJob) TableName() string {
	return "rag_jobs"
}
