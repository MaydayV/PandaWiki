package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"gorm.io/gorm"
)

type contributeRow struct {
	ID             string                  `gorm:"column:id"`
	AuthID         *int64                  `gorm:"column:auth_id"`
	KBID           string                  `gorm:"column:kb_id"`
	Status         consts.ContributeStatus `gorm:"column:status"`
	Type           consts.ContributeType   `gorm:"column:type"`
	NodeID         string                  `gorm:"column:node_id"`
	NodeName       string                  `gorm:"column:node_name"`
	ContributeName string                  `gorm:"column:contribute_name"`
	Content        string                  `gorm:"column:content"`
	Meta           domain.NodeMeta         `gorm:"column:meta"`
	Reason         string                  `gorm:"column:reason"`
	AuditUserID    string                  `gorm:"column:audit_user_id"`
	AuditTime      *time.Time              `gorm:"column:audit_time"`
	RemoteIP       string                  `gorm:"column:remote_ip"`
	CreatedAt      time.Time               `gorm:"column:created_at"`
	UpdatedAt      time.Time               `gorm:"column:updated_at"`
	AuthUserInfo   []byte                  `gorm:"column:auth_user_info"`
}

func (r *NodeRepository) CreateContribute(ctx context.Context, contribute *domain.Contribute) error {
	return r.db.WithContext(ctx).Create(contribute).Error
}

func (r *NodeRepository) GetContributeByID(ctx context.Context, kbID, id string) (*domain.Contribute, error) {
	var contribute domain.Contribute
	if err := r.db.WithContext(ctx).
		Model(&domain.Contribute{}).
		Where("kb_id = ?", kbID).
		Where("id = ?", id).
		First(&contribute).Error; err != nil {
		return nil, err
	}
	return &contribute, nil
}

func (r *NodeRepository) GetContributeList(ctx context.Context, req *domain.ContributeListReq) ([]*domain.ContributeItemResp, int64, error) {
	baseQuery := r.buildContributeBaseQuery(ctx, req)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := make([]*contributeRow, 0)
	if err := baseQuery.
		Select(`
			c.id,
			c.auth_id,
			c.kb_id,
			c.status,
			c.type,
			c.node_id,
			COALESCE(NULLIF(c.name, ''), n.name, '') AS node_name,
			COALESCE(c.name, '') AS contribute_name,
			COALESCE(c.meta, '{}'::jsonb) AS meta,
			c.reason,
			c.audit_user_id,
			c.audit_time,
			c.remote_ip,
			c.created_at,
			c.updated_at,
			COALESCE(a.user_info, '{}'::jsonb) AS auth_user_info
		`).
		Order("c.created_at DESC").
		Offset(req.Offset()).
		Limit(req.Limit()).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*domain.ContributeItemResp, 0, len(rows))
	for _, row := range rows {
		authUserInfo := parseAuthUserInfo(row.AuthUserInfo)
		item := &domain.ContributeItemResp{
			ID:             row.ID,
			AuthID:         row.AuthID,
			AuthName:       authUserInfo.Username,
			Avatar:         authUserInfo.AvatarUrl,
			KBID:           row.KBID,
			Status:         row.Status,
			Type:           row.Type,
			NodeID:         row.NodeID,
			NodeName:       row.NodeName,
			ContributeName: row.ContributeName,
			Meta:           row.Meta,
			Reason:         row.Reason,
			AuditUserID:    row.AuditUserID,
			AuditTime:      row.AuditTime,
			RemoteIP:       row.RemoteIP,
			IPAddress: &domain.IPAddress{
				IP: row.RemoteIP,
			},
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *NodeRepository) GetContributeDetail(ctx context.Context, req *domain.ContributeDetailReq) (*domain.ContributeDetailResp, error) {
	row := &contributeRow{}
	err := r.db.WithContext(ctx).
		Table("contributes AS c").
		Joins("LEFT JOIN auths AS a ON a.id = c.auth_id").
		Joins("LEFT JOIN nodes AS n ON n.id = c.node_id").
		Where("c.kb_id = ?", req.KBID).
		Where("c.id = ?", req.ID).
		Select(`
			c.id,
			c.auth_id,
			c.kb_id,
			c.status,
			c.type,
			c.node_id,
			COALESCE(NULLIF(c.name, ''), n.name, '') AS node_name,
			c.content,
			COALESCE(c.meta, '{}'::jsonb) AS meta,
			c.reason,
			c.audit_user_id,
			c.audit_time,
			c.created_at,
			c.updated_at,
			COALESCE(a.user_info, '{}'::jsonb) AS auth_user_info
		`).
		First(row).Error
	if err != nil {
		return nil, err
	}

	authUserInfo := parseAuthUserInfo(row.AuthUserInfo)
	return &domain.ContributeDetailResp{
		ID:          row.ID,
		AuthID:      row.AuthID,
		AuthName:    authUserInfo.Username,
		Avatar:      authUserInfo.AvatarUrl,
		KBID:        row.KBID,
		Status:      row.Status,
		Type:        row.Type,
		NodeID:      row.NodeID,
		NodeName:    row.NodeName,
		Content:     row.Content,
		Meta:        row.Meta,
		Reason:      row.Reason,
		AuditUserID: row.AuditUserID,
		AuditTime:   row.AuditTime,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *NodeRepository) UpdateContributeAudit(ctx context.Context, kbID, id, auditUserID string, status consts.ContributeStatus, approvedNodeID *string) (int64, error) {
	updateMap := map[string]any{
		"status":        status,
		"audit_user_id": auditUserID,
		"audit_time":    time.Now(),
	}
	if approvedNodeID != nil && *approvedNodeID != "" {
		updateMap["node_id"] = *approvedNodeID
	}

	tx := r.db.WithContext(ctx).
		Model(&domain.Contribute{}).
		Where("id = ?", id).
		Where("kb_id = ?", kbID).
		Where("status = ?", consts.ContributeStatusPending).
		Updates(updateMap)
	return tx.RowsAffected, tx.Error
}

func (r *NodeRepository) buildContributeBaseQuery(ctx context.Context, req *domain.ContributeListReq) *gorm.DB {
	query := r.db.WithContext(ctx).
		Table("contributes AS c").
		Joins("LEFT JOIN auths AS a ON a.id = c.auth_id").
		Joins("LEFT JOIN nodes AS n ON n.id = c.node_id").
		Where("c.kb_id = ?", req.KBID)

	if req.Status != "" {
		query = query.Where("c.status = ?", req.Status)
	}
	if req.AuthName != "" {
		likePattern := fmt.Sprintf("%%%s%%", req.AuthName)
		query = query.Where("COALESCE(a.user_info->>'username', '') LIKE ?", likePattern)
	}
	if req.NodeName != "" {
		likePattern := fmt.Sprintf("%%%s%%", req.NodeName)
		query = query.Where("COALESCE(NULLIF(c.name, ''), n.name, '') LIKE ?", likePattern)
	}
	return query
}

func parseAuthUserInfo(raw []byte) domain.AuthUserInfo {
	if len(raw) == 0 {
		return domain.AuthUserInfo{}
	}
	info := domain.AuthUserInfo{}
	if err := json.Unmarshal(raw, &info); err != nil {
		return domain.AuthUserInfo{}
	}
	return info
}
