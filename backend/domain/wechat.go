package domain

import (
	"bytes"
	"sync"
	"time"
)

// NotifySendTimeout 是生产者向无缓冲 NotificationChan 推送的最大等待时间：
// 正常消费是毫秒级同步交接；超时意味着消费端已断开，放弃实时推送（内容已在 Buffer）。
const NotifySendTimeout = 5 * time.Second

// ConversationState
type ConversationState struct {
	Mutex            sync.Mutex
	KBID             string // 会话归属 KB，answer SSE 端点据此校验跨 KB 读取
	Question         string
	Buffer           bytes.Buffer
	IsVisited        bool
	IsDone           bool
	NotificationChan chan string
}

// ConversationManager
var ConversationManager = sync.Map{}

type WechatStatic struct {
	BaseUrl   string `json:"base_url"`
	ImagePath string `json:"image_path"`
}
