package handlers

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"container/list"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
	"time"
)

func StartHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "START")
	_, err := bot.Send(msg)
	return err
}

func MessageHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	db_url := config.LoadConfig().DbUrl
	database, err := db.Connect(db_url)
	if err != nil {
		return err
	}
	defer database.Close()

	var userId string
	var groupId string
	var groupName string

	userId = strconv.FormatInt(update.Message.From.ID, 10)
	groupId = strconv.FormatInt(update.Message.Chat.ID, 10)
	groupName = update.Message.Chat.Title

	err = database.CheckExist(userId, groupId, groupName)
	return err

}

func Update(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	err := MessageHandler(bot, update)
	if err != nil {
		return err
	}
	db_url := config.LoadConfig().DbUrl
	database, err := db.Connect(db_url)
	if err != nil {
		return err
	}
	defer database.Close()

	// Проверка на возможность обновления
	canUpdate, err := database.CheckTimeUpdate(strconv.FormatInt(update.Message.Chat.ID, 10))
	if err != nil {
		return err
	}
	var msg tgbotapi.MessageConfig
	if !canUpdate {
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Для повторного обновления подождите 20 минут, от последнего запроса /update")
	} else {
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Проверяю группу на наличие новых людей")
		err = database.SetTimeUpdate(strconv.FormatInt(update.Message.Chat.ID, 10))
		if err != nil {
			return err
		}

	}
	_, err = bot.Send(msg)
	return err
}

//func Add(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
//	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Check")
//	_, err := bot.Send(msg)
//	cfg := config.LoadConfig()
//	database_url := cfg.DbUrl
//	database, err_db := db.Connect(database_url)
//	if err_db != nil {
//		return err_db
//	}
//
//	defer database.Close()
//
//	userId := update.Message.From.ID
//	groupId := update.Message.Chat.ID
//	groupName := update.Message.Chat.Title
//	err_check := database.CheckExist(strconv.FormatInt(userId, 10), strconv.FormatInt(groupId, 10), groupName)
//	if err_check != nil {
//		return err_check
//	}
//	return err
//}

func NewMembers(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	users := update.Message.NewChatMembers
	groupId := strconv.FormatInt(update.Message.Chat.ID, 10)
	database_url := config.LoadConfig().DbUrl
	database, err_db := db.Connect(database_url)
	if err_db != nil {
		return err_db
	}
	for _, user := range users {
		userId := strconv.FormatInt(user.ID, 10)
		err := database.CheckExistUser(userId, groupId)
		if err != nil {
			return err
		}
	}
	return err_db
}
func LeftMember(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	var userId string
	var groupId string
	userId = strconv.FormatInt(update.Message.LeftChatMember.ID, 10)
	groupId = strconv.FormatInt(update.Message.Chat.ID, 10)
	database_url := config.LoadConfig().DbUrl
	database, err_db := db.Connect(database_url)
	if err_db != nil {
		return err_db
	}
	defer database.Close()
	err := database.LeftUserFromGroup(userId, groupId)
	return err

}
func AllTags(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	var listMembers list.List
	databaseUrl := config.LoadConfig().DbUrl
	database, err := db.Connect(databaseUrl)
	if err != nil {
		return err
	}
	defer database.Close()

	var userId string
	var groupId string
	userId = strconv.FormatInt(update.Message.From.ID, 10)
	groupId = strconv.FormatInt(update.Message.Chat.ID, 10)

	var dateEndSub time.Time
	err = database.GetSubEndDate(groupId, &dateEndSub)
	if err != nil {
		return err
	}

	if !dateEndSub.After(time.Now()) {
		var lastUseDay time.Time

		err = database.GetLastTimeUse(groupId, &lastUseDay)
		if err != nil {
			return err
		}

		if lastUseDay.Truncate(24 * time.Hour).Equal(time.Now().Truncate(24 * time.Hour)) {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Использовать эту команду бесплатно можно раз в день, рассмотрите варианты нашей подписки...")
			_, err = bot.Send(msg)
			return err
		}
	}

	err = database.GetUsersOfGroup(userId, groupId, &listMembers)
	if err != nil {
		return err
	}
	s := "Tag this: "
	for e := listMembers.Front(); e != nil; e = e.Next() {
		s += e.Value.(string) + " "
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, s)
	_, err = bot.Send(msg)
	return err

}
