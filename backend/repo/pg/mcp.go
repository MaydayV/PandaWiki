package pg

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/pg"
)

type MCPRepository struct {
	db     *pg.DB
	logger *log.Logger
}

func NewMCPRepository(db *pg.DB, logger *log.Logger) *MCPRepository {
	return &MCPRepository{db: db, logger: logger}
}

func (r *MCPRepository) GetMCPCallCount(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Table("mcp_calls").Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

type MCPDocSearchResult struct {
	NodeID  string `json:"node_id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

func (r *MCPRepository) SearchReleasedDocs(ctx context.Context, kbID, query string, limit int) ([]*MCPDocSearchResult, error) {
	if limit <= 0 {
		limit = 5
	}
	query = strings.TrimSpace(query)
	queryPattern := "%" + query + "%"

	results := make([]*MCPDocSearchResult, 0)
	if err := r.db.WithContext(ctx).
		Table("kb_release_node_releases krnr").
		Select("nr.node_id, nr.name, nr.meta->>'summary' AS summary, nr.content").
		Joins("JOIN node_releases nr ON nr.id = krnr.node_release_id").
		Joins(`
			JOIN (
				SELECT id
				FROM kb_releases
				WHERE kb_id = ?
				ORDER BY created_at DESC
				LIMIT 1
			) latest_release ON latest_release.id = krnr.release_id
		`, kbID).
		Where("krnr.kb_id = ?", kbID).
		Where("nr.type = ?", domain.NodeTypeDocument).
		Where("( ? = '' OR nr.name ILIKE ? OR nr.content ILIKE ? OR nr.meta->>'summary' ILIKE ? )", query, queryPattern, queryPattern, queryPattern).
		Order("nr.updated_at DESC").
		Limit(limit).
		Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *MCPRepository) LogInitializeCall(ctx context.Context, sessionID, kbID, remoteIP string, req, resp any) error {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "unknown"
	}
	initializeReq, err := marshalJSON(req)
	if err != nil {
		return err
	}
	initializeResp, err := marshalJSON(resp)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Exec(
		`INSERT INTO mcp_calls (mcp_session_id, kb_id, remote_ip, initialize_req, initialize_resp)
		 VALUES (?, ?, ?, ?::jsonb, ?::jsonb)`,
		sessionID, kbID, remoteIP, initializeReq, initializeResp,
	).Error
}

func (r *MCPRepository) LogToolCall(ctx context.Context, sessionID, kbID, remoteIP string, req, resp any) error {
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "unknown"
	}
	toolCallReq, err := marshalJSON(req)
	if err != nil {
		return err
	}
	toolCallResp, err := marshalJSON(resp)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Exec(
		`INSERT INTO mcp_calls (mcp_session_id, kb_id, remote_ip, tool_call_req, tool_call_resp)
		 VALUES (?, ?, ?, ?::jsonb, ?)`,
		sessionID, kbID, remoteIP, toolCallReq, toolCallResp,
	).Error
}

func marshalJSON(v any) (string, error) {
	if v == nil {
		return "null", nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
