package bot

import "context"

// PushNotifier defines the interface for sending push messages to group chats
// across different bot platforms (Feishu, DingTalk, WeChat, Discord, etc.)
type PushNotifier interface {
	// SendTextMessage sends a plain text message to a group chat.
	SendTextMessage(ctx context.Context, chatID string, content string) error
}
