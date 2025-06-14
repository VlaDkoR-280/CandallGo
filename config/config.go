package config

import (
	"CandallGo/logs"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
)

type Config struct {
	BotToken  string
	DbUrl     string
	ChannelId int64
	LogUrl    string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		logs.SendLog(logs.LogEntry{
			Level:     "error",
			EventType: "system",
			Error:     fmt.Sprintf("%s\n%s", "Error loading .env file", err.Error()),
		})
		os.Exit(1)
	}
	chId, err := strconv.ParseInt(os.Getenv("main_channel_id"), 10, 64)
	if err != nil {
		logs.SendLog(logs.LogEntry{
			Level:     "error",
			EventType: "system",
			Error:     fmt.Sprintf("%s\n%s", "Error parse main_channel_id", err.Error()),
		})
	}
	return Config{
		BotToken:  os.Getenv("BotToken"),
		DbUrl:     os.Getenv("database_url"),
		ChannelId: chId,
		LogUrl:    os.Getenv("log_url"),
	}
}
