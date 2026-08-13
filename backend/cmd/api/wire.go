//go:build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/chaitin/panda-wiki/config"
	mqhandler "github.com/chaitin/panda-wiki/handler/mq"
	share "github.com/chaitin/panda-wiki/handler/share"
	v1 "github.com/chaitin/panda-wiki/handler/v1"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/mq"
	"github.com/chaitin/panda-wiki/server/http"
	"github.com/chaitin/panda-wiki/telemetry"
)

func createApp() (*App, error) {
	wire.Build(
		wire.Struct(new(App), "*"),
		wire.NewSet(
			config.ProviderSet,
			log.ProviderSet,
			telemetry.ProviderSet,

			http.ProviderSet,
			v1.ProviderSet,
			share.ProviderSet,

			mqhandler.NewRAGMQHandler,
			mqhandler.NewRAGJobWorker,
			mqhandler.NewCronHandler,
		),
	)
	return &App{}, nil
}

type App struct {
	HTTPServer          *http.HTTPServer
	Handlers            *v1.APIHandlers
	ShareHandlers       *share.ShareHandler
	Config              *config.Config
	Logger              *log.Logger
	Telemetry           *telemetry.Client
	MQConsumer          mq.MQConsumer
	RAGMQHandler    *mqhandler.RAGMQHandler
	RAGJobWorker    *mqhandler.RAGJobWorker
	StatCronHandler *mqhandler.CronHandler
}
