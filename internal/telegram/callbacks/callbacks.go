package callbacks

import (
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	"CandallGo/internal/static"
	"CandallGo/internal/telegram/payment"
	"CandallGo/logs"
	"container/list"
	"errors"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type data struct {
	api    *tgbotapi.BotAPI
	update tgbotapi.Update
	state  static.State
	conn   *db.DB
	loc    *localization.Local
}

func MainCallback(api *tgbotapi.BotAPI, update tgbotapi.Update, conn *db.DB, groupListCallback func(*tgbotapi.BotAPI, *db.DB, tgbotapi.Update, *localization.Local) error, loc *localization.Local) error {
	state, err := static.DecodeState(update.CallbackData())
	if err != nil {
		return err
	}
	callData := &data{
		api:    api,
		update: update,
		conn:   conn,
		state:  state,
		loc:    loc,
	}
	switch state.Action {
	case "delete":
		err = callData.deleteMsg()
	case "groups":
		_ = callData.deleteMsg()
		err = groupListCallback(api, conn, update, loc)
		if err == nil {
			go logs.SendLog(logs.LogEntry{
				Level:     "info",
				EventType: "user_callback",
				TgUserID:  strconv.FormatInt(callData.update.CallbackQuery.From.ID, 10),
				Msg:       callData.update.CallbackQuery.Data,
				Info:      "To groupListCallback",
			})
		}
	case "group":
		err = callData.deleteMsg()
		if err != nil {
			return err
		}
		err = callData.groupMsg()
		if err == nil {
			go logs.SendLog(logs.LogEntry{
				Level:     "info",
				EventType: "user_callback",
				TgUserID:  strconv.FormatInt(callData.update.CallbackQuery.From.ID, 10),
				Msg:       callData.update.CallbackQuery.Data,
				Info:      "To groupMsg",
			})
		}
	default:
		err = payment.PaymentCallback(api, update, state, conn, loc)
		if err == nil {
			go logs.SendLog(logs.LogEntry{
				Level:     "info",
				EventType: "user_callback",
				TgUserID:  strconv.FormatInt(callData.update.CallbackQuery.From.ID, 10),
				Msg:       callData.update.CallbackQuery.Data,
				Info:      "To PaymentCallback",
			})
		}
	}
	return err
}

func (callData data) deleteMsg() error {
	delMsg := tgbotapi.NewDeleteMessage(callData.update.CallbackQuery.Message.Chat.ID, callData.update.CallbackQuery.Message.MessageID)
	resApi, err := callData.api.Request(delMsg)
	if err != nil {
		return fmt.Errorf("DeleteMsgErr: %s | ApiResponse: %s", err.Error(), resApi.Description)
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
			userIsInGroup <- false
		}
		if groups.Len() < 0 {
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
	} else {
		msg := tgbotapi.NewMessage(callData.update.CallbackQuery.Message.Chat.ID, callData.loc.Get("ru", "post_group_empty"))
		_, err = callData.api.Send(msg)
		return err
	}
}

func (callData data) generateMsgForGroupData(groupData *db.GroupData) error {
	var isSub = groupData.SubDateEnd.Truncate(24 * time.Hour).After(time.Now().Truncate(24 * time.Hour))
	var statusSub string
	if isSub {
		statusSub = groupData.SubDateEnd.Truncate(24 * time.Hour).Format("02\\-01\\-2006")
	} else {
		statusSub = "\\-"
	}
	str := fmt.Sprintf(callData.loc.Get("ru", "group_info_text"), groupData.GroupName, statusSub)
	msg := tgbotapi.NewMessage(callData.update.CallbackQuery.Message.Chat.ID, str)
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	callbackDataBack, err := static.EncodeState(static.State{Action: "groups", Data: ""})
	if err != nil {
		return err
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.InlineKeyboardButton{
				Text: callData.loc.Get("ru", "button_back"), CallbackData: &callbackDataBack,
			}))

	//refundCallback, err := static.EncodeState(
	//	static.State{Action: "refund", Data: groupData.TgId})
	//if err != nil {
	//	return err
	//}

	//if isSub {
	//	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
	//		tgbotapi.NewInlineKeyboardRow(
	//			tgbotapi.InlineKeyboardButton{
	//				Text: "Вернуть деньги за подписку", CallbackData: &refundCallback,
	//			}))
	//}
	if !isSub {
		subCallback, err := static.EncodeState(
			static.State{Action: "subscribe_methods", Data: groupData.TgId})
		if err != nil {
			return err
		}

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.InlineKeyboardButton{
					Text: callData.loc.Get("ru", "group_info_button_sub"), CallbackData: &subCallback,
				}))
	}
	msg.ReplyMarkup = keyboard
	_, err = callData.api.Send(msg)
	return err
}
