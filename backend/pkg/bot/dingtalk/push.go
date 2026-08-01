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
	"strconv"
	"strings"
	"time"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
)

// DingTalkPushNotifier sends messages to DingTalk group chats via Incoming Webhook.
// The "chatID" parameter is expected to be a webhook URL, optionally followed by a
// signing secret separated by "|":
//
//	https://oapi.dingtalk.com/robot/send?access_token=XXX
//	https://oapi.dingtalk.com/robot/send?access_token=XXX|SECxxxx
type DingTalkPushNotifier struct {
	httpClient *http.Client
	logger     *log.Logger
	now        func() time.Time
}

func NewDingTalkPushNotifier(logger *log.Logger) *DingTalkPushNotifier {
	return &DingTalkPushNotifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger.WithModule("bot.dingtalk.push"),
		now:        time.Now,
	}
}

type dingTalkWebhookMsg struct {
	MsgType  string               `json:"msgtype"`
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
// When a signing secret is provided (url|secret), timestamp and sign are appended to the query.
func (n *DingTalkPushNotifier) SendTextMessage(ctx context.Context, chatID string, content string) error {
	webhookURL, secret := bot.ParseWebhookTarget(chatID)
	if webhookURL == "" {
		return fmt.Errorf("dingtalk webhook url is empty")
	}
	if secret != "" {
		signedURL, err := AppendDingTalkWebhookSign(webhookURL, n.now().Unix(), secret)
		if err != nil {
			return err
		}
		webhookURL = signedURL
	}

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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
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

// SignDingTalkWebhook generates the HMAC-SHA256 signature for a DingTalk webhook.
// DingTalk docs: HMAC-SHA256(key=secret, msg=timestamp+"\n"+secret), then Base64.
// Callers should URL-encode when placing the value into a query string.
func SignDingTalkWebhook(timestamp int64, secret string) (string, error) {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	if _, err := h.Write([]byte(stringToSign)); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// AppendDingTalkWebhookSign adds timestamp and sign query params required by signed webhooks.
func AppendDingTalkWebhookSign(webhookURL string, timestamp int64, secret string) (string, error) {
	sign, err := SignDingTalkWebhook(timestamp, secret)
	if err != nil {
		return "", fmt.Errorf("sign dingtalk webhook failed: %w", err)
	}
	u, err := url.Parse(webhookURL)
	if err != nil {
		return "", fmt.Errorf("parse dingtalk webhook url failed: %w", err)
	}
	q := u.Query()
	q.Set("timestamp", strconv.FormatInt(timestamp, 10))
	q.Set("sign", sign)
	u.RawQuery = q.Encode()
	return u.String(), nil
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
