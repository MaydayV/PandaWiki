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
	"strings"
	"time"
)

// FeishuWebhookNotifier sends messages to Feishu group chats via Custom Bot Webhook.
// The "chatID" parameter is expected to be a full webhook URL:
//
//	https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxxx
//
// or with signing:
//
//	https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxxx?timestamp=...&sign=...
type FeishuWebhookNotifier struct {
	httpClient *http.Client
}

func NewFeishuWebhookNotifier() *FeishuWebhookNotifier {
	return &FeishuWebhookNotifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type feishuWebhookMsg struct {
	MsgType string          `json:"msg_type"`
	Content json.RawMessage `json:"content"`
	// signing fields (optional)
	Timestamp string `json:"timestamp,omitempty"`
	Sign      string `json:"sign,omitempty"`
}

type feishuWebhookResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendTextMessage sends a text message to a Feishu group via webhook URL.
// The chatID is the full webhook URL (https://open.feishu.cn/open-apis/bot/v2/hook/xxx).
func (n *FeishuWebhookNotifier) SendTextMessage(ctx context.Context, chatID string, content string) error {
	webhookURL := chatID
	textContent, _ := json.Marshal(map[string]string{"text": content})
	msg := feishuWebhookMsg{
		MsgType: "text",
		Content: textContent,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal feishu webhook message failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
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

// SignFeishuWebhook generates the HMAC-SHA256 signature for a Feishu webhook URL.
// Feishu signing: timestamp + "\n" + secret → HMAC-SHA256 → base64
func SignFeishuWebhook(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte{})
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
