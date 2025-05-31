package telegram

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"CandallGo/internal/telegram/callbacks"
	"CandallGo/internal/telegram/handlers"
	"container/list"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
	"sync"
)

type Bot struct {
	api  *tgbotapi.BotAPI
	conn *db.DB
}

func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	conn, err := db.Connect(config.LoadConfig().DbUrl)
	if err != nil {
		log.Fatal(err)
	}
	return &Bot{api: api, conn: conn}, nil
}

func (bot *Bot) Close() {
	bot.conn.Close()
}

func (bot *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.api.GetUpdatesChan(u)

	var maxUpdates = make(chan struct{}, 50)
	for update := range updates {
		maxUpdates <- struct{}{}

		go func() {
			defer func() { <-maxUpdates }()
			bot.myUpdate(update)
		}()
		//// CHAT TYPE: “private”, “group”, “supergroup” or “channel”

	}
}

func (bot *Bot) myUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		if update.Message.From.IsBot {
			log.Printf("MSG_FROM_BOT <%s|%s>: %s", update.Message.From.LastName, update.Message.From.ID, update.Message.Text)
			return
		}
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bot.checkGroup(update)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bot.checkUser(update)
		}()

		wg.Wait()

		_ = bot.checkExistOfUserInGroup(update)

		switch update.Message.Chat.Type {
		case "private":
			_ = handlers.PrivateHandler(bot.api, bot.conn, update)
		case "group", "supergroup":
			_ = handlers.GroupHandler(bot.api, bot.conn, update)
		case "channel":
			_ = handlers.ChannelHandler(bot.api, bot.conn, update)
		}
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.From.IsBot {
			log.Printf("MSG_FROM_BOT <%s|%s>: %s", update.CallbackQuery.From.LastName, update.CallbackQuery.From.ID, update.CallbackQuery.Data)
			return
		}
		_ = callbacks.MainCallback(bot.api, update, bot.conn)
	}
}

func (bot *Bot) checkGroup(update tgbotapi.Update) error {
	var groupData db.GroupData
	groupData.TgId = strconv.FormatInt(update.Message.Chat.ID, 10)
	groupData.GroupName = update.Message.Chat.Title
	groupData.IsGroup = update.Message.Chat.IsGroup()

	dbData, err := bot.conn.GetGroupData(groupData.TgId)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows in result") {
			return err
		}

		err = bot.conn.AddGroup(groupData.TgId, groupData.GroupName, groupData.IsGroup)
		if err != nil {
			log.Println("AddGroup", err)
		}
		return nil
	}
	var needUpdate = false
	if dbData.GroupName != groupData.GroupName {
		needUpdate = true
		groupData.GroupName = ""
	}

	if needUpdate {
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
			log.Println("GetUserData", err)
		}

		err = bot.conn.AddUser(userId)
		if err != nil {
			log.Println("AddUser", err)
		}

		return nil
	}
	return nil
}

func (bot *Bot) checkExistOfUserInGroup(update tgbotapi.Update) error {
	var userId = strconv.FormatInt(update.Message.From.ID, 10)
	var chatId = strconv.FormatInt(update.Message.Chat.ID, 10)
	var users list.List
	_ = bot.conn.GetUsersFromGroup(chatId, &users)

	var isExist = false

	for user := users.Front(); user != nil; user = user.Next() {
		if user.Value.(db.UserData).TgId == userId {
			isExist = true
			break
		}
	}

	if !isExist {
		_ = bot.conn.AddUserToGroup(userId, chatId)
	}
	return nil
}
