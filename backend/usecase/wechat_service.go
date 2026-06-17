package usecase

import (
	"context"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
	"github.com/chaitin/panda-wiki/pkg/bot/wechat_service"
	"github.com/chaitin/panda-wiki/repo/pg"
)

type WechatServiceUsecase struct {
	logger      *log.Logger
	AppUsecase  *AppUsecase
	authRepo    *pg.AuthRepo
	chatUsecase *ChatUsecase
	weRepo      *pg.WechatRepository
}

func NewWechatUsecase(logger *log.Logger, AppUsecase *AppUsecase, chatUsecase *ChatUsecase, weRepo *pg.WechatRepository, authRepo *pg.AuthRepo) *WechatServiceUsecase {
	return &WechatServiceUsecase{
		logger:      logger.WithModule("usecase.wechatUsecase"),
		AppUsecase:  AppUsecase,
		chatUsecase: chatUsecase,
		weRepo:      weRepo,
		authRepo:    authRepo,
	}
}

func (u *WechatServiceUsecase) VerifyUrlWechatService(ctx context.Context, signature, timestamp, nonce, echoStr string,
	WechatServiceConf *wechat_service.WechatServiceConfig) ([]byte, error) {
	body, err := WechatServiceConf.VerifyUrlWechatService(signature, timestamp, nonce, echoStr)
	if err != nil {
		u.logger.Error("WechatServiceConf verify url failed", log.Error(err))
		return nil, err
	}
	return body, nil
}

func (u *WechatServiceUsecase) WechatService(ctx context.Context, msg *wechat_service.WeixinUserAskMsg, kbID string, WechatServiceConfig *wechat_service.WechatServiceConfig) error {
	getQA := u.getQAFunc(kbID, domain.AppTypeWechatServiceBot)
	WechatServiceConfig.WeRepo = u.weRepo

	err := WechatServiceConfig.Wechat(msg, getQA, u.AppUsecase.GetWebAppCopyright(ctx, kbID))
	if err != nil {
		u.logger.Error("WechatServiceConf wechat failed", log.Error(err))
		return err
	}
	return nil
}

func (u *WechatServiceUsecase) NewWechatServiceConfig(ctx context.Context, kbID string, appInfo *domain.AppDetailResp) (*wechat_service.WechatServiceConfig, error) {
	return wechat_service.NewWechatServiceConfig(
		ctx,
		u.logger,
		kbID,
		appInfo.Settings.WeChatServiceCorpID,
		appInfo.Settings.WeChatServiceToken,
		appInfo.Settings.WeChatServiceEncodingAESKey,
		appInfo.Settings.WeChatServiceSecret,
		appInfo.Settings.WechatServiceLogo,
		appInfo.Settings.WechatServiceContainKeywords,
		appInfo.Settings.WechatServiceEqualKeywords,
	)
}

func (u *WechatServiceUsecase) getQAFunc(kbID string, appType domain.AppType) bot.GetQAFun {
	return func(ctx context.Context, msg string, info domain.ConversationInfo, ConversationID string) (chan string, error) {
		auth, err := u.authRepo.GetAuthBySourceType(ctx, domain.AppTypeWechatServiceBot.ToSourceType())
		if err != nil {
			u.logger.Error("get auth failed", log.Error(err))
			return nil, err
		}
		info.UserInfo.AuthUserID = auth.ID

		eventCh, err := u.chatUsecase.Chat(ctx, &domain.ChatRequest{
			Message:        msg,
			KBID:           kbID,
			AppType:        appType,
			RemoteIP:       "",
			ConversationID: ConversationID,
			Info:           info,
		})
		if err != nil {
			return nil, err
		}
		contentCh := make(chan string, 10)
		go func() {
			defer close(contentCh)
			for event := range eventCh {
				if event.Type == "done" || event.Type == "error" {
					break
				}
				if event.Type == "data" {
					contentCh <- event.Content
				}
			}
		}()
		return contentCh, nil
	}
}
