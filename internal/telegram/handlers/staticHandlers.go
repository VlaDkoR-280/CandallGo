package handlers

import (
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	api    *tgbotapi.BotAPI
	conn   *db.DB
	update tgbotapi.Update
	loc    *localization.Local
}
