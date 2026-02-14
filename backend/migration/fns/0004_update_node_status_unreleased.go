package fns

import (
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
)

type MigrationUpdateNodeStatusUnreleased struct {
	Name   string
	logger *log.Logger
}

func NewMigrationUpdateNodeStatusUnreleased(logger *log.Logger) *MigrationUpdateNodeStatusUnreleased {
	return &MigrationUpdateNodeStatusUnreleased{
		Name:   "0004_update_node_status_unreleased",
		logger: logger,
	}
}

func (m *MigrationUpdateNodeStatusUnreleased) Execute(tx *gorm.DB) error {
	// 将所有 status=1 (Draft) 且从未发布过的节点更新为 status=0 (Unreleased)
	// 判断条件：node_releases 表中不存在该 node_id 的记录
	result := tx.Model(&domain.Node{}).
		Where("status = ?", domain.NodeStatusDraft).
		Where("id NOT IN (SELECT DISTINCT node_id FROM node_releases)").
		Update("status", domain.NodeStatusUnreleased)

	if result.Error != nil {
		return result.Error
	}

	m.logger.Info("migration update node status unreleased", log.Int64("affected_rows", result.RowsAffected))
	return nil
}
