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
	"time"
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
	u.AllowedUpdates = []string{"my_chat_member"}
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
	if update.PreCheckoutQuery != nil {
		answer := tgbotapi.PreCheckoutConfig{
			PreCheckoutQueryID: update.PreCheckoutQuery.ID,
			OK:                 true,
		}
		_, err := bot.api.Request(answer)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(update.PreCheckoutQuery.From.ID, "*Счет устрарел*\nПовтори процесс выбора группы -> выбора подписки")
			if _, err := bot.api.Send(msg); err != nil {
				log.Println(err)
			}
		}
		return
	}

	// Удаление бота
	//if update.MyChatMember != nil {
	//	newStatus := update.MyChatMember.NewChatMember.Status
	//
	//	switch newStatus {
	//	case "member", "administration":
	//		msg := tgbotapi.NewMessage(update.MyChatMember.Chat.ID,
	//			"Привет, я бот для тега всех участников в группе. Пожалуйста, выдай мне права администратора, иначе я не смогу видеть участников группы")
	//		if _, err := bot.api.Send(msg); err != nil {
	//			log.Println(err)
	//		}
	//	case "left", "kicked":
	//		err := bot.conn.RemoveLinkUsersWithGroup(strconv.FormatInt(update.MyChatMember.Chat.ID, 10))
	//		if err != nil {
	//			log.Println(err)
	//		}
	//	}
	//	return
	//}

	if update.Message != nil {
		//if update.Message.NewChatMembers != nil {
		//	return
		//}
		if update.Message.From.IsBot {
			log.Printf("MSG_FROM_BOT <%s|%s>: %s", update.Message.From.LastName, update.Message.From.ID, update.Message.Text)
			return
		}

		if update.Message.SuccessfulPayment != nil {
			log.Println("SUCCSESSFULPAYMENT")
			var payload = update.Message.SuccessfulPayment.InvoicePayload
			var p = update.Message.SuccessfulPayment
			go func() {
				err := bot.conn.UpdateSuccessfulPayment(p.InvoicePayload, p.ProviderPaymentChargeID, p.TelegramPaymentChargeID, true, false, time.Now())
				if err != nil {
					log.Println(err)
				}
			}()
			go func() {

				var subDays = make(chan int)
				var cGroupId = make(chan string, 1)
				go func() {
					mProduct, err := bot.conn.GetProductData(payload)
					if err != nil {
						log.Println(err)
						return
					}

					subDays <- mProduct.DaysSubscribe
				}()

				go func() {
					mPayment, err := bot.conn.GetPaymentDataFromInvoice(payload)
					if err != nil {
						log.Println(err)
						return
					}
					cGroupId <- mPayment.GroupId
				}()

				var groupId = <-cGroupId
				var timeSub = <-subDays
				newTimeSub := time.Now().AddDate(0, 0, timeSub)
				err := bot.conn.UpdateSubDate(groupId, newTimeSub)
				if err != nil {
					log.Println(err)
					return
				}
			}()

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
			err := handlers.GroupHandler(bot.api, bot.conn, update)
			if err != nil {
				log.Println(err)
			}
		case "channel":
			_ = handlers.ChannelHandler(bot.api, bot.conn, update)
		}
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.From.IsBot {
			log.Printf("MSG_FROM_BOT <%s|%s>: %s", update.CallbackQuery.From.LastName, update.CallbackQuery.From.ID, update.CallbackQuery.Data)
			return
		}
		// специальные исключения
		if err := callbacks.MainCallback(bot.api, update, bot.conn, handlers.PrivateHandler); err != nil {
			log.Println(err)
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
