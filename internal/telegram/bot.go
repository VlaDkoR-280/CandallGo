package telegram

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	"CandallGo/internal/telegram/callbacks"
	"CandallGo/internal/telegram/handlers"
	"CandallGo/internal/telegram/payment"
	"CandallGo/logs"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api  *tgbotapi.BotAPI
	conn *db.DB
	loc  *localization.Local
}

func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	conn, err := db.Connect(config.LoadConfig().DbUrl)
	if err != nil {
		logs.SendLog(logs.LogEntry{
			Level:     "fatal",
			EventType: "data_base",
			Error:     err.Error(),
		})
		os.Exit(1)
	}
	var loc localization.Local
	err = loc.Update()
	if err != nil {
		logs.SendLog(logs.LogEntry{
			Level:     "fatal",
			EventType: "system",
			Error:     err.Error(),
		})
		os.Exit(1)
	}
	return &Bot{api: api, conn: conn, loc: &loc}, nil
}

func (bot *Bot) Close() {
	bot.conn.Close()
}

func (bot *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.api.GetUpdatesChan(u)
	u.AllowedUpdates = []string{"my_chat_member"}
	var maxUpdates = make(chan struct{}, 50)

	for update := range updates {
		maxUpdates <- struct{}{}

		go func() {
			defer func() { <-maxUpdates }()
			bot.myUpdate(update)
		}()
	}
}

func (bot *Bot) myUpdate(update tgbotapi.Update) {
	if update.PreCheckoutQuery != nil {
		payment.PreCheckoutQuery(update, bot.api)
		return
	}

	if update.Message != nil {

		if update.Message.From.IsBot {
			logs.SendLog(logs.LogEntry{
				Level:     "info",
				EventType: "bot_send_message",
				Msg:       update.Message.Text,
				TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
			})
			return
		}

		if update.Message.SuccessfulPayment != nil {
			payment.SuccessfulPayment(update, bot.conn)
			return
		}

		bot.fullCheck(update)

		switch update.Message.Chat.Type {
		case "private":
			err := handlers.PrivateHandler(bot.api, bot.conn, update, bot.loc)
			if err != nil {
				go logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "telegram_private_handler",
					TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
					TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
					Error:     err.Error(),
				})
				return
			}
		case "group", "supergroup":
			err := handlers.GroupHandler(bot.api, bot.conn, update, bot.loc)
			if err != nil {
				go logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "telegram_group_handler",
					TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
					TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
					Error:     err.Error(),
				})
				return
			}
		case "channel":
			err := handlers.ChannelHandler(bot.api, bot.conn, update, bot.loc)
			if err != nil {
				go logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "telegram_channel_handler",
					TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
					TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
					Error:     err.Error(),
				})
				return
			}
		}
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.From.IsBot {
			logs.SendLog(logs.LogEntry{
				Level:     "info",
				EventType: "bot_callback",
				Msg:       update.CallbackQuery.Data,
				TgUserID:  strconv.FormatInt(update.CallbackQuery.From.ID, 10),
			})
			return
		}
		// специальные исключения
		if err := callbacks.MainCallback(bot.api, update, bot.conn, handlers.PrivateHandler, bot.loc); err != nil {
			go logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "user_callback",
				Msg:       update.CallbackQuery.Data,
				TgUserID:  strconv.FormatInt(update.CallbackQuery.Message.From.ID, 10),
				TgGroupID: strconv.FormatInt(update.CallbackQuery.Message.Chat.ID, 10),
				Error:     err.Error(),
			})
			return
		}

	}

	if update.ChannelPost != nil && update.ChannelPost.Chat.Type == "channel" {
		err := handlers.ChannelHandler(bot.api, bot.conn, update, bot.loc)
		if err != nil {
			go logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "channel_post",
				TgPostId:  strconv.FormatInt(int64(update.ChannelPost.MessageID), 10),
				TgGroupID: strconv.FormatInt(update.ChannelPost.Chat.ID, 10),
				Error:     err.Error(),
			})
			return
		}
	}

}
