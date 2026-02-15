package domain

import (
	"context"
	"encoding/json"
)

const ContextKeyEditionLimitation contextKey = "edition_limitation"

type BaseEditionLimitation struct {
	MaxKb                  int   `json:"max_kb"`                     // 知识库站点数量
	MaxNode                int   `json:"max_node"`                   // 单个知识库下文档数量
	MaxSSOUser             int   `json:"max_sso_users"`              // SSO认证用户数量
	MaxAdmin               int64 `json:"max_admin"`                  // 后台管理员数量
	AllowAdminPerm         bool  `json:"allow_admin_perm"`           // 支持管理员分权控制
	AllowCustomCopyright   bool  `json:"allow_custom_copyright"`     // 支持自定义版权信息
	AllowCommentAudit      bool  `json:"allow_comment_audit"`        // 支持评论审核
	AllowAdvancedBot       bool  `json:"allow_advanced_bot"`         // 支持高级机器人配置
	AllowWatermark         bool  `json:"allow_watermark"`            // 支持水印
	AllowCopyProtection    bool  `json:"allow_copy_protection"`      // 支持内容复制保护
	AllowOpenAIBotSettings bool  `json:"allow_open_ai_bot_settings"` // 支持问答机器人
	AllowMCPServer         bool  `json:"allow_mcp_server"`           // 支持创建MCP Server
	AllowNodeStats         bool  `json:"allow_node_stats"`           // 支持文档统计
}

var baseEditionLimitationDefault = BaseEditionLimitation{
	// 乘风版默认全量开放能力，避免受原版授权档位限制。
	MaxKb:                  1000000,
	MaxNode:                1000000,
	MaxSSOUser:             1000000,
	MaxAdmin:               1000000,
	AllowAdminPerm:         true,
	AllowCustomCopyright:   true,
	AllowCommentAudit:      true,
	AllowAdvancedBot:       true,
	AllowWatermark:         true,
	AllowCopyProtection:    true,
	AllowOpenAIBotSettings: true,
	AllowMCPServer:         true,
	AllowNodeStats:         true,
}

func GetBaseEditionLimitation(c context.Context) BaseEditionLimitation {

	edition, ok := c.Value(ContextKeyEditionLimitation).([]byte)
	if !ok {
		return baseEditionLimitationDefault
	}

	var editionLimitation BaseEditionLimitation
	if err := json.Unmarshal(edition, &editionLimitation); err != nil {
		return baseEditionLimitationDefault
	}

	return editionLimitation
}
