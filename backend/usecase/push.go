package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
	"github.com/chaitin/panda-wiki/pkg/bot/dingtalk"
	"github.com/chaitin/panda-wiki/pkg/bot/discord"
	"github.com/chaitin/panda-wiki/pkg/bot/feishu"
	"github.com/chaitin/panda-wiki/pkg/bot/wecom"
	"github.com/chaitin/panda-wiki/repo/pg"
)

const defaultPushTemplate = "📚 知识库「{kb_name}」已更新\n版本：{tag} | 发布说明：{message}\n发布时间：{release_time}"

// PushUsecase manages push notifications for knowledge base updates.
// Webhook targets (Feishu/DingTalk/WeCom) are routed by URL.
// Discord channel IDs use notifiers registered when Discord bots start.
type PushUsecase struct {
	appRepo   *pg.AppRepository
	kbRepo    *pg.KnowledgeBaseRepository
	logger    *log.Logger
	mu        sync.RWMutex
	notifiers map[string]bot.PushNotifier // appID → Discord (or other token-based) notifier
}

func NewPushUsecase(appRepo *pg.AppRepository, kbRepo *pg.KnowledgeBaseRepository, logger *log.Logger) *PushUsecase {
	return &PushUsecase{
		appRepo:   appRepo,
		kbRepo:    kbRepo,
		logger:    logger.WithModule("usecase.push"),
		notifiers: make(map[string]bot.PushNotifier),
	}
}

// RegisterNotifier registers a token-based push notifier for an app (e.g. Discord).
func (u *PushUsecase) RegisterNotifier(appID string, notifier bot.PushNotifier) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.notifiers[appID] = notifier
	u.logger.Info("push notifier registered", log.String("app_id", appID))
}

// UnregisterNotifier removes a push notifier for an app.
func (u *PushUsecase) UnregisterNotifier(appID string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.notifiers, appID)
}

// NotifyKBUpdate sends push notifications to all configured group chats
// for apps associated with the given knowledge base.
// This is intended to be called asynchronously (in a goroutine) — errors are logged, not returned.
func (u *PushUsecase) NotifyKBUpdate(ctx context.Context, kbID string, release *domain.KBRelease) {
	apps, err := u.appRepo.GetAppList(ctx, kbID)
	if err != nil {
		u.logger.Error("push: failed to get apps for kb", log.String("kb_id", kbID), log.Error(err))
		return
	}

	kb, err := u.kbRepo.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		u.logger.Error("push: failed to get knowledge base", log.String("kb_id", kbID), log.Error(err))
		return
	}

	for _, app := range apps {
		if !app.Settings.KBUpdatePushEnabled {
			continue
		}
		if strings.TrimSpace(app.Settings.KBUpdatePushChatIDs) == "" {
			continue
		}

		content := u.renderTemplate(app.Settings.KBUpdatePushContent, kb, release)
		chatIDs := strings.Split(app.Settings.KBUpdatePushChatIDs, ",")
		for _, chatID := range chatIDs {
			chatID = strings.TrimSpace(chatID)
			if chatID == "" {
				continue
			}
			notifier, err := u.resolveNotifier(ctx, kbID, app.ID, chatID)
			if err != nil {
				u.logger.Error("push: resolve notifier failed",
					log.String("app_id", app.ID),
					log.String("chat_id", chatID),
					log.Error(err))
				continue
			}
			if err := notifier.SendTextMessage(ctx, chatID, content); err != nil {
				u.logger.Error("push: send failed",
					log.String("app_id", app.ID),
					log.String("chat_id", chatID),
					log.Error(err))
			} else {
				u.logger.Info("push: sent successfully",
					log.String("app_id", app.ID),
					log.String("chat_id", chatID))
			}
			// rate limit: 1 second between sends
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}
}

func (u *PushUsecase) resolveNotifier(ctx context.Context, kbID, preferredAppID, chatID string) (bot.PushNotifier, error) {
	rawURL, _ := bot.ParseWebhookTarget(chatID)
	lower := strings.ToLower(rawURL)
	switch {
	case strings.Contains(lower, "qyapi.weixin.qq.com"):
		return wecom.NewWecomWebhookNotifier(u.logger), nil
	case strings.Contains(lower, "oapi.dingtalk.com"), strings.Contains(lower, "dingtalk.com"):
		return dingtalk.NewDingTalkPushNotifier(u.logger), nil
	case strings.Contains(lower, "feishu.cn"),
		strings.Contains(lower, "larksuite.com"),
		strings.Contains(lower, "larkoffice.com"):
		return feishu.NewFeishuWebhookNotifier(), nil
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return nil, fmt.Errorf("unsupported webhook host for push target")
	}

	// Non-URL targets are treated as Discord channel IDs.
	if n := u.lookupRegisteredNotifier(preferredAppID); n != nil {
		return n, nil
	}
	if kbID == "" && preferredAppID != "" {
		app, err := u.appRepo.GetAppDetail(ctx, preferredAppID)
		if err == nil {
			kbID = app.KBID
		}
	}
	if kbID != "" {
		apps, err := u.appRepo.GetAppList(ctx, kbID)
		if err != nil {
			return nil, fmt.Errorf("list apps for discord notifier: %w", err)
		}
		for _, app := range apps {
			if app.Type != domain.AppTypeDisCordBot {
				continue
			}
			if n := u.lookupRegisteredNotifier(app.ID); n != nil {
				return n, nil
			}
			token := strings.TrimSpace(app.Settings.DiscordBotToken)
			if token != "" && (app.Settings.DiscordBotIsEnabled == nil || *app.Settings.DiscordBotIsEnabled) {
				return discord.NewDiscordPushNotifier(u.logger, token), nil
			}
		}
	}
	return nil, fmt.Errorf("no discord push notifier available (enable Discord bot and configure channel id)")
}

func (u *PushUsecase) lookupRegisteredNotifier(appID string) bot.PushNotifier {
	if appID == "" {
		return nil
	}
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.notifiers[appID]
}

func (u *PushUsecase) renderTemplate(tmpl string, kb *domain.KnowledgeBase, release *domain.KBRelease) string {
	if strings.TrimSpace(tmpl) == "" {
		tmpl = defaultPushTemplate
	}
	releaseTime := release.CreatedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
	// escape markdown special characters to prevent format corruption
	safeName := escapeMarkdown(kb.Name)
	safeTag := escapeMarkdown(release.Tag)
	safeMsg := escapeMarkdown(release.Message)
	replacer := strings.NewReplacer(
		"{kb_name}", safeName,
		"{tag}", safeTag,
		"{message}", safeMsg,
		"{release_time}", releaseTime,
	)
	return replacer.Replace(tmpl)
}

// escapeMarkdown escapes characters that have special meaning in Markdown.
func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"#", "\\#",
		"`", "\\`",
		">", "\\>",
		"[", "\\[",
		"]", "\\]",
		"|", "\\|",
	)
	return replacer.Replace(s)
}

// TestPush sends a test message to a specific chat ID.
// Webhook URLs are routed by host; Discord channel IDs use the KB's Discord bot token.
func (u *PushUsecase) TestPush(ctx context.Context, appID, chatID string) error {
	notifier, err := u.resolveNotifier(ctx, "", appID, chatID)
	if err != nil {
		return err
	}
	return notifier.SendTextMessage(ctx, chatID, "✅ PandaWiki 推送测试成功")
}
