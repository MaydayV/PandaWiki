package pg

import (
	"github.com/google/wire"

	"github.com/chaitin/panda-wiki/store/pg"
	"github.com/chaitin/panda-wiki/store/rag"
)

var ProviderSet = wire.NewSet(
	pg.ProviderSet,

	NewNodeRepository,
	NewAppRepository,
	NewConversationRepository,
	NewUserRepository,
	NewUserAccessRepository,
	NewModelRepository,
	NewKnowledgeBaseRepository,
	NewStatRepository,
	NewCommentRepository,
	NewPromptRepo,
	NewBlockWordRepo,
	NewAuthRepo,
	NewWechatRepository,
	NewAPITokenRepo,
	NewAPICallAuditRepo,
	NewSystemSettingRepo,
	NewMCPRepository,
	NewNavRepository,

	wire.Bind(new(rag.ModelProvider), new(*ModelRepository)),
)
