package main

import (
	"CandallGo/config"
	"CandallGo/internal/telegram"
	"CandallGo/logs"
)

func main() {
	cfg := config.LoadConfig()
	bot, err := telegram.NewBot(cfg.BotToken)
	if err != nil {
		logs.SendLog(logs.LogEntry{Level: "fatal", EventType: "system", Msg: "Error creating bot", Error: err.Error()})
		return
	}
	defer bot.Close()
	logs.SendLog(logs.LogEntry{
		Level: "info", EventType: "system",
		Msg: "Bot started",
	})
	bot.Start()

}
