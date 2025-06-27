package handlers

import (
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	"CandallGo/internal/static"
	"container/list"
	"errors"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
)

func PrivateHandler(api *tgbotapi.BotAPI, conn *db.DB, update tgbotapi.Update, loc *localization.Local) error {

	var handler = Handler{api: api, conn: conn, update: update, loc: loc}
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
	case "subs":
		text := fmt.Sprintf(loc.Get("ru", "subscribes_info"), "hello")
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, err := handler.api.Send(msg)
		return err
	}
	return nil
}

func (handler *Handler) startPrivateCommand() error {
	text := handler.loc.Get("ru", "start")
	jsonMenu := `{
        "type": "web_app",
        "text": "Группы",
        "web_app": {
            "url": "https://candall.ru"
        }
    }`
	params := tgbotapi.Params{
		"menu_button": jsonMenu,
	}
	if _, err := handler.api.MakeRequest("setChatMenuButton", params); err != nil {
		return err
	}
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
		msg := tgbotapi.NewMessage(mainChatId, handler.loc.Get("ru", "group_list_empty"))
		_, err = handler.api.Send(msg)
		return err
	}
	deleteState, err := static.EncodeState(static.State{Action: "delete", Data: ""})
	if err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.InlineKeyboardButton{
		Text: handler.loc.Get("ru", "button_back"), CallbackData: &deleteState,
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

	msg := tgbotapi.NewMessage(mainChatId, handler.loc.Get("ru", "groups_list"))
	msg.ReplyMarkup = keyboard
	_, err = handler.api.Send(msg)
	return err

}
