package pg

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	v1 "github.com/chaitin/panda-wiki/api/stat/v1"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/store/pg"
	"github.com/chaitin/panda-wiki/utils"
)

type StatRepository struct {
	db    *pg.DB
	cache *cache.Cache
}

type SourceConversionStat struct {
	RefererHost string
	Visits      int64
	Conversions int64
}

func NewStatRepository(db *pg.DB, cahe *cache.Cache) *StatRepository {
	return &StatRepository{
		db:    db,
		cache: cahe,
	}
}

func (r *StatRepository) CreateStatPage(ctx context.Context, stat *domain.StatPage) error {
	return r.db.WithContext(ctx).Model(&domain.StatPage{}).Create(stat).Error
}

func (r *StatRepository) GetHotPages(ctx context.Context, kbID string) ([]*domain.HotPage, error) {
	var hotPages []*domain.HotPage
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Where("node_id != '' ").
		Where("scene = ?", domain.StatPageSceneNodeDetail).
		Group("node_id").
		Select("node_id, COUNT(*) as count").
		Order("count DESC").
		Limit(10).
		Find(&hotPages).Error; err != nil {
		return nil, err
	}
	return hotPages, nil
}

func (r *StatRepository) GetHotPagesNoLimit(ctx context.Context, kbID string) ([]*domain.HotPage, error) {
	var hotPages []*domain.HotPage
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Where("node_id != '' ").
		Where("scene = ?", domain.StatPageSceneNodeDetail).
		Group("node_id").
		Select("node_id, COUNT(*) as count").
		Find(&hotPages).Error; err != nil {
		return nil, err
	}
	return hotPages, nil
}

func (r *StatRepository) GetHotScene(ctx context.Context, kbID string) (map[domain.StatPageScene]int64, error) {
	var scenes map[domain.StatPageScene]int64
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Group("scene").
		Select("scene, COUNT(*) as count").
		Order("count DESC").
		Limit(10).
		Find(&scenes).Error; err != nil {
		return nil, err
	}
	return scenes, nil
}

func (r *StatRepository) GetHotRefererHosts(ctx context.Context, kbID string) ([]*domain.HotRefererHost, error) {
	var hotRefererHosts []*domain.HotRefererHost
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ? AND referer_host != ?", kbID, "").
		Group("referer_host").
		Select("referer_host, COUNT(*) as count").
		Order("count DESC").
		Limit(10).
		Find(&hotRefererHosts).Error; err != nil {
		return nil, err
	}
	return hotRefererHosts, nil
}

func (r *StatRepository) GetHotBrowsers(ctx context.Context, kbID string) (*domain.HotBrowser, error) {
	var hotBrowsers *domain.HotBrowser
	var osCount []domain.BrowserCount
	var browserCount []domain.BrowserCount

	query := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Where("browser_name != '' ").
		Group("browser_name").
		Select("browser_name as name, COUNT(*) as count")
	if err := query.Order("count DESC").Limit(10).Find(&browserCount).Error; err != nil {
		return nil, err
	}

	query = r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Where("browser_os != '' ").
		Group("browser_os").
		Select("browser_os as name, COUNT(*) as count")
	if err := query.Order("count DESC").Limit(10).Find(&osCount).Error; err != nil {
		return nil, err
	}

	hotBrowsers = &domain.HotBrowser{
		OS:      osCount,
		Browser: browserCount,
	}

	return hotBrowsers, nil
}

func (r *StatRepository) GetSourceConversionStats(ctx context.Context, kbID string) ([]SourceConversionStat, error) {
	stats := make([]SourceConversionStat, 0)
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Where("referer_host != '' ").
		Where("created_at > now() - interval '24h'").
		Select(`
			referer_host,
			COUNT(*) as visits,
			SUM(CASE WHEN scene = ? THEN 1 ELSE 0 END) as conversions
		`, domain.StatPageSceneChat).
		Group("referer_host").
		Order("visits DESC").
		Limit(20).
		Scan(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *StatRepository) GetStatPageCount(ctx context.Context, kbID string) (*v1.StatCountResp, error) {
	var count v1.StatCountResp
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Select("COUNT(DISTINCT ip) as ip_count, COUNT(DISTINCT session_id) as session_count, COUNT(*) as page_visit_count").
		Scan(&count).Error; err != nil {
		return nil, err
	}
	return &count, nil
}

func (r *StatRepository) GetInstantCount(ctx context.Context, kbID string) ([]*domain.InstantCountResp, error) {
	var instantCount []*domain.InstantCountResp
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ? AND created_at >= NOW() - INTERVAL '1h'", kbID).
		Select("date_trunc('minute', created_at) as time, COUNT(*) as count").
		Group("time").
		Order("time ASC").
		Find(&instantCount).Error; err != nil {
		return nil, err
	}
	return instantCount, nil
}

func (r *StatRepository) GetInstantPages(ctx context.Context, kbID string) ([]*domain.InstantPageResp, error) {
	var instantPages []*domain.InstantPageResp
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("kb_id = ?", kbID).
		Select("node_id, ip, scene, created_at,user_id").
		Order("created_at DESC").
		Limit(10).
		Find(&instantPages).Error; err != nil {
		return nil, err
	}
	return instantPages, nil
}

func (r *StatRepository) RemoveOldData(ctx context.Context) error {
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("created_at < ?", utils.GetTimeHourOffset(-24)).
		Delete(&domain.StatPage{}).Error; err != nil {
		return err
	}
	return nil
}

// GetYesterdayPVByNode 获取昨天的PV数据，按node_id分组
func (r *StatRepository) GetYesterdayPVByNode(ctx context.Context) (map[string]int64, error) {
	type PVResult struct {
		NodeID string
		Count  int64
	}

	var results []PVResult
	if err := r.db.WithContext(ctx).Model(&domain.StatPage{}).
		Where("created_at < ?", utils.GetTimeHourOffset(0)).
		Where("created_at >= ?", utils.GetTimeHourOffset(-24)).
		Where("node_id != ?", "").
		Group("node_id").
		Select("node_id, COUNT(*) as count").
		Find(&results).Error; err != nil {
		return nil, err
	}

	pvMap := make(map[string]int64)
	for _, result := range results {
		pvMap[result.NodeID] = result.Count
	}
	return pvMap, nil
}

// UpsertNodeStats 插入或更新node_stats表
func (r *StatRepository) UpsertNodeStats(ctx context.Context, nodeID string, pvCount int64) error {
	nodeStats := &domain.NodeStats{
		NodeID: nodeID,
		PV:     pvCount,
	}

	// 使用GORM的Clauses进行upsert操作
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "node_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"pv": gorm.Expr("node_stats.pv + ?", pvCount),
			}),
		}).
		Create(nodeStats).Error
}
