package telegram

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	"CandallGo/internal/telegram/callbacks"
	"CandallGo/internal/telegram/handlers"
	"CandallGo/logs"
	"container/list"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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
		answer := tgbotapi.PreCheckoutConfig{
			PreCheckoutQueryID: update.PreCheckoutQuery.ID,
			OK:                 true,
		}
		_, err := bot.api.Request(answer)
		if err != nil {
			logs.SendLog(logs.LogEntry{
				Level:          "error",
				EventType:      "payment_invoice",
				Error:          err.Error(),
				InvoicePayload: update.PreCheckoutQuery.InvoicePayload,
				TgUserID:       strconv.FormatInt(update.PreCheckoutQuery.From.ID, 10),
			})
			msg := tgbotapi.NewMessage(update.PreCheckoutQuery.From.ID, "*Счет устрарел*\nПовтори процесс выбора группы -> выбора подписки")
			if _, err := bot.api.Send(msg); err != nil {
				logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "system",
					Error:     err.Error(),
					TgUserID:  strconv.FormatInt(update.PreCheckoutQuery.From.ID, 10),
				})
			}
		}
		logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "payment_invoice",
			Msg:       "Successful send PreCheckoutQuery",
			TgUserID:  strconv.FormatInt(update.PreCheckoutQuery.From.ID, 10),
		})
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
			var payload = update.Message.SuccessfulPayment.InvoicePayload
			var p = update.Message.SuccessfulPayment
			go func() {
				err := bot.conn.UpdateSuccessfulPayment(p.InvoicePayload, p.ProviderPaymentChargeID, p.TelegramPaymentChargeID, true, false, time.Now())
				if err != nil {
					logs.SendLog(logs.LogEntry{
						Level:     "error",
						EventType: "data_base",
						TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
						Error:     fmt.Sprintf("%s\n%s", "Error Update SuccessfulPayment", err.Error()),
					})
					return
				}
				logs.SendLog(logs.LogEntry{
					Level:     "info",
					EventType: "data_base",
					TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
					Info:      "Update SuccessfulPayment",
				})
			}()
			go func() {
				var subDays = make(chan int)
				var cGroupId = make(chan string, 1)
				go func() {
					mProduct, err := bot.conn.GetProductData(payload)
					if err != nil {
						logs.SendLog(logs.LogEntry{
							Level:     "error",
							EventType: "data_base",
							TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
							Error:     fmt.Sprintf("%s\n%s", "Error GetProductData", err.Error()),
						})
						return
					}
					logs.SendLog(logs.LogEntry{
						Level:     "info",
						EventType: "data_base",
						TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
						Info:      "GetProductData",
					})
					subDays <- mProduct.DaysSubscribe
				}()

				go func() {
					mPayment, err := bot.conn.GetPaymentDataFromInvoice(payload)
					if err != nil {
						logs.SendLog(logs.LogEntry{
							Level:     "error",
							EventType: "data_base",
							TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
							Error:     fmt.Sprintf("%s\n%s", "Error GetPaymentDataFromInvoice", err.Error()),
						})
						return
					}
					go logs.SendLog(logs.LogEntry{
						Level:     "info",
						EventType: "data_base",
						TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
						Info:      "GetPaymentDataFromInvoice",
					})
					cGroupId <- mPayment.GroupId
				}()

				var groupId = <-cGroupId
				var timeSub = <-subDays
				newTimeSub := time.Now().AddDate(0, 0, timeSub)
				err := bot.conn.UpdateSubDate(groupId, newTimeSub)
				if err != nil {
					logs.SendLog(logs.LogEntry{
						Level:     "error",
						EventType: "data_base",
						TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
						Error:     fmt.Sprintf("%s\n%s", "Error UpdateSubDate", err.Error()),
					})
					return
				}
				logs.SendLog(logs.LogEntry{
					Level:     "info",
					EventType: "data_base",
					TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
					Info:      "UpdateSubDate",
				})
			}()

		}

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
			return
		}
		go logs.SendLog(logs.LogEntry{
			Level:     "info",
			EventType: "telegram",
			TgUserID:  strconv.FormatInt(update.Message.From.ID, 10),
			TgGroupID: strconv.FormatInt(update.Message.Chat.ID, 10),
			Info:      "checkExistOfUserInGroup",
		})

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
			go logs.SendLog(logs.LogEntry{
				Level:     "info",
				EventType: "data_base",
				TgGroupID: groupData.TgId,
				Info:      "AddGroup",
			})
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
