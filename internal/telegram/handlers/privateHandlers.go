package handlers

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"CandallGo/internal/telegram/callbacks"
	_ "CandallGo/internal/telegram/callbacks"
	"container/list"
	"encoding/json"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
)

func PrivateStart(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Сообщение в личном чате")
	_, err := bot.Send(msg)
	return err

}

func PrivateGetGroups(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	var listGroups list.List
	db_url := config.LoadConfig().DbUrl
	database, err := db.Connect(db_url)
	if err != nil {
		return err
	}
	defer database.Close()

	userId := strconv.FormatInt(update.Message.From.ID, 10)
	groupId := strconv.FormatInt(update.Message.Chat.ID, 10)
	err = database.GetGroupsOfUser(userId, groupId, &listGroups)
	if err != nil {
		return err
	}
	if listGroups.Len() <= 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы добавлены ни в одну группу. Если вы добавили бота в группу, но здесь всё еще пусто, воспользуйтесь /help")
		_, err := bot.Send(msg)
		return err
	}
	deleteCallback, err := getStr(callbacks.MyCallback{
		Action: "delete",
	})
	if err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.InlineKeyboardButton{
			Text: "Закрыть", CallbackData: &deleteCallback,
		},
	))
	for el := listGroups.Front(); el != nil; el = el.Next() {
		groupData := el.Value.(db.GroupData)
		if groupData.Name != "" && groupData.Tg_id != "" {
			selectGroupCallback, err := getStr(
				callbacks.MyCallback{Action: "group", GroupId: groupData.Tg_id},
			)
			if err != nil {
				return err
			}

			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.InlineKeyboardButton{Text: groupData.Name, CallbackData: &selectGroupCallback}))
		}

	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "<UNK> <UNK> <UNK>")
	msg.ReplyMarkup = keyboard
	_, err = bot.Send(msg)
	return err

}

func getStr(callback callbacks.MyCallback) (string, error) {
	callbackJSON, err := json.Marshal(callback)
	if err != nil {
		return "", err
	}
	callbackString := string(callbackJSON)
	return callbackString, nil
}
