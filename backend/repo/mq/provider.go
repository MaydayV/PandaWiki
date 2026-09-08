package mq

import (
	"github.com/google/wire"

	"github.com/chaitin/panda-wiki/repo/cache"
)

var ProviderSet = wire.NewSet(
	cache.ProviderSet,
	NewRAGRepository,
)
