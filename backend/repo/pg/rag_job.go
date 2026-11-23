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
	// 分批插入：rag_jobs 10 列，单语句最多约 6553 行（PG 65535 参数上限），
	// 全站迁移/大 KB 全量发布一次性入队会直接报 too many SQL variables
	if err := r.db.WithContext(ctx).CreateInBatches(&jobs, 500).Error; err != nil {
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
	// 失败重试按 attempts 指数退避（10s/40s/90s/...，上限 5 分钟），避免上游
	// embedding 故障时 5 次重试在数秒内烧完转 failed；新任务（attempts=0）立即领取
	err := r.db.WithContext(ctx).Raw(`
WITH picked AS (
    SELECT id
    FROM rag_jobs
    WHERE status = 'pending'
      AND (
        attempts = 0
        OR locked_at IS NULL
        OR locked_at <= NOW() - (LEAST(attempts * attempts * 10, 300) || ' seconds')::interval
      )
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

// RequeueStale 将 processing 超过 timeout 的僵尸任务重回 pending（仅限未耗尽
// max_attempts 的任务）；已耗尽重试次数的超时任务标记为 failed，避免毒丸任务无限循环。
func (r *RagJobRepository) RequeueStale(ctx context.Context, timeout time.Duration) (int64, error) {
	intervalArg := fmt.Sprintf("%d seconds", int(timeout.Seconds()))
	if err := r.db.WithContext(ctx).Exec(`
UPDATE rag_jobs
SET status = 'failed',
    last_error = 'worker timed out after max attempts',
    locked_at = NULL,
    locked_by = '',
    updated_at = NOW()
WHERE status = 'processing'
  AND attempts >= max_attempts
  AND locked_at <= NOW() - ?::interval
`, intervalArg).Error; err != nil {
		return 0, fmt.Errorf("fail stale exhausted rag jobs: %w", err)
	}

	result := r.db.WithContext(ctx).Exec(`
UPDATE rag_jobs
SET status = 'pending',
    locked_at = NULL,
    locked_by = '',
    updated_at = NOW()
WHERE status = 'processing'
  AND attempts < max_attempts
  AND locked_at <= NOW() - ?::interval
`, intervalArg)
	if result.Error != nil {
		return 0, fmt.Errorf("requeue stale rag jobs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// PurgeFailed 清理最终失败且超过保留期的任务记录，避免表无限增长
func (r *RagJobRepository) PurgeFailed(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND updated_at < ?", domain.RagJobStatusFailed, before).
		Delete(&domain.RagJob{})
	if result.Error != nil {
		return 0, fmt.Errorf("purge failed rag jobs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *RagJobRepository) markFailed(ctx context.Context, jobID string, attempts, maxAttempts int, jobErr error) error {
	status := domain.RagJobStatusPending
	if attempts >= maxAttempts {
		status = domain.RagJobStatusFailed
	}
	updates := map[string]interface{}{
		"status":     status,
		"last_error": jobErr.Error(),
		"locked_by":  "",
		"updated_at": gorm.Expr("NOW()"),
	}
	if status == domain.RagJobStatusPending {
		// 保留 locked_at 作为重试退避的起点（Claim 按它计算等待窗口）
		updates["locked_at"] = gorm.Expr("NOW()")
	} else {
		updates["locked_at"] = nil
	}
	return r.db.WithContext(ctx).Model(&domain.RagJob{}).
		Where("id = ?", jobID).
		Updates(updates).Error
}
