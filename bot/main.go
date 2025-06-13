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
		logs.SendLog("error", err.Error())
		return
		//log.Fatal("Bot connect Error", err)
	}
	defer bot.Close()
	logs.SendLog("info", "bot has been started")
	bot.Start()

}
