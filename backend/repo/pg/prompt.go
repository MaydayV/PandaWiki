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

func (r *PromptRepo) UpsertPrompt(ctx context.Context, kbID string, prompt promptJson) error {
	value, err := json.Marshal(prompt)
	if err != nil {
		return err
	}

	var setting domain.Setting
	err = r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeySystemPrompt).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Table("settings").Create(&domain.Setting{
				KBID:  kbID,
				Key:   domain.SettingKeySystemPrompt,
				Value: value,
			}).Error
		}
		return err
	}

	return r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingKeySystemPrompt).
		Updates(map[string]any{
			"value":      value,
			"updated_at": time.Now(),
		}).Error
}
