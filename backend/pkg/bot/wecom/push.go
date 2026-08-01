package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
)

// WecomWebhookNotifier sends messages to WeCom (企业微信) group chats via Group Robot Webhook.
// The "chatID" parameter is expected to be a webhook URL:
//
//	https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=XXXXXXXX
type WecomWebhookNotifier struct {
	httpClient *http.Client
	logger     *log.Logger
}

func NewWecomWebhookNotifier(logger *log.Logger) *WecomWebhookNotifier {
	return &WecomWebhookNotifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger.WithModule("bot.wecom.push"),
	}
}

type wecomWebhookMsg struct {
	MsgType  string             `json:"msgtype"`
	Markdown *wecomMarkdownBody `json:"markdown,omitempty"`
	Text     *wecomTextBody     `json:"text,omitempty"`
}

type wecomMarkdownBody struct {
	Content string `json:"content"`
}

type wecomTextBody struct {
	Content string `json:"content"`
}

type wecomWebhookResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// SendTextMessage sends a markdown message to a WeCom group via webhook URL.
func (n *WecomWebhookNotifier) SendTextMessage(ctx context.Context, chatID string, content string) error {
	webhookURL, _ := bot.ParseWebhookTarget(chatID)
	if webhookURL == "" {
		return fmt.Errorf("wecom webhook url is empty")
	}

	msg := wecomWebhookMsg{
		MsgType: "markdown",
		Markdown: &wecomMarkdownBody{
			Content: content,
		},
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal wecom webhook message failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create wecom webhook request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send wecom webhook failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result wecomWebhookResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse wecom webhook response failed: %w (body: %s)", err, string(respBody))
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("wecom webhook error: code=%d msg=%s", result.ErrCode, result.ErrMsg)
	}

	n.logger.Info("wecom push message sent")
	return nil
}
