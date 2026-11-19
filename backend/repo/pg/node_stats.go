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

	// 实时增量从"当天零点"起算：午夜迁移已把昨天全天 PV 累加进 node_stats.pv，
	// 若这里仍用滚动 24h 窗口，昨天的 PV 在当天会被双计
	midnight := time.Now().Truncate(24 * time.Hour)
	var todayStats int64
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("created_at >= ?", midnight).
		Where("node_id = ?", nodeId).Count(&todayStats).Error; err != nil {
		return nil, err
	}
	nodeStats.PV += todayStats

	return nodeStats, nil
}
