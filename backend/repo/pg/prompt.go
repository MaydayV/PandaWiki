package pg

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

type PromptRepo struct {
	db     *pg.DB
	logger *log.Logger
}

type promptJson struct {
	Content        string `json:"content"`
	SummaryContent string `json:"summary_content"`
}

func NewPromptRepo(db *pg.DB, logger *log.Logger) *PromptRepo {
	return &PromptRepo{
		db:     db,
		logger: logger,
	}
}

func (r *PromptRepo) GetPrompt(ctx context.Context, kbID string) (string, error) {
	var setting domain.Setting
	var prompt promptJson
	err := r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeySystemPrompt).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	if err := json.Unmarshal(setting.Value, &prompt); err != nil {
		return "", err
	}
	return prompt.Content, nil
}

func (r *PromptRepo) GetSummaryPrompt(ctx context.Context, kbID string) (string, error) {
	var setting domain.Setting
	var prompt promptJson
	err := r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeySystemPrompt).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.SystemDefaultSummaryPrompt, nil
		}
		return "", err
	}
	if err := json.Unmarshal(setting.Value, &prompt); err != nil {
		return "", err
	}
	if strings.TrimSpace(prompt.SummaryContent) == "" {
		prompt.SummaryContent = domain.SystemDefaultSummaryPrompt
	}
	return prompt.SummaryContent, nil
}

func (r *PromptRepo) GetPromptSettings(ctx context.Context, kbID string) (promptJson, bool, error) {
	var setting domain.Setting
	var prompt promptJson
	err := r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeySystemPrompt).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return prompt, false, nil
		}
		return prompt, false, err
	}
	if err := json.Unmarshal(setting.Value, &prompt); err != nil {
		return prompt, false, err
	}
	return prompt, true, nil
}

func (r *PromptRepo) GetPromptVersionList(ctx context.Context, kbID string) ([]*domain.PromptVersion, error) {
	items := make([]*domain.PromptVersion, 0)
	if err := r.db.WithContext(ctx).
		Table("prompt_versions").
		Where("kb_id = ?", kbID).
		Order("version DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PromptRepo) GetPromptVersionDetail(ctx context.Context, kbID string, version int) (*domain.PromptVersion, error) {
	item := &domain.PromptVersion{}
	if err := r.db.WithContext(ctx).
		Table("prompt_versions").
		Where("kb_id = ?", kbID).
		Where("version = ?", version).
		First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return item, nil
}

func (r *PromptRepo) UpsertPrompt(ctx context.Context, kbID string, prompt promptJson) error {
	return r.upsertPromptWithDB(ctx, r.db.WithContext(ctx), kbID, prompt)
}

func (r *PromptRepo) UpsertPromptWithVersion(ctx context.Context, kbID, content, summaryContent, operatorUserID string) (int, error) {
	if strings.TrimSpace(operatorUserID) == "" {
		return 0, errors.New("operator user id is required")
	}

	nextVersion := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		prompt := promptJson{
			Content:        content,
			SummaryContent: summaryContent,
		}
		if err := r.upsertPromptWithDB(ctx, tx, kbID, prompt); err != nil {
			return err
		}

		if err := tx.Exec("LOCK TABLE prompt_versions IN EXCLUSIVE MODE").Error; err != nil {
			return err
		}

		var latestVersion int
		if err := tx.Table("prompt_versions").
			Where("kb_id = ?", kbID).
			Select("COALESCE(MAX(version), 0)").
			Scan(&latestVersion).Error; err != nil {
			return err
		}
		nextVersion = latestVersion + 1

		return tx.Table("prompt_versions").Create(&domain.PromptVersion{
			KBID:           kbID,
			Version:        nextVersion,
			Content:        content,
			SummaryContent: summaryContent,
			OperatorUserID: operatorUserID,
		}).Error
	})
	if err != nil {
		return 0, err
	}
	return nextVersion, nil
}

func (r *PromptRepo) upsertPromptWithDB(ctx context.Context, db *gorm.DB, kbID string, prompt promptJson) error {
	value, err := json.Marshal(prompt)
	if err != nil {
		return err
	}

	var setting domain.Setting
	err = db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeySystemPrompt).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return db.WithContext(ctx).Table("settings").Create(&domain.Setting{
				KBID:  kbID,
				Key:   domain.SettingKeySystemPrompt,
				Value: value,
			}).Error
		}
		return err
	}

	return db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeySystemPrompt).
		Updates(map[string]any{
			"value":      value,
			"updated_at": time.Now(),
		}).Error
}
