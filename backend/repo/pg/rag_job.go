package pg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

var ErrNoRagJob = errors.New("no rag job available")

type RagJobRepository struct {
	db     *pg.DB
	logger *log.Logger
}

func NewRagJobRepository(db *pg.DB, logger *log.Logger) *RagJobRepository {
	return &RagJobRepository{
		db:     db,
		logger: logger.WithModule("repo.pg.rag_job"),
	}
}

func (r *RagJobRepository) Enqueue(ctx context.Context, requests []*domain.NodeReleaseVectorRequest) error {
	if len(requests) == 0 {
		return nil
	}
	jobs := make([]domain.RagJob, 0, len(requests))
	for _, req := range requests {
		payload, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal rag job payload: %w", err)
		}
		jobs = append(jobs, domain.RagJob{
			Payload:     payload,
			Status:      domain.RagJobStatusPending,
			MaxAttempts: 5,
		})
	}
	if err := r.db.WithContext(ctx).Create(&jobs).Error; err != nil {
		return fmt.Errorf("create rag jobs: %w", err)
	}
	return nil
}

type claimedRagJob struct {
	ID          string
	Payload     []byte
	Attempts    int
	MaxAttempts int
}

func (r *RagJobRepository) Claim(ctx context.Context, workerID string) (*claimedRagJob, *domain.NodeReleaseVectorRequest, error) {
	var row claimedRagJob
	err := r.db.WithContext(ctx).Raw(`
WITH picked AS (
    SELECT id
    FROM rag_jobs
    WHERE status = 'pending'
    ORDER BY created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE rag_jobs AS j
SET status = 'processing',
    locked_at = NOW(),
    locked_by = ?,
    attempts = j.attempts + 1,
    updated_at = NOW()
FROM picked
WHERE j.id = picked.id
RETURNING j.id, j.payload, j.attempts, j.max_attempts
`, workerID).Scan(&row).Error
	if err != nil {
		return nil, nil, fmt.Errorf("claim rag job: %w", err)
	}
	if row.ID == "" {
		return nil, nil, ErrNoRagJob
	}

	var req domain.NodeReleaseVectorRequest
	if err := json.Unmarshal(row.Payload, &req); err != nil {
		if markErr := r.markFailed(ctx, row.ID, row.Attempts, row.MaxAttempts, err); markErr != nil {
			r.logger.Error("mark malformed rag job failed", log.Error(markErr))
		}
		return nil, nil, fmt.Errorf("unmarshal rag job payload: %w", err)
	}
	return &row, &req, nil
}

func (r *RagJobRepository) MarkDone(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).Model(&domain.RagJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":     domain.RagJobStatusDone,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *RagJobRepository) MarkFailed(ctx context.Context, jobID string, attempts, maxAttempts int, jobErr error) error {
	return r.markFailed(ctx, jobID, attempts, maxAttempts, jobErr)
}

func (r *RagJobRepository) PurgeDone(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND updated_at < ?", domain.RagJobStatusDone, before).
		Delete(&domain.RagJob{})
	if result.Error != nil {
		return 0, fmt.Errorf("purge done rag jobs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *RagJobRepository) markFailed(ctx context.Context, jobID string, attempts, maxAttempts int, jobErr error) error {
	status := domain.RagJobStatusPending
	if attempts >= maxAttempts {
		status = domain.RagJobStatusFailed
	}
	return r.db.WithContext(ctx).Model(&domain.RagJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":     status,
			"last_error": jobErr.Error(),
			"locked_at":  nil,
			"locked_by":  "",
			"updated_at": gorm.Expr("NOW()"),
		}).Error
}
