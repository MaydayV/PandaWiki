package fns

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
)

type MigrationCreateFirstNavs struct {
	Name   string
	logger *log.Logger
}

func NewMigrationCreateFirstNavs(logger *log.Logger) *MigrationCreateFirstNavs {
	return &MigrationCreateFirstNavs{
		Name:   "0005_create_first_navs",
		logger: logger,
	}
}

func (m *MigrationCreateFirstNavs) Execute(tx *gorm.DB) error {

	var kbs []*domain.KnowledgeBaseListItem
	if err := tx.Model(&domain.KnowledgeBase{}).
		Order("created_at ASC").
		Find(&kbs).Error; err != nil {
		return err
	}

	for _, kb := range kbs {

		nav := &domain.Nav{
			ID:   uuid.New().String(),
			Name: kb.Name,
			KbID: kb.ID,
		}

		if err := tx.Model(&domain.Nav{}).Create(nav).Error; err != nil {
			return err
		}

		if err := tx.Model(&domain.Node{}).
			Where("kb_id = ?", kb.ID).
			Update("nav_id", nav.ID).Error; err != nil {
			return err
		}

		var release domain.KBRelease
		err := tx.Model(&domain.KBRelease{}).
			Where("kb_id = ?", kb.ID).
			Order("created_at DESC").
			First(&release).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}

		navRelease := &domain.NavRelease{
			ID:        uuid.New().String(),
			NavID:     nav.ID,
			ReleaseID: release.ID,
			KbID:      release.KBID,
			Name:      nav.Name,
			Position:  nav.Position,
			CreatedAt: time.Now(),
		}
		if err := tx.Model(&domain.NavRelease{}).Create(navRelease).Error; err != nil {
			return err
		}

		if err := tx.Model(&domain.KBReleaseNodeRelease{}).
			Where("kb_id = ? AND release_id = ?", kb.ID, release.ID).
			Update("nav_id", nav.ID).Error; err != nil {
			return err
		}
	}

	return nil
}
