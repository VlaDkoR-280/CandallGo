package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

type Config struct {
	BotToken  string
	DbUrl     string
	ChannelId int64
	LogUrl    string
	WebAppUrl string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	chId, err := strconv.ParseInt(os.Getenv("main_channel_id"), 10, 64)
	if err != nil {
		log.Println(err)
		chId = -1
	}
	return Config{
		BotToken:  os.Getenv("BotToken"),
		DbUrl:     os.Getenv("database_url"),
		ChannelId: chId,
		LogUrl:    os.Getenv("log_url"),
		WebAppUrl: os.Getenv("web_app_url"),
	}
}
