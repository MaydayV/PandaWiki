package pg

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

type AppRepository struct {
	db     *pg.DB
	logger *log.Logger
}

func NewAppRepository(db *pg.DB, logger *log.Logger) *AppRepository {
	return &AppRepository{
		db:     db,
		logger: logger.WithModule("repo.pg.app"),
	}
}

func (r *AppRepository) GetAppDetail(ctx context.Context, id string) (*domain.App, error) {
	app := &domain.App{}
	if err := r.db.WithContext(ctx).
		Model(&domain.App{}).
		Where("id = ?", id).
		First(app).Error; err != nil {
		return nil, err
	}
	return app, nil
}

func (r *AppRepository) UpdateApp(ctx context.Context, id, kbId string, appRequest *domain.UpdateAppReq) error {
	updateMap := map[string]any{}
	if appRequest.Name != nil {
		updateMap["name"] = appRequest.Name
	}
	if appRequest.Settings != nil {
		updateMap["settings"] = appRequest.Settings
	}
	return r.db.WithContext(ctx).Model(&domain.App{}).Where("id = ? and kb_id = ?", id, kbId).Updates(updateMap).Error
}

func (r *AppRepository) DeleteApp(ctx context.Context, id, kbId string) error {
	return r.db.WithContext(ctx).Delete(&domain.App{}, "id = ? and kb_id = ?", id, kbId).Error
}

func (r *AppRepository) GetOrCreateAppByKBIDAndType(ctx context.Context, kbID string, appType domain.AppType) (*domain.App, error) {
	app := &domain.App{}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&domain.App{}).Where("kb_id = ? AND type = ?", kbID, appType).First(app).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create app if kb is exist
				if err := tx.Model(&domain.KnowledgeBase{}).Where("id = ?", kbID).First(&domain.KnowledgeBase{}).Error; err != nil {
					return err
				}
				app = &domain.App{
					ID:   uuid.New().String(),
					KBID: kbID,
					Type: appType,
					Settings: domain.AppSettings{
						StatsSetting: domain.StatsSetting{
							PVEnable: true,
						},
					},
				}
				return tx.Create(app).Error
			}
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return app, nil
}

// GetAppsByTypes returns all apps of a specific type
func (r *AppRepository) GetAppsByTypes(ctx context.Context, appTypes []domain.AppType) ([]*domain.App, error) {
	var apps []*domain.App
	if err := r.db.WithContext(ctx).
		Model(&domain.App{}).
		Where("type IN (?)", appTypes).
		Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

func (r *AppRepository) GetAppList(ctx context.Context, kbID string) (map[string]*domain.App, error) {
	var apps []*domain.App
	if err := r.db.WithContext(ctx).
		Model(&domain.App{}).
		Where("kb_id = ?", kbID).
		Find(&apps).Error; err != nil {
		return nil, err
	}
	return lo.SliceToMap(apps, func(app *domain.App) (string, *domain.App) {
		return app.ID, app
	}), nil
}
