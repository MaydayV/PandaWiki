package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chaitin/panda-wiki/log"
	pwhttp "github.com/chaitin/panda-wiki/server/http"
	"github.com/chaitin/panda-wiki/setup"
	"github.com/chaitin/panda-wiki/utils"
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

	// 注入凭据落库加密密钥（API token / SimpleAuth 口令；未配置 = 明文兼容）
	utils.SetEncryptionKey(app.Config.EncryptionKey)

	// SIGTERM/SIGINT 触发优雅退出（docker stop 会先发 SIGTERM）
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if app.Config.RunWorker {
		app.Logger.Info("embedded worker enabled (RUN_WORKER=1)")
		app.StatCronHandler.Start()
		// workerCtx 派生自信号 ctx：SIGTERM 后自动取消，worker 走正常退出路径
		workerCtx, workerCancel := context.WithCancel(ctx)
		defer workerCancel()
		go app.RAGJobWorker.Start(workerCtx)
	} else {
		app.Logger.Info("embedded worker disabled (RUN_WORKER=0)")
	}

	go func() {
		if err := pwhttp.StartAdminServer(app.Config, app.Logger); err != nil {
			app.Logger.Error("admin server stopped", log.Error(err))
		}
	}()

	port := app.Config.HTTP.Port
	app.Logger.Info(fmt.Sprintf("Starting server on port %d", port))
	// 不设 WriteTimeout：会切断 SSE 长流响应；ReadHeaderTimeout/IdleTimeout 防 slowloris
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           app.HTTPServer.Echo,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		<-ctx.Done()
		app.Logger.Info("shutting down server...")
		// worker ctx 派生自信号 ctx，已自动取消
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			app.Logger.Error("server shutdown failed", log.Error(err))
		}
	}()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		app.HTTPServer.Echo.Logger.Fatal(err)
	}
	app.Logger.Info("server exited gracefully")
}
