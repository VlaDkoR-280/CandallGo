package handlers

import (
	"CandallGo/internal/db"
	"container/list"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
	"time"
)

func GroupHandler(api *tgbotapi.BotAPI, conn *db.DB, update tgbotapi.Update) error {
	var handler = Handler{api: api, conn: conn, update: update}
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
				log.Println(err)
				return
			}
			if userId == strconv.FormatInt(bot.ID, 10) {
				err = handler.startCommand()
				if err != nil {
					log.Println(err)
				}
				return
			}

			if el.IsBot {
				return
			}

			_, err = handler.conn.GetUserData(userId)
			if err != nil {
				if !strings.Contains(err.Error(), "no rows in result") {
					log.Println("GetUserData", err)
				}

				err = handler.conn.AddUser(userId)
				if err != nil {
					log.Println("AddUser", err)
				}
			}
			err = handler.conn.AddUserToGroup(userId, groupId)
			if err != nil {
				log.Println("AddUserToGroup", err)
			}
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
	return err
}

func (handler *Handler) startCommand() error {
	text := "Я бот для тега всех участников в группе\n" +
		"Можешь воспользовать этими командами\n" +
		"/start - вывод основной информации о боте\n" +
		"/all - тег всех участников"
	msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, text)
	_, _ = handler.api.Send(msg)
	return nil
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
			msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, "Сегодня вы уже использовали бота, попробуйте завтра или купите подписку.")
			_, err := handler.api.Send(msg)
			if err != nil {
				log.Println(err)
			}
			canTag <- false
		}
		if err := handler.conn.UpdateGroupData(db.GroupData{DateLastUse: dateNow, TgId: groupId}); err != nil {
			log.Println(err)
		}
		canTag <- true
	}()
	go func() {
		if groupData.SubDateEnd.After(time.Now()) {
			canTag <- true
		} else {
			canTag <- false
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

	for el := userList.Front(); el != nil; el = el.Next() {
		elValue := el.Value.(db.UserData)
		if elValue.TgId == strconv.FormatInt(handler.update.Message.From.ID, 10) {
			continue
		}
		addText := fmt.Sprintf("[@](tg://user?id=%s) ", elValue.TgId)
		tagText = tagText + addText
	}

	msg := tgbotapi.NewMessage(handler.update.Message.Chat.ID, tagText)
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	if <-canTag || <-isSub {
		if _, err := handler.api.Send(msg); err != nil {
			return err
		}
	}
	return nil
}
