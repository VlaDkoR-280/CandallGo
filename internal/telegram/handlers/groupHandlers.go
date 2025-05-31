package handlers

import (
	"CandallGo/internal/db"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GroupHandler(api *tgbotapi.BotAPI, conn *db.DB, update tgbotapi.Update) error {
	var handler = Handler{api: api, conn: conn, update: update}
	msg := update.Message
	//userId := msg.From.ID
	//chatId := msg.Chat.ID

	switch msg.Command() {
	case "start":
		_ = handler.startCommand()
	}
	return nil
}

func (handler *Handler) startCommand() error {
	msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, "START")
	_, _ = handler.api.Send(msg)
	return nil
}
