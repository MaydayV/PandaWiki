package share

import (
	"github.com/google/wire"

	"github.com/chaitin/panda-wiki/pkg/captcha"
)

type ShareHandler struct {
	ShareNodeHandler         *ShareNodeHandler
	ShareAppHandler          *ShareAppHandler
	ShareChatHandler         *ShareChatHandler
	ShareSitemapHandler      *ShareSitemapHandler
	ShareStatHandler         *ShareStatHandler
	ShareCommentHandler      *ShareCommentHandler
	ShareAuthHandler         *ShareAuthHandler
	ShareConversationHandler *ShareConversationHandler
	ShareWechatHandler       *ShareWechatHandler
	ShareCaptchaHandler      *ShareCaptchaHandler
	ShareMCPHandler          *ShareMCPHandler
	OpenapiV1Handler         *OpenapiV1Handler
	ShareCommonHandler       *ShareCommonHandler
}

var ProviderSet = wire.NewSet(
	captcha.NewCaptcha,

	NewShareNodeHandler,
	NewShareAppHandler,
	NewShareChatHandler,
	NewShareSitemapHandler,
	NewShareStatHandler,
	NewShareCommentHandler,
	NewShareAuthHandler,
	NewShareConversationHandler,
	NewShareWechatHandler,
	NewShareCaptchaHandler,
	NewShareMCPHandler,
	NewShareCommonHandler,
	NewOpenapiV1Handler,

	wire.Struct(new(ShareHandler), "*"),
)
