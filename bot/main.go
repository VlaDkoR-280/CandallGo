package main

import (
	"CandallGo/config"
	"CandallGo/internal/telegram"
	"CandallGo/logs"
	"context"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.LoadConfig()
	bot, err := telegram.NewBot(cfg.BotToken)
	if err != nil {
		logs.SendLog(logs.LogEntry{Level: "fatal", EventType: "system", Info: "Error creating bot", Error: err.Error()})
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		bot.Close()
		logs.SendLog(logs.LogEntry{Level: "info", EventType: "system", Info: "Bot is stopped"})

	}()

	logs.SendLog(logs.LogEntry{
		Level: "info", EventType: "system",
		Info: "Bot started",
	})
	bot.Start()

}
