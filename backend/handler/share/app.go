package share

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	wechat_v2 "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/usecase"
)

type ShareAppHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	usecase *usecase.AppUsecase
}

func NewShareAppHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	usecase *usecase.AppUsecase,
) *ShareAppHandler {
	h := &ShareAppHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.share.app"),
		usecase:     usecase,
	}

	share := e.Group("share/v1/app",
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Access-Control-Allow-Origin", "*")
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept")
				if c.Request().Method == "OPTIONS" {
					return c.NoContent(http.StatusOK)
				}
				return next(c)
			}
		})
	share.GET("/web/info", h.GetWebAppInfo)
	share.GET("/widget/info", h.GetWidgetAppInfo)
	share.GET("/wechat/info", h.WechatAppInfo)

	// wechat official account
	share.GET("/wechat/official_account", h.VerifyUrlWechatOfficialAccount)
	share.POST("/wechat/official_account", h.WechatHandlerOfficialAccount)
	return h
}

// GetWebAppInfo
//
//	@Summary		GetAppInfo
//	@Description	GetAppInfo
//	@Tags			share_app
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string	true	"kb id"
//	@Success		200		{object}	domain.Response{data=domain.AppInfoResp}
//	@Router			/share/v1/app/web/info [get]
func (h *ShareAppHandler) GetWebAppInfo(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	ctx := context.WithValue(c.Request().Context(), consts.ContextKeyEdition, consts.GetLicenseEdition(c))
	appInfo, err := h.usecase.ShareGetWebAppInfo(ctx, kbID, domain.GetAuthID(c))
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, appInfo)
}

// GetWidgetAppInfo
//
//	@Summary		GetWidgetAppInfo
//	@Description	GetWidgetAppInfo
//	@Tags			share_app
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string	true	"kb id"
//	@Success		200		{object}	domain.Response
//	@Router			/share/v1/app/widget/info [get]
func (h *ShareAppHandler) GetWidgetAppInfo(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	appInfo, err := h.usecase.GetWidgetAppInfo(c.Request().Context(), kbID)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, appInfo)
}

// WechatAppInfo
//
//	@Summary		WechatAppInfo
//	@Description	WechatAppInfo
//	@Tags			share_chat
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string	true	"kb id"
//	@Success		200		{object}	domain.Response{data=v1.WechatAppInfoResp}
//	@Router			/share/v1/app/wechat/info [get]
func (h *ShareAppHandler) WechatAppInfo(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")
	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	appInfo, err := h.usecase.GetWechatAppInfo(c.Request().Context(), kbID)
	if err != nil {
		return h.NewResponseWithError(c, err.Error(), err)
	}
	return h.NewResponseWithData(c, appInfo)
}

func (h *ShareAppHandler) VerifyUrlWechatOfficialAccount(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")

	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	ctx := c.Request().Context()

	// get wechat official account info
	appInfo, err := h.usecase.GetAppDetailByKBIDAndAppType(ctx, kbID, domain.AppTypeWechatOfficialAccount)
	if err != nil {
		h.logger.Error("get app detail failed")
		return h.NewResponseWithError(c, "GetAppDetailByKBIDAndAppType failed", err)
	}

	if appInfo.Settings.WechatOfficialAccountIsEnabled != nil && !*appInfo.Settings.WechatOfficialAccountIsEnabled {
		return h.NewResponseWithError(c, "wechat official account is not enabled", err)
	}
	wc := wechat_v2.NewWechat()
	memory := cache.NewMemory()
	cfg := &offConfig.Config{
		AppID:          appInfo.Settings.WechatOfficialAccountAppID,
		AppSecret:      appInfo.Settings.WechatOfficialAccountAppSecret,
		Token:          appInfo.Settings.WechatOfficialAccountToken,
		EncodingAESKey: appInfo.Settings.WechatOfficialAccountEncodingAESKey,
		Cache:          memory,
	}
	officialAccount := wc.GetOfficialAccount(cfg)
	server := officialAccount.GetServer(c.Request(), c.Response().Writer)

	// success
	err = server.Serve()
	if err != nil {
		return h.NewResponseWithError(c, "serve message failed", err)
	}
	return nil
}

func (h *ShareAppHandler) WechatHandlerOfficialAccount(c echo.Context) error {
	kbID := c.Request().Header.Get("X-KB-ID")

	if kbID == "" {
		return h.NewResponseWithError(c, "kb_id is required", nil)
	}
	ctx := c.Request().Context()

	// get wechat official account info
	appInfo, err := h.usecase.GetAppDetailByKBIDAndAppType(ctx, kbID, domain.AppTypeWechatOfficialAccount)
	if err != nil {
		h.logger.Error("get app detail failed")
		return h.NewResponseWithError(c, "GetAppDetailByKBIDAndAppType failed", err)
	}

	if appInfo.Settings.WechatOfficialAccountIsEnabled != nil && !*appInfo.Settings.WechatOfficialAccountIsEnabled {
		return h.NewResponseWithError(c, "wechat official account is not enabled", err)
	}
	wc := wechat_v2.NewWechat()
	memory := cache.NewMemory()
	cfg := &offConfig.Config{
		AppID:          appInfo.Settings.WechatOfficialAccountAppID,
		AppSecret:      appInfo.Settings.WechatOfficialAccountAppSecret,
		Token:          appInfo.Settings.WechatOfficialAccountToken,
		EncodingAESKey: appInfo.Settings.WechatOfficialAccountEncodingAESKey,
		Cache:          memory,
	}
	officialAccount := wc.GetOfficialAccount(cfg)
	server := officialAccount.GetServer(c.Request(), c.Response().Writer)

	// message handler
	server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {
		h.logger.Debug("received message:", log.Any("msgtype", msg.MsgType), log.Any("fromUserName", msg.FromUserName), log.String("content", msg.Content), log.Any("event type", msg.Event))

		switch msg.MsgType {
		case message.MsgTypeText:
			// text消息
			userOpenID := msg.FromUserName
			userContent := msg.Content
			h.logger.Debug("user_open_id user_content", log.Any("user_open_id", userOpenID), log.Any("user content", userContent))
			// 异步发送
			go func(openID, content string) {
				// 裸 goroutine 内 panic 会击穿整个进程（echo 只 recover handler 本体）
				defer func() {
					if r := recover(); r != nil {
						h.logger.Error("official account async handler panicked", log.Any("panic", r))
					}
				}()
				ctx := context.Background()
				// send content to ai
				result, err := h.usecase.GetWechatOfficialAccountResponse(ctx, officialAccount, kbID, openID, content)
				if err != nil {
					h.logger.Error("get wechat official account response failed", log.Error(err))
					return
				}
				// send response to user --> 需要开启客服消息权限
				err = h.usecase.SendCustomerServiceMessage(officialAccount, string(userOpenID), result)
				if err != nil {
					h.logger.Error("send to customer service failed", log.Error(err))
				}
			}(string(userOpenID), userContent)
			return &message.Reply{MsgType: message.MsgTypeText, MsgData: message.NewText("您的问题已经收到，正在努力思考中，请稍候...")}
		case message.MsgTypeEvent:
			if msg.Event == message.EventSubscribe {
				return &message.Reply{MsgType: message.MsgTypeText, MsgData: message.NewText("感谢关注,欢迎提问！")} // 立即回复简单信息
			}
			return nil
		default:
			h.logger.Info("unknown message type", log.Any("message type", msg.MsgType))
			return &message.Reply{MsgType: message.MsgTypeText, MsgData: message.NewText("未知消息类型，请发送正确的类型...")}
		}
	})

	// success
	err = server.Serve()
	if err != nil {
		h.logger.Error("serve message failed", log.Error(err))
		return h.NewResponseWithError(c, "serve message failed", err)
	}

	// send message to user
	err = server.Send()
	if err != nil {
		h.logger.Error("send message failed", log.Error(err))
		return h.NewResponseWithError(c, "send message failed", err)
	}
	return nil
}
