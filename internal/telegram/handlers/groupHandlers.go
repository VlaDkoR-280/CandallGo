package handlers

import (
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	"CandallGo/logs"
	"container/list"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"strings"
	"time"
)

func GroupHandler(api *tgbotapi.BotAPI, conn *db.DB, update tgbotapi.Update, loc *localization.Local) error {
	var handler = Handler{api: api, conn: conn, update: update, loc: loc}
	msg := update.Message
	//userId := msg.From.ID
	//chatId := msg.Chat.ID
	if update.Message.NewChatMembers != nil {
		return handler.newMember()
	} else if update.Message.LeftChatMember != nil {
		return handler.leftMember()
	}

	switch msg.Command() {
	case "start":
		return handler.startCommand()
	case "all":
		return handler.allCommand()
	}
	return nil
}

func (handler *Handler) newMember() error {
	var ch1 = make(chan struct{}, 5)
	for _, el := range handler.update.Message.NewChatMembers {
		ch1 <- struct{}{}
		go func() {
			defer func() { <-ch1 }()
			var userId = strconv.FormatInt(el.ID, 10)
			var groupId = strconv.FormatInt(handler.update.Message.Chat.ID, 10)
			bot, err := handler.api.GetMe()
			if err != nil {
				logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "telegram",
					Info:      fmt.Sprintf("%s\n%s", "Error api GetMe", err.Error()),
					TgUserID:  userId,
					TgGroupID: groupId,
				})
				return
			}
			if userId == strconv.FormatInt(bot.ID, 10) {
				err = handler.startCommand()
				if err != nil {
					go logs.SendLog(logs.LogEntry{
						Level:     "error",
						EventType: "telegram",
						Info:      fmt.Sprintf("%s\n%s", "Error api startCommand", err.Error()),
						TgUserID:  userId,
						TgGroupID: groupId,
					})
				}
				return
			}

			if el.IsBot {
				return
			}

			_, err = handler.conn.GetUserData(userId)
			if err != nil {
				if !strings.Contains(err.Error(), "no rows in result") {
					go logs.SendLog(logs.LogEntry{
						Level:     "error",
						EventType: "data_base",
						Info:      fmt.Sprintf("%s\n%s", "Error GetUserData", err.Error()),
						TgUserID:  userId,
						TgGroupID: groupId,
					})
					return
				}

				err = handler.conn.AddUser(userId)
				if err != nil {
					go logs.SendLog(logs.LogEntry{
						Level:     "error",
						EventType: "data_base",
						Info:      fmt.Sprintf("%s\n%s", "Error GetUserData", err.Error()),
						TgUserID:  userId,
						TgGroupID: groupId,
					})
				}
			}
			err = handler.conn.AddUserToGroup(userId, groupId)
			if err != nil {
				go logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "data_base",
					Info:      fmt.Sprintf("%s\n%s", "Error AddUserToGroup", err.Error()),
					TgUserID:  userId,
					TgGroupID: groupId,
				})
			}
			go logs.SendLog(logs.LogEntry{
				Level:     "info",
				EventType: "data_base",
				Info:      "AddUserToGroup",
				TgUserID:  userId,
				TgGroupID: groupId,
			})
		}()

	}

	return nil
}

func (handler *Handler) leftMember() error {
	var userId = strconv.FormatInt(handler.update.Message.LeftChatMember.ID, 10)
	var groupId = strconv.FormatInt(handler.update.Message.Chat.ID, 10)

	var bot, err = handler.api.GetMe()
	if err != nil {
		return err
	}
	if strconv.FormatInt(bot.ID, 10) == userId {
		err = handler.conn.RemoveLinkUsersWithGroup(groupId)
		return err
	}

	if handler.update.Message.LeftChatMember.IsBot {
		return nil
	}

	err = handler.conn.RemoveLinkUserWithGroup(userId, groupId)
	if err == nil {
		go logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "data_base",
			Info:      "RemoveLinkUserWithGroup",
			TgUserID:  userId,
			TgGroupID: groupId,
		})
	}
	return err
}

func (handler *Handler) startCommand() error {
	text := handler.loc.Get("ru", "group_start")
	msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, text)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err := handler.api.Send(msg)
	return err
}

func (handler *Handler) allCommand() error {
	var userList list.List
	var groupId = strconv.FormatInt(handler.update.Message.Chat.ID, 10)

	groupData, err := handler.conn.GetGroupData(groupId)
	if err != nil {
		return err
	}
	var canTag = make(chan bool, 1)
	var isSub = make(chan bool, 1)
	go func() {
		dateUse := groupData.DateLastUse.Truncate(24 * time.Hour)
		dateNow := time.Now().Truncate(24 * time.Hour)
		if dateUse.Equal(dateNow) {
			canTag <- false
		}
		if err := handler.conn.UpdateGroupData(db.GroupData{DateLastUse: dateNow, TgId: groupId}); err != nil {
			go logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "data_base",
				Info:      "UpdateGroupData",
				TgGroupID: groupId,
			})
			canTag <- false
			return
		}
		canTag <- true
	}()
	go func() {
		if groupData.SubDateEnd.After(time.Now()) {
			isSub <- true
		} else {
			isSub <- false
		}
	}()
	if err := handler.conn.GetUsersFromGroup(groupId, &userList); err != nil {
		return err
	}

	if userList.Len() <= 0 {
		msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, "Список пользователей: 0")
		if _, err := handler.api.Send(msg); err != nil {
			return err
		}
		return nil
	}
	var tagText = fmt.Sprintf("*Оповещено людей*: %d\n", userList.Len()-1)
	var messages = list.List{}

	for el := userList.Front(); el != nil; el = el.Next() {
		elValue := el.Value.(db.UserData)
		if elValue.TgId == strconv.FormatInt(handler.update.Message.From.ID, 10) {
			continue
		}
		addText := fmt.Sprintf("[@](tg://user?id=%s) ", elValue.TgId)
		if len(tagText)+len(tagText) > 4000 {
			messages.PushBack(tagText)
			tagText = ""
		}
		tagText = tagText + addText
	}

	if <-canTag || <-isSub {
		for el := messages.Front(); el != nil; el = el.Next() {
			msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, el.Value.(string))
			msg.ParseMode = tgbotapi.ModeMarkdownV2
			_, err := handler.api.Send(msg)
			if err == nil {
				go logs.SendLog(logs.LogEntry{
					Level:     "info",
					EventType: "user_action",
					Info:      "AllCommand",
					TgGroupID: groupId,
					TgUserID:  strconv.FormatInt(handler.update.Message.From.ID, 10),
				})
			} else {
				return err
			}
		}
		return nil
	}
	msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, handler.loc.Get("ru", "already_use"))
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err = handler.api.Send(msg)

	return err

}
