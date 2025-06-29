package handlers

import (
	"CandallGo/internal/localization"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MessageOfNeedChangeStatus(api *tgbotapi.BotAPI, update tgbotapi.Update, loc *localization.Local) error {
	text := loc.Get("ru", "change_group_status")
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err := api.Send(msg)
	if err != nil {
		return err
	}

	return leaveFromTheGroup(api, update.Message.Chat.ID)
}

func leaveFromTheGroup(api *tgbotapi.BotAPI, chatId int64) error {
	leaveConfig := tgbotapi.LeaveChatConfig{
		ChatID: chatId,
	}
	_, err := api.Request(leaveConfig)
	return err
}
