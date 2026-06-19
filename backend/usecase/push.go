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
	"github.com/chaitin/panda-wiki/repo/pg"
)

const defaultPushTemplate = "📚 知识库「{kb_name}」已更新\n版本：{tag} | 发布说明：{message}\n发布时间：{release_time}"

// PushUsecase manages push notifications for knowledge base updates.
// Notifiers are registered at runtime when bots start (via RegisterNotifier).
type PushUsecase struct {
	appRepo   *pg.AppRepository
	kbRepo    *pg.KnowledgeBaseRepository
	logger    *log.Logger
	mu        sync.RWMutex
	notifiers map[string]bot.PushNotifier // appID → notifier
}

func NewPushUsecase(appRepo *pg.AppRepository, kbRepo *pg.KnowledgeBaseRepository, logger *log.Logger) *PushUsecase {
	return &PushUsecase{
		appRepo:   appRepo,
		kbRepo:    kbRepo,
		logger:    logger.WithModule("usecase.push"),
		notifiers: make(map[string]bot.PushNotifier),
	}
}

// RegisterNotifier registers a push notifier for an app.
// Called by AppUsecase when a bot starts.
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

		u.mu.RLock()
		notifier, ok := u.notifiers[app.ID]
		u.mu.RUnlock()
		if !ok {
			u.logger.Debug("push: no notifier for app", log.String("app_id", app.ID))
			continue
		}

		content := u.renderTemplate(app.Settings.KBUpdatePushContent, kb, release)
		chatIDs := strings.Split(app.Settings.KBUpdatePushChatIDs, ",")
		for _, chatID := range chatIDs {
			chatID = strings.TrimSpace(chatID)
			if chatID == "" {
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
			time.Sleep(time.Second)
		}
	}
}

func (u *PushUsecase) renderTemplate(tmpl string, kb *domain.KnowledgeBase, release *domain.KBRelease) string {
	if strings.TrimSpace(tmpl) == "" {
		tmpl = defaultPushTemplate
	}
	releaseTime := release.CreatedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
	replacer := strings.NewReplacer(
		"{kb_name}", kb.Name,
		"{tag}", release.Tag,
		"{message}", release.Message,
		"{release_time}", releaseTime,
	)
	return replacer.Replace(tmpl)
}

// TestPush sends a test message to a specific chat ID via the notifier for the given app.
func (u *PushUsecase) TestPush(ctx context.Context, appID, chatID string) error {
	u.mu.RLock()
	notifier, ok := u.notifiers[appID]
	u.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no push notifier registered for app %s", appID)
	}
	return notifier.SendTextMessage(ctx, chatID, "✅ PandaWiki 推送测试成功")
}
