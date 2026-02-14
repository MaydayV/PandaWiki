package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

type APICallAuditRepo struct {
	db     *pg.DB
	logger *log.Logger
}

func NewAPICallAuditRepo(db *pg.DB, logger *log.Logger) *APICallAuditRepo {
	return &APICallAuditRepo{
		db:     db,
		logger: logger.WithModule("repo.pg.api_call_audit"),
	}
}

func (r *APICallAuditRepo) Create(ctx context.Context, audit *domain.APICallAudit) error {
	if audit == nil {
		return fmt.Errorf("api call audit is nil")
	}
	if err := r.db.WithContext(ctx).Create(audit).Error; err != nil {
		return fmt.Errorf("create api call audit failed: %w", err)
	}
	return nil
}

func (r *APICallAuditRepo) CountByTokenSince(ctx context.Context, apiTokenID, endpoint string, since time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&domain.APICallAudit{}).
		Where("api_token_id = ? AND created_at >= ?", apiTokenID, since)
	if endpoint != "" {
		query = query.Where("endpoint = ?", endpoint)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count api call audits failed: %w", err)
	}
	return count, nil
}
