package main

import (
	"context"
	"os"
)

func main() {
	// consumer 进程始终启用 worker，不受 compose 中 RUN_WORKER=0（API 侧）影响
	_ = os.Setenv("RUN_WORKER", "1")

	app, err := createApp()
	if err != nil {
		panic(err)
	}
	app.StatCronHandler.Start()
	if err := app.MQConsumer.StartConsumerHandlers(context.Background()); err != nil {
		panic(err)
	}
	if err := app.MQConsumer.Close(); err != nil {
		panic(err)
	}
}
