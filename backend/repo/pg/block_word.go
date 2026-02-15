package pg

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
	"gorm.io/gorm"
)

type BlockWordRepo struct {
	db     *pg.DB
	logger *log.Logger
}

type BlockWords struct {
	Words []string
}

func NewBlockWordRepo(db *pg.DB, logger *log.Logger) *BlockWordRepo {
	return &BlockWordRepo{
		db:     db,
		logger: logger,
	}
}

func (r *BlockWordRepo) GetBlockWords(ctx context.Context, kbID string) ([]string, error) {
	var setting domain.Setting
	var words BlockWords
	err := r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingBlockWords).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(setting.Value, &words); err != nil {
		return nil, err
	}
	return words.Words, nil
}

func (r *BlockWordRepo) UpsertBlockWords(ctx context.Context, kbID string, blockWords []string) error {
	value, err := json.Marshal(BlockWords{Words: blockWords})
	if err != nil {
		return err
	}

	var setting domain.Setting
	err = r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingBlockWords).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Table("settings").Create(&domain.Setting{
				KBID:  kbID,
				Key:   domain.SettingBlockWords,
				Value: value,
			}).Error
		}
		return err
	}

	return r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingBlockWords).
		Updates(map[string]any{
			"value":      value,
			"updated_at": time.Now(),
		}).Error
}
