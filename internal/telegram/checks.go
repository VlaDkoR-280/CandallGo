package telegram

import (
	"CandallGo/internal/db"
	"CandallGo/logs"
	"container/list"
	"fmt"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (bot *Bot) fullCheck(update tgbotapi.Update) error {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := bot.checkGroup(update)
		if err != nil {
			logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "telegram",
				TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
				TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
				Error:     fmt.Sprintf("%s\n%s", "Error checkGroup", err.Error()),
			})
			return
		}
		logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "telegram",
			TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
			TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
			Info:      "checkGroup",
		})
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := bot.checkUser(update)
		if err != nil {
			logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "telegram",
				TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
				TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
				Error:     fmt.Sprintf("%s\n%s", "Error checkUser", err.Error()),
			})
			return
		}
		logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "telegram",
			TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
			TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
			Info:      "checkUser",
		})
	}()

	wg.Wait()

	if err := bot.checkExistOfUserInGroup(update); err != nil {
		go logs.SendLog(logs.LogEntry{
			Level:     "error",
			EventType: "telegram",
			TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
			TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
			Error:     fmt.Sprintf("%s\n%s", "Error checkExistOfUserInGroup", err.Error()),
		})
		return err
	}
	go logs.SendLog(logs.LogEntry{
		Level:     "info",
		EventType: "telegram",
		TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
		TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
		Info:      "checkExistOfUserInGroup",
	})
	return nil
}

func (bot *Bot) checkGroup(update tgbotapi.Update) error {
	var groupData db.GroupData
	groupData.TgId = strconv.FormatInt(update.Message.Chat.ID, 10)
	groupData.GroupName = update.Message.Chat.Title
	groupData.IsGroup = update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup()

	dbData, err := bot.conn.GetGroupData(groupData.TgId)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows in result") {
			return err
		}

		err = bot.conn.AddGroup(groupData.TgId, groupData.GroupName, groupData.IsGroup)
		if err != nil {
			return err
		}
		go logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "data_base",
			TgGroupID: groupData.TgId,
			Info:      "AddGroup",
		})
		return nil
	}

	if dbData.GroupName != groupData.GroupName {
		err = bot.conn.UpdateGroupData(groupData)
		return err
	}
	return nil
}

func (bot *Bot) checkUser(update tgbotapi.Update) error {
	var userId = strconv.FormatInt(update.Message.From.ID, 10)
	_, err := bot.conn.GetUserData(userId)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows in result") {
			return err
		}
		err = bot.conn.AddUser(userId)
		if err != nil {
			return err
		}
		go logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "data_base",
			TgUserID:  userId,
			Info:      "AddUser",
		})
		return nil
	}
	return nil
}

func (bot *Bot) checkExistOfUserInGroup(update tgbotapi.Update) error {
	var userId = strconv.FormatInt(update.Message.From.ID, 10)
	var chatId = strconv.FormatInt(update.Message.Chat.ID, 10)
	var users list.List
	err := bot.conn.GetUsersFromGroup(chatId, &users)
	if err != nil {
		return err
	}
	var isExist = false

	for user := users.Front(); user != nil; user = user.Next() {
		if user.Value.(db.UserData).TgId == userId {
			isExist = true
			break
		}
	}

	if !isExist {
		err = bot.conn.AddUserToGroup(userId, chatId)
		if err != nil {
			return err
		}
		go logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "data_base",
			TgUserID:  userId,
			TgGroupID: chatId,
			Info:      "AddUserToGroup",
		})
	}
	return nil
}
