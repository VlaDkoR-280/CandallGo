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
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	chId, err := strconv.ParseInt(os.Getenv("main_channel_id"), 10, 64)
	if err != nil {
		log.Println(err)
	}
	return Config{
		BotToken:  os.Getenv("BotToken"),
		DbUrl:     os.Getenv("database_url"),
		ChannelId: chId,
	}
}
