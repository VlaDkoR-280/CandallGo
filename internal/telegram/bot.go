package telegram

import (
	"CandallGo/internal/telegram/callbacks"
	"CandallGo/internal/telegram/handlers"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
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
	for update := range updates {
		if update.CallbackQuery != nil {
			err := callbacks.Callback(bot.api, update)
			if err != nil {
				log.Printf("Error: %s | Callback: %s ", err, update.CallbackQuery.Data)
			}
			continue
		}
		if update.Message == nil {
			continue
		}
		// CHAT TYPE: “private”, “group”, “supergroup” or “channel”
		switch update.Message.Chat.Type {
		case "supergroup", "group":
			groupCommands(bot, update)
		case "private":
			privateCommands(bot, update)
		case "channel":

		}

	}
}
func channelReply(bot *Bot, update tgbotapi.Update) {}
func privateCommands(bot *Bot, update tgbotapi.Update) {
	cmd := update.Message.Command()
	switch cmd {
	case "start":
		err := handlers.PrivateStart(bot.api, update)
		if err != nil {
			log.Println("ErrorStartPrivate: ", err)
		}
	case "group":
		// Get groups of user
		err := handlers.PrivateGetGroups(bot.api, update)
		if err != nil {
			log.Println("ErrorGetGroups: ", err)
		}
	}
}
func groupCommands(bot *Bot, update tgbotapi.Update) {
	cmd := update.Message.Command()
	switch cmd {
	case "all":
		err := handlers.AllTags(bot.api, update)
		if err != nil {
			log.Println("ErrorAllTag: ", err)
		}
	case "start":
		err := handlers.StartHandler(bot.api, update)
		if err != nil {
			log.Println("ErrorStartMsg: Start: ", err)
		}
	case "update":
		err := handlers.Update(bot.api, update)
		if err != nil {
			log.Println("ErrorUpdateUsers: ", err)
		}
	default:
		if len(update.Message.NewChatMembers) > 0 {
			err := handlers.NewMembers(bot.api, update)
			if err != nil {
				log.Println("Error new members: ", err)
			}
		} else if update.Message.LeftChatMember != nil {
			err := handlers.LeftMember(bot.api, update)
			if err != nil {
				log.Println("Error leftMember: ", err)
			}
		}
		err := handlers.MessageHandler(bot.api, update)
		if err != nil {
			log.Println("ErrorMsg: Default: ", err)
		}
	}
}
