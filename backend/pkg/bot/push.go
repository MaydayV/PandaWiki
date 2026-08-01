package bot

import (
	"context"
	"strings"
)

// PushNotifier defines the interface for sending push messages to group chats
// across different bot platforms (Feishu, DingTalk, WeChat, Discord, etc.)
type PushNotifier interface {
	// SendTextMessage sends a plain text message to a group chat.
	SendTextMessage(ctx context.Context, chatID string, content string) error
}

// ParseWebhookTarget splits "url" or "url|secret" into webhook URL and optional secret.
func ParseWebhookTarget(chatID string) (webhookURL, secret string) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "", ""
	}
	if i := strings.LastIndex(chatID, "|"); i > 0 {
		return strings.TrimSpace(chatID[:i]), strings.TrimSpace(chatID[i+1:])
	}
	return chatID, ""
}
