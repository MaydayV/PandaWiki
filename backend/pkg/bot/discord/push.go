package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
)

const discordAPIBase = "https://discord.com/api/v10"
const discordContentLimit = 2000

// DiscordPushNotifier sends messages to Discord channels via Bot Token REST API.
// The "chatID" parameter is expected to be a channel snowflake ID.
type DiscordPushNotifier struct {
	httpClient *http.Client
	logger     *log.Logger
	token      string
	apiBase    string
}

func NewDiscordPushNotifier(logger *log.Logger, token string) *DiscordPushNotifier {
	return &DiscordPushNotifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger.WithModule("bot.discord.push"),
		token:      strings.TrimSpace(token),
		apiBase:    discordAPIBase,
	}
}

type discordMessageReq struct {
	Content string `json:"content"`
}

type discordAPIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SendTextMessage sends a text/markdown message to a Discord channel.
func (n *DiscordPushNotifier) SendTextMessage(ctx context.Context, chatID string, content string) error {
	channelID, _ := bot.ParseWebhookTarget(chatID)
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return fmt.Errorf("discord channel id is empty")
	}
	if n.token == "" {
		return fmt.Errorf("discord bot token is empty")
	}
	if len(content) > discordContentLimit {
		content = content[:discordContentLimit-3] + "..."
	}

	body, err := json.Marshal(discordMessageReq{Content: content})
	if err != nil {
		return fmt.Errorf("marshal discord message failed: %w", err)
	}

	url := fmt.Sprintf("%s/channels/%s/messages", n.apiBase, channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create discord request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+n.token)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send discord message failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var apiErr discordAPIError
		_ = json.Unmarshal(respBody, &apiErr)
		if apiErr.Message != "" {
			return fmt.Errorf("discord api error: status=%d code=%d msg=%s", resp.StatusCode, apiErr.Code, apiErr.Message)
		}
		return fmt.Errorf("discord api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	n.logger.Info("discord push message sent", log.String("channel_id", channelID))
	return nil
}
