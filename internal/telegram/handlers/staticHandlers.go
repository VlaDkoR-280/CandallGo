package handlers

import (
	"CandallGo/internal/db"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	api    *tgbotapi.BotAPI
	conn   *db.DB
	update tgbotapi.Update
}
