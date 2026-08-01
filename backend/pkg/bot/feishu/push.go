package feishu

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
	"strconv"
	"strings"
	"time"

	"github.com/chaitin/panda-wiki/pkg/bot"
)

// FeishuWebhookNotifier sends messages to Feishu group chats via Custom Bot Webhook.
// The "chatID" parameter is expected to be a webhook URL, optionally followed by a
// signing secret separated by "|":
//
//	https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxxx
//	https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxxx|SECXXXX
type FeishuWebhookNotifier struct {
	httpClient *http.Client
	now        func() time.Time
}

func NewFeishuWebhookNotifier() *FeishuWebhookNotifier {
	return &FeishuWebhookNotifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		now:        time.Now,
	}
}

type feishuWebhookMsg struct {
	MsgType   string          `json:"msg_type"`
	Content   json.RawMessage `json:"content"`
	Timestamp string          `json:"timestamp,omitempty"`
	Sign      string          `json:"sign,omitempty"`
}

type feishuWebhookResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendTextMessage sends a text message to a Feishu group via webhook URL.
// When a signing secret is provided (url|secret), timestamp and sign are attached to the body.
func (n *FeishuWebhookNotifier) SendTextMessage(ctx context.Context, chatID string, content string) error {
	webhookURL, secret := bot.ParseWebhookTarget(chatID)
	if webhookURL == "" {
		return fmt.Errorf("feishu webhook url is empty")
	}

	textContent, err := json.Marshal(map[string]string{"text": content})
	if err != nil {
		return fmt.Errorf("marshal feishu webhook content failed: %w", err)
	}
	msg := feishuWebhookMsg{
		MsgType: "text",
		Content: textContent,
	}
	if secret != "" {
		ts := n.now().Unix()
		msg.Timestamp = strconv.FormatInt(ts, 10)
		msg.Sign = SignFeishuWebhook(ts, secret)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal feishu webhook message failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create feishu webhook request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu webhook failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result feishuWebhookResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse feishu webhook response failed: %w (body: %s)", err, string(respBody))
	}
	if result.Code != 0 {
		return fmt.Errorf("feishu webhook error: code=%d msg=%s", result.Code, result.Msg)
	}

	return nil
}

// SignFeishuWebhook generates the HMAC-SHA256 signature for a Feishu webhook.
// Feishu docs: use timestamp+"\n"+secret as the HMAC key and sign an empty message.
func SignFeishuWebhook(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write(nil)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// maskFeishuWebhookURL masks the hook key in a webhook URL for safe logging.
func maskFeishuWebhookURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "***"
	}
	parts := strings.Split(u.Path, "/")
	if len(parts) > 0 {
		key := parts[len(parts)-1]
		if len(key) > 6 {
			parts[len(parts)-1] = key[:3] + "***" + key[len(key)-3:]
			u.Path = strings.Join(parts, "/")
		}
	}
	return u.String()
}
