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
	ID          string     `gorm:"column:id;primaryKey"`
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
