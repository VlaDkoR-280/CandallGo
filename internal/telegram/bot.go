package telegram

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

type Bot struct {
	api *tgbotapi.BotAPI
}

func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &Bot{api: api}, nil
}

func (bot *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.api.GetUpdatesChan(u)
	conn, err := db.Connect(config.LoadConfig().DbUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	var maxUpdates = make(chan struct{}, 50)
	for update := range updates {
		if update.Message == nil || update.Message.From.IsBot {
			continue
		}
		maxUpdates <- struct{}{}

		go func() {
			defer func() { <-maxUpdates }()
			err := checkGroup(&update, conn)
			if err != nil {
				log.Println(err)
			}
			log.Printf("Обработано: %s", update.Message.Text)
		}()
		//// CHAT TYPE: “private”, “group”, “supergroup” or “channel”

	}
}

func checkGroup(update *tgbotapi.Update, conn *db.DB) error {
	var groupData db.GroupData
	groupData.TgId = strconv.FormatInt(update.Message.Chat.ID, 10)
	groupData.GroupName = update.Message.Chat.Title
	groupData.IsGroup = update.Message.Chat.IsGroup()

	dbData, err := conn.GetGroupData(groupData.TgId)
	if err != nil {
		if !strings.Contains(err.Error(), "no rows in result") {
			return err
		}

		err = conn.AddGroup(groupData.TgId, groupData.GroupName, groupData.IsGroup)
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
		err = conn.UpdateGroupData(groupData)
		return err
	}
	return nil
}
