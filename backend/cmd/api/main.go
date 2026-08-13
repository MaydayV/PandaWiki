package main

import (
	"context"
	"fmt"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/server/http"
	"github.com/chaitin/panda-wiki/setup"
)

// @title panda-wiki API
// @version 1.0
// @description panda-wiki API documentation
// @BasePath /
// @securityDefinitions.apikey	bearerAuth
// @in	header
// @name	Authorization
// @description	Type "Bearer" + a space + your token to authorize
func main() {
	app, err := createApp()
	if err != nil {
		panic(err)
	}
	if err := setup.CheckInitCert(); err != nil {
		panic(err)
	}
	if app.Config.RunWorker {
		app.Logger.Info("embedded worker enabled (RUN_WORKER=1)")
		app.StatCronHandler.Start()
		workerCtx, workerCancel := context.WithCancel(context.Background())
		defer workerCancel()
		go func() {
			if err := app.MQConsumer.StartConsumerHandlers(workerCtx); err != nil {
				app.Logger.Error("mq consumer stopped", log.Error(err))
			}
		}()
	} else {
		app.Logger.Info("embedded worker disabled (RUN_WORKER=0); use consumer service or set RUN_WORKER=1")
	}

	go func() {
		if err := http.StartAdminServer(app.Config, app.Logger); err != nil {
			app.Logger.Error("admin server stopped", log.Error(err))
		}
	}()

	port := app.Config.HTTP.Port
	app.Logger.Info(fmt.Sprintf("Starting server on port %d", port))
	app.HTTPServer.Echo.Logger.Fatal(app.HTTPServer.Echo.Start(fmt.Sprintf(":%d", port)))
}
