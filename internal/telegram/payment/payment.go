package payment

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Provider struct {
	Id       int64
	Name     string
	Token    string
	Currency string
}

func PaymentCallback(bot *tgbotapi.BotAPI, callback tgbotapi.CallbackQuery) error {
	
	return nil
}
