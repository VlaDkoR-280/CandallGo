package callbacks

import (
	"CandallGo/internal/db"
	"CandallGo/internal/static"
	"CandallGo/internal/telegram/payment"
	"container/list"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"time"
)

type data struct {
	api    *tgbotapi.BotAPI
	update tgbotapi.Update
	state  static.State
	conn   *db.DB
}

func MainCallback(api *tgbotapi.BotAPI, update tgbotapi.Update, conn *db.DB, groupListCallback func(*tgbotapi.BotAPI, *db.DB, tgbotapi.Update) error) error {
	state, err := static.DecodeState(update.CallbackData())
	if err != nil {
		return err
	}
	callData := &data{
		api:    api,
		update: update,
		conn:   conn,
		state:  state,
	}
	switch state.Action {
	case "delete":
		err = callData.deleteMsg()
	case "groups":
		err = callData.deleteMsg()
		return groupListCallback(api, conn, update)
	case "group":
		err = callData.groupMsg()
		if err != nil {
			return err
		}
		err = callData.deleteMsg()
	default:
		return payment.PaymentCallback(api, update, state, conn)
	}
	log.Println(state)
	return err
}

func (callData data) deleteMsg() error {
	delMsg := tgbotapi.NewDeleteMessage(callData.update.CallbackQuery.Message.Chat.ID, callData.update.CallbackQuery.Message.MessageID)
	resApi, err := callData.api.Request(delMsg)
	if err != nil {
		return errors.New(fmt.Sprintf("DeleteMsgErr: %s | ApiResponse: %s", err.Error(), resApi.Description))
	}
	return nil
}

func (callData data) groupMsg() error {
	if callData.state.Data == "" {
		return errors.New("state.Data is empty")
	}

	// проверяем имеет ли доступ пользователь к этой группе

	var userId = strconv.FormatInt(callData.update.CallbackQuery.From.ID, 10)
	var userIsInGroup = make(chan bool, 1)
	go func() {
		defer close(userIsInGroup)
		var groups list.List
		err := callData.conn.GetGroupsOfUser(userId, &groups, true)
		if err != nil {
			log.Println(err.Error())
			userIsInGroup <- false
		}
		if groups.Len() < 0 {
			log.Println(errors.New("groups is empty"))
			userIsInGroup <- false
		}
		var isExist = false
		for el := groups.Front(); el != nil; el = el.Next() {
			if el.Value.(db.GroupData).TgId == callData.state.Data {
				isExist = true
				break
			}
		}
		userIsInGroup <- isExist
	}()

	groupData, err := callData.conn.GetGroupData(callData.state.Data)
	if err != nil {
		return err
	}

	if <-userIsInGroup {
		return callData.generateMsgForGroupData(&groupData)
		//msg := tgbotapi.NewMessage(callData.update.CallbackQuery.Message.Chat.ID, "Данные о группе: "+groupData.GroupName)
		//_, err := callData.api.Send(msg)
		//return err
	} else {
		msg := tgbotapi.NewMessage(callData.update.CallbackQuery.Message.Chat.ID, "Вы уже не принадлежите выбранной группе, проверьте в каких группах вы есть /group")
		_, err = callData.api.Send(msg)
		return err
	}
	return nil

}

func (callData data) generateMsgForGroupData(groupData *db.GroupData) error {
	var isSub = groupData.SubDateEnd.Truncate(24 * time.Hour).After(time.Now().Truncate(24 * time.Hour))
	var statusSub string
	if isSub {
		statusSub = "Active"
	} else {
		statusSub = "Inactive"
	}
	str := fmt.Sprintf("*Название группы*: %s\n"+
		"*Подписка*: %s\n", groupData.GroupName, statusSub)
	msg := tgbotapi.NewMessage(callData.update.CallbackQuery.Message.Chat.ID, str)
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	callbackDataBack, err := static.EncodeState(static.State{Action: "groups", Data: ""})
	if err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.InlineKeyboardButton{
				Text: "К списку групп", CallbackData: &callbackDataBack,
			}))

	refundCallback, err := static.EncodeState(
		static.State{Action: "refund", Data: groupData.TgId})
	if err != nil {
		return err
	}

	if isSub {
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.InlineKeyboardButton{
					Text: "Вернуть деньги за подписку", CallbackData: &refundCallback,
				}))
	} else {
		subCallback, err := static.EncodeState(
			static.State{Action: "subscribe_methods", Data: groupData.TgId})
		if err != nil {
			return err
		}

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.InlineKeyboardButton{
					Text: "Подписки", CallbackData: &subCallback,
				}))
	}
	msg.ReplyMarkup = keyboard
	_, err = callData.api.Send(msg)
	return err
}
