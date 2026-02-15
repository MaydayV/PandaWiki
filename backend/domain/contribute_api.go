package domain

import (
	"time"

	"github.com/chaitin/panda-wiki/consts"
)

type ContributeListReq struct {
	KBID     string                  `json:"kb_id" query:"kb_id" validate:"required"`
	NodeName string                  `json:"node_name" query:"node_name"`
	AuthName string                  `json:"auth_name" query:"auth_name"`
	Status   consts.ContributeStatus `json:"status" query:"status" validate:"omitempty,oneof=pending approved rejected"`
	Pager
}

type ContributeListResp struct {
	List  []*ContributeItemResp `json:"list"`
	Total int64                 `json:"total"`
}

type ContributeItemResp struct {
	ID             string                  `json:"id"`
	AuthID         *int64                  `json:"auth_id"`
	AuthName       string                  `json:"auth_name"`
	Avatar         string                  `json:"avatar,omitempty"`
	KBID           string                  `json:"kb_id"`
	Status         consts.ContributeStatus `json:"status"`
	Type           consts.ContributeType   `json:"type"`
	NodeID         string                  `json:"node_id"`
	NodeName       string                  `json:"node_name"`
	ContributeName string                  `json:"contribute_name"`
	Meta           NodeMeta                `json:"meta"`
	Reason         string                  `json:"reason"`
	AuditUserID    string                  `json:"audit_user_id"`
	AuditTime      *time.Time              `json:"audit_time"`
	RemoteIP       string                  `json:"remote_ip"`
	IPAddress      *IPAddress              `json:"ip_address"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type ContributeDetailReq struct {
	ID   string `json:"id" query:"id" validate:"required"`
	KBID string `json:"kb_id" query:"kb_id" validate:"required"`
}

type ContributeOriginalNodeInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Content string   `json:"content"`
	Meta    NodeMeta `json:"meta"`
}

type ContributeDetailResp struct {
	ID           string                      `json:"id"`
	AuthID       *int64                      `json:"auth_id"`
	AuthName     string                      `json:"auth_name"`
	Avatar       string                      `json:"avatar,omitempty"`
	KBID         string                      `json:"kb_id"`
	Status       consts.ContributeStatus     `json:"status"`
	Type         consts.ContributeType       `json:"type"`
	NodeID       string                      `json:"node_id"`
	NodeName     string                      `json:"node_name"`
	Content      string                      `json:"content"`
	Meta         NodeMeta                    `json:"meta"`
	Reason       string                      `json:"reason"`
	AuditUserID  string                      `json:"audit_user_id"`
	AuditTime    *time.Time                  `json:"audit_time"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	OriginalNode *ContributeOriginalNodeInfo `json:"original_node,omitempty"`
}

type ContributeAuditReq struct {
	ID       string                  `json:"id" validate:"required"`
	KBID     string                  `json:"kb_id" validate:"required"`
	ParentID string                  `json:"parent_id"`
	Position *float64                `json:"position"`
	Status   consts.ContributeStatus `json:"status" validate:"required,oneof=approved rejected"`
}

type ContributeAuditResp struct {
	Message string `json:"message"`
}

type SubmitContributeReq struct {
	CaptchaToken string                `json:"captcha_token" validate:"required"`
	Type         consts.ContributeType `json:"type" validate:"required,oneof=add edit"`
	NodeID       string                `json:"node_id"`
	Name         string                `json:"name"`
	Reason       string                `json:"reason" validate:"required"`
	Content      string                `json:"content"`
	Emoji        string                `json:"emoji"`
	ContentType  string                `json:"content_type" validate:"required,oneof=html md"`
}

type SubmitContributeResp struct {
	ID string `json:"id"`
}
