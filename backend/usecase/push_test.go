package usecase

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
	"github.com/chaitin/panda-wiki/pkg/bot/discord"
	"github.com/stretchr/testify/require"
)

func testPushLogger() *log.Logger {
	return &log.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func TestRenderTemplateUsesDefaultsAndEscapesMarkdown(t *testing.T) {
	u := &PushUsecase{}
	kb := &domain.KnowledgeBase{Name: "Docs *A*"}
	release := &domain.KBRelease{
		Tag:       "v1.0_beta",
		Message:   "fix #1 | update",
		CreatedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}

	got := u.renderTemplate("", kb, release)
	require.Contains(t, got, "Docs \\*A\\*")
	require.Contains(t, got, "v1.0\\_beta")
	require.Contains(t, got, "fix \\#1 \\| update")
	require.Contains(t, got, "2026-06-20")
}

func TestRenderTemplateCustomContent(t *testing.T) {
	u := &PushUsecase{}
	kb := &domain.KnowledgeBase{Name: "KB"}
	release := &domain.KBRelease{
		Tag:       "v2",
		Message:   "msg",
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	got := u.renderTemplate("更新 {kb_name}/{tag}: {message} @ {release_time}", kb, release)
	require.Equal(t, "更新 KB/v2: msg @ 2026-01-02 11:04:05", got)
}

func TestEscapeMarkdown(t *testing.T) {
	require.Equal(t, "a\\*b\\_c", escapeMarkdown("a*b_c"))
}

func TestResolveNotifierByWebhookHost(t *testing.T) {
	u := &PushUsecase{logger: testPushLogger(), notifiers: map[string]bot.PushNotifier{}}

	cases := []struct {
		name   string
		chatID string
	}{
		{name: "feishu", chatID: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"},
		{name: "dingtalk", chatID: "https://oapi.dingtalk.com/robot/send?access_token=abc"},
		{name: "wecom", chatID: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=abc"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			n, err := u.resolveNotifier(t.Context(), "", "app-1", tt.chatID)
			require.NoError(t, err)
			require.NotNil(t, n)
		})
	}
}

func TestResolveNotifierUnsupportedWebhook(t *testing.T) {
	u := &PushUsecase{logger: testPushLogger(), notifiers: map[string]bot.PushNotifier{}}
	_, err := u.resolveNotifier(t.Context(), "", "app-1", "https://example.com/hook")
	require.Error(t, err)
}

func TestResolveNotifierDiscordFromRegistry(t *testing.T) {
	discordNotifier := discord.NewDiscordPushNotifier(testPushLogger(), "token")
	u := &PushUsecase{
		logger: testPushLogger(),
		notifiers: map[string]bot.PushNotifier{
			"discord-app": discordNotifier,
		},
	}
	n, err := u.resolveNotifier(t.Context(), "", "discord-app", "1234567890")
	require.NoError(t, err)
	require.Equal(t, discordNotifier, n)
}
