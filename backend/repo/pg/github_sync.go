package pg

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

// GitHubSyncRepo 封装 git 同步模式的配置存取（settings 表，kb_id + key 隔离）
// 与同步节点的幂等映射查询。
type GitHubSyncRepo struct {
	db     *pg.DB
	logger *log.Logger
}

func NewGitHubSyncRepo(db *pg.DB, logger *log.Logger) *GitHubSyncRepo {
	return &GitHubSyncRepo{
		db:     db,
		logger: logger.WithModule("repo.pg.github_sync"),
	}
}

// GetSetting 读取指定 KB 的 git 同步配置（未配置返回 nil, nil）
func (r *GitHubSyncRepo) GetSetting(ctx context.Context, kbID string) (*domain.GitHubSyncSetting, error) {
	var setting domain.Setting
	err := r.db.WithContext(ctx).Table("settings").
		Where("kb_id = ? AND key = ?", kbID, domain.SettingGitHubSync).
		First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get github sync setting failed: %w", err)
	}
	var cfg domain.GitHubSyncSetting
	if err := json.Unmarshal(setting.Value, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal github sync setting failed: %w", err)
	}
	cfg.KBID = setting.KBID
	return &cfg, nil
}

// SaveSetting 写入（upsert）指定 KB 的 git 同步配置
func (r *GitHubSyncRepo) SaveSetting(ctx context.Context, kbID string, cfg *domain.GitHubSyncSetting) error {
	cfg.KBID = kbID
	value, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal github sync setting failed: %w", err)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("settings").
			Where("kb_id = ? AND key = ?", kbID, domain.SettingGitHubSync).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return tx.Table("settings").
				Where("kb_id = ? AND key = ?", kbID, domain.SettingGitHubSync).
				Update("value", value).Error
		}
		setting := domain.Setting{
			KBID:        kbID,
			Key:         domain.SettingGitHubSync,
			Value:       value,
			Description: "github md sync config",
		}
		return tx.Table("settings").Create(&setting).Error
	})
}

// GitHubSyncedNode 是已建立同步映射的节点摘要
type GitHubSyncedNode struct {
	ID    string `gorm:"column:id"`
	Name  string `gorm:"column:name"`
	NavId string `gorm:"column:nav_id"`
	Path  string `gorm:"column:path"`
	SHA   string `gorm:"column:sha"`
	Type  int    `gorm:"column:type"`
}

// GetSyncedNodes 返回该 KB 下全部建立了 github_sync 映射的节点（文档与目录）
func (r *GitHubSyncRepo) GetSyncedNodes(ctx context.Context, kbID, repoFullName string) ([]GitHubSyncedNode, error) {
	var nodes []GitHubSyncedNode
	err := r.db.WithContext(ctx).Model(&domain.Node{}).
		Select("id, name, nav_id, type, meta->'github_sync'->>'path' as path, meta->'github_sync'->>'sha' as sha").
		Where("kb_id = ? AND meta->'github_sync'->>'repo' = ?", kbID, repoFullName).
		Find(&nodes).Error
	if err != nil {
		return nil, fmt.Errorf("get github synced nodes failed: %w", err)
	}
	return nodes, nil
}

// ListEnabledSettings 返回全部开启了 git 同步且配置了仓库的 KB 配置
func (r *GitHubSyncRepo) ListEnabledSettings(ctx context.Context) ([]*domain.GitHubSyncSetting, error) {
	var settings []domain.Setting
	err := r.db.WithContext(ctx).Table("settings").
		Where("key = ? AND value->>'enabled' = 'true'", domain.SettingGitHubSync).
		Find(&settings).Error
	if err != nil {
		return nil, fmt.Errorf("list github sync settings failed: %w", err)
	}
	cfgs := make([]*domain.GitHubSyncSetting, 0, len(settings))
	for _, setting := range settings {
		var cfg domain.GitHubSyncSetting
		if err := json.Unmarshal(setting.Value, &cfg); err != nil {
			r.logger.Error("unmarshal github sync setting failed", log.String("kb_id", setting.KBID), log.Error(err))
			continue
		}
		cfg.KBID = setting.KBID
		cfgs = append(cfgs, &cfg)
	}
	return cfgs, nil
}
