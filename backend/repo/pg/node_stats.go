package pg

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
)

func (r *NodeRepository) GetNodeStatsByNodeId(ctx context.Context, nodeId string) (*domain.NodeStats, error) {
	var nodeStats *domain.NodeStats
	if err := r.db.WithContext(ctx).
		Model(&domain.NodeStats{}).
		Where("node_id = ?", nodeId).
		First(&nodeStats).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nodeStats = &domain.NodeStats{
				ID:     0,
				NodeID: nodeId,
				PV:     0,
			}
		} else {
			return nil, err
		}
	}

	// 实时增量从"本地当天零点"起算：午夜迁移（cron Hour()==0，本地时区）已把
	// 昨天全天 PV 累加进 node_stats.pv，窗口边界必须与迁移边界同为本地零点
	// （time.Truncate 按 UTC 截断，直接用会与本地零点错位）
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var todayStats int64
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("created_at >= ?", midnight).
		Where("node_id = ?", nodeId).Count(&todayStats).Error; err != nil {
		return nil, err
	}
	nodeStats.PV += todayStats

	return nodeStats, nil
}
