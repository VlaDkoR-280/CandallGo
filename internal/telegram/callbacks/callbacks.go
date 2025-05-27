package callbacks

import (
	"encoding/json"
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strings"
)

type MyCallback struct {
	Action  string `json:"action"`
	Id      int64  `json:"id"`
	GroupId string `json:"group_id"`
}

func Callback(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	callback := update.CallbackQuery
	var parsed MyCallback
	err := json.Unmarshal([]byte(callback.Data), &parsed)
	if err != nil && !strings.Contains(err.Error(), "cannot unmarshal") {
		return err
	}

	switch parsed.Action {
	case "delete":
		return deleteCallback(bot, *callback)
	}
	return errors.New("Error of callback Action: " + parsed.Action)

}

func deleteCallback(bot *tgbotapi.BotAPI, callback tgbotapi.CallbackQuery) error {
	deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	_, err := bot.Request(deleteMsg)

	return err

}
