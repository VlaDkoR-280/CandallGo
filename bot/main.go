package main

import (
	"CandallGo/config"
	"CandallGo/internal/telegram"
	"log"
)

func main() {
	cfg := config.LoadConfig()
	bot, err := telegram.NewBot(cfg.BotToken)
	if err != nil {
		log.Fatal("Bot connect Error", err)
	}
	defer bot.Close()
	bot.Start()

}
