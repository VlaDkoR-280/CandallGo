package callbacks

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"CandallGo/internal/static"
	"container/list"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
	"time"
)

//type MyCallback struct {
//	Action  string `json:"action"`
//	Id      int64  `json:"id"`
//	GroupId string `json:"group_id"`
//}

func Callback(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	callback := update.CallbackQuery
	parsed, err := static.DecodeState(callback.Data)
	if err != nil && !strings.Contains(err.Error(), "cannot unmarshal") {
		return err
	}
	if parsed.Data == "" {
		return errors.New("GroupId from callback is empty")
	}
	switch parsed.Action {
	case "delete":
		return deleteCallback(bot, *callback)
	case "groups":
		return groupInfo(bot, *callback, parsed)
	case "refund":
		return nil
	case "list_subs":
		return nil
	case "list_types_of_sub":
		return nil
	}

	return errors.New("Error of callback Action: " + parsed.Action)

}

func deleteCallback(bot *tgbotapi.BotAPI, callback tgbotapi.CallbackQuery) error {
	deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	_, err := bot.Request(deleteMsg)

	return err

}

func groupInfo(bot *tgbotapi.BotAPI, callback tgbotapi.CallbackQuery, state static.State) error {

	// проверка прав доступа
	db_url := config.LoadConfig().DbUrl
	database, err := db.Connect(db_url)
	if err != nil {
		return err
	}
	defer database.Close()

	var groupList list.List
	err = database.GetGroupsOfUser(strconv.FormatInt(callback.From.ID, 10), state.Data, &groupList)
	if err != nil {
		return err
	}

	checkPermission := func(mList list.List, value string) bool {
		for el := mList.Front(); el != nil; el = el.Next() {
			if el.Value.(db.GroupData).Tg_id == value {
				return true
			}
		}
		return false
	}

	if !checkPermission(groupList, state.Data) {
		return errors.New("User does not have permission")
	}

	type mData struct {
		GroupId      string `json:"group_id"`
		Name         string
		DateOfEndSub time.Time
	}
	var groupData mData

	err = database.GetGroupInfo(state.Data, &groupData.GroupId, &groupData.Name, &groupData.DateOfEndSub)
	if err != nil {
		return err
	}
	var statusSub string
	if groupData.DateOfEndSub.After(time.Now()) {
		statusSub = "OK"
	} else {
		statusSub = "NO"
	}
	text := fmt.Sprintf("*Название группы*: %s\n*Статус подписки*: %s", groupData.Name, statusSub)
	msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	var finalText string
	var finalCallback string
	if statusSub == "OK" {
		finalText = "Запросить возврат средств"
		finalCallback, err = static.EncodeState(static.State{
			Action: "refund",
			Data:   groupData.GroupId,
		})
	} else if statusSub == "NO" {
		finalText = "Список подписок"
		finalCallback, err = static.EncodeState(static.State{
			Action: "list_of_sub",
			Data:   groupData.GroupId,
		})
	} else {
		return errors.New(fmt.Sprintf("Failed statusSub {%s} ", statusSub))
	}
	if err != nil {
		return err
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(
		tgbotapi.InlineKeyboardButton{Text: finalText, CallbackData: &finalCallback},
	))

	msg.ReplyMarkup = keyboard
	_, err = bot.Send(msg)
	if err != nil {
		return err
	}
	deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	_, err = bot.Request(deleteMsg)
	return err
}
