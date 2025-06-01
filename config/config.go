package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	BotToken string
	DbUrl    string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return Config{
		BotToken: os.Getenv("BotToken"),
		DbUrl:    os.Getenv("database_url"),
	}
}
