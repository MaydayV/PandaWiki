package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chaitin/panda-wiki/log"
)

// DingTalkPushNotifier sends messages to DingTalk group chats via Incoming Webhook.
// The "chatID" parameter is expected to be a full webhook URL:
//
//	https://oapi.dingtalk.com/robot/send?access_token=XXX
//
// or with signing:
//
//	https://oapi.dingtalk.com/robot/send?access_token=XXX&timestamp=...&sign=...
type DingTalkPushNotifier struct {
	httpClient *http.Client
	logger     *log.Logger
}

func NewDingTalkPushNotifier(logger *log.Logger) *DingTalkPushNotifier {
	return &DingTalkPushNotifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger.WithModule("bot.dingtalk.push"),
	}
}

type dingTalkWebhookMsg struct {
	MsgType  string              `json:"msgtype"`
	Markdown *dingTalkMarkdownMsg `json:"markdown,omitempty"`
	Text     *dingTalkTextMsg     `json:"text,omitempty"`
}

type dingTalkMarkdownMsg struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type dingTalkTextMsg struct {
	Content string `json:"content"`
}

type dingTalkWebhookResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// SendTextMessage sends a markdown message to a DingTalk group via webhook URL.
// The chatID is the full webhook URL (with access_token parameter).
func (n *DingTalkPushNotifier) SendTextMessage(ctx context.Context, chatID string, content string) error {
	webhookURL := chatID
	msg := dingTalkWebhookMsg{
		MsgType: "markdown",
		Markdown: &dingTalkMarkdownMsg{
			Title: "知识库更新通知",
			Text:  content,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal dingtalk webhook message failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create dingtalk webhook request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send dingtalk webhook failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result dingTalkWebhookResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse dingtalk webhook response failed: %w (body: %s)", err, string(respBody))
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("dingtalk webhook error: code=%d msg=%s", result.ErrCode, result.ErrMsg)
	}

	n.logger.Info("dingtalk push message sent", log.String("webhook", maskWebhookURL(webhookURL)))
	return nil
}

// SignDingTalkWebhook generates the HMAC-SHA256 signature for a DingTalk webhook URL.
// This is used when the webhook has signing enabled.
func SignDingTalkWebhook(timestamp int64, secret string) (string, error) {
	// DingTalk signing: HMAC-SHA256(key=timestamp+"\n"+secret, msg=empty)
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	// h.Write(nil) computes HMAC of empty message — this is correct per DingTalk docs
	if _, err := h.Write(nil); err != nil {
		return "", err
	}
	return url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil))), nil
}

// maskWebhookURL masks the access_token in a webhook URL for safe logging.
func maskWebhookURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "***"
	}
	token := u.Query().Get("access_token")
	if len(token) > 8 {
		u.RawQuery = strings.Replace(u.RawQuery, token, token[:4]+"****"+token[len(token)-4:], 1)
	}
	return u.String()
}
