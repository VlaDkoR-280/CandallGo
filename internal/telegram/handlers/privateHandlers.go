package handlers

import (
	"CandallGo/internal/db"
	"CandallGo/internal/static"
	"container/list"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
)

func PrivateHandler(api *tgbotapi.BotAPI, conn *db.DB, update tgbotapi.Update) error {

	var handler = Handler{api: api, conn: conn, update: update}
	if update.CallbackQuery != nil {
		state, err := static.DecodeState(update.CallbackData())
		if err != nil {
			return err
		}
		switch state.Action {
		case "groups":
			return handler.groupsHandler()
		}
	}
	switch update.Message.Command() {
	case "start":
		return handler.startPrivateCommand()
	case "groups":
		return handler.groupsHandler()
	}
	return nil
}

func (handler *Handler) startPrivateCommand() error {
	text := fmt.Sprintf("Привет, я бот для тега участников в группе\\!\n" +
		"Я работаю внутри групп с помощью команды /all\n" +
		"Эту команду ты можешь использовать один раз в день\n\n" +
		"Также ты можешь купить подписку для группы, для этого сначала выбери группу с помощью команды /groups " +
		"после чего выбери способ оплаты и подписку:\n\n" +
		"\\- *Подписка на неделю* \\- _предоставляет безлимитное использование бота в группе, для которой куплена подписка_\n\n" +
		"\\- *Подписка на месяц* \\- _предоставляет безлимитное использование бота в группе, для которой куплена подписка_\n\n" +
		"\\- *Подписка на неделю* \\- _предоставляет безлимитное использование бота в группе, для которой куплена подписка_\n\n")
	msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err := handler.api.Send(msg)
	return err
}

func (handler *Handler) groupsHandler() error {
	var groups list.List

	var mainChatId int64
	if handler.update.Message != nil {
		mainChatId = handler.update.Message.From.ID
	} else if handler.update.CallbackQuery != nil {
		mainChatId = handler.update.CallbackQuery.From.ID
	} else {
		return errors.New("Message and callbackQuery both are nil")
	}
	userId := strconv.FormatInt(mainChatId, 10)
	err := handler.conn.GetGroupsOfUser(userId, &groups, true)
	if err != nil {
		return err
	}
	if groups.Len() <= 0 {
		msg := tgbotapi.NewMessage(mainChatId, "Я не знаю ни одной группы с вами, проверьте есть ли я в группе, даны ли мне права администротора"+
			"писали ли вы в группе после того, как меня добавили")
		_, err = handler.api.Send(msg)
		return err
	}
	deleteState, err := static.EncodeState(static.State{Action: "delete", Data: ""})
	if err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.InlineKeyboardButton{
		Text: "<< Назад", CallbackData: &deleteState,
	}))

	for el := groups.Front(); el != nil; el = el.Next() {
		group := el.Value.(db.GroupData)
		callbackGroup, err := static.EncodeState(static.State{Action: "group", Data: group.TgId})
		if err != nil {
			return err
		}
		var keyRow = tgbotapi.NewInlineKeyboardRow(tgbotapi.InlineKeyboardButton{
			Text: group.GroupName, CallbackData: &callbackGroup,
		})

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyRow)
	}

	msg := tgbotapi.NewMessage(mainChatId, "Выберите группу для просмотра информации о ней")
	msg.ReplyMarkup = keyboard
	_, err = handler.api.Send(msg)
	return err

}
