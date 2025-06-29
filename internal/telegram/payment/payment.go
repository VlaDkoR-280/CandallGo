package payment

import (
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	"CandallGo/internal/static"
	"CandallGo/logs"
	"container/list"
	"errors"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type Provider struct {
	Id       int64
	Name     string
	Token    string
	Currency string
}

type data struct {
	api          *tgbotapi.BotAPI
	update       tgbotapi.Update
	conn         *db.DB
	state        static.State
	paymentState *static.PaymentState
	loc          *localization.Local
}

func PaymentCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update, state static.State, conn *db.DB, loc *localization.Local) error {
	myData := &data{
		api:    bot,
		update: update,
		conn:   conn,
		state:  state,
		loc:    loc,
	}
	if err := myData.deleteMsg(); err != nil {
		return err
	}
	switch state.Action {
	case "subscribe_methods":
		if err := myData.paymentMethods(); err != nil {
			return err
		}
	case "sub_options":

		var bufState, err = static.DecodePayment(update.CallbackData())
		if err != nil {
			return err
		}
		myData.paymentState = &bufState
		myData.state.Data = myData.paymentState.Data.GroupId
		return myData.purchaseCallback()

	case "invoice":
		var bufState, err = static.DecodePayment(update.CallbackData())
		if err != nil {
			return err
		}
		myData.paymentState = &bufState
		myData.state.Data = myData.paymentState.Data.GroupId
		return myData.invoiceCallback()

	}

	return nil
}

func (myData *data) deleteMsg() error {
	msg := tgbotapi.NewDeleteMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.update.CallbackQuery.Message.MessageID)
	_, err := myData.api.Request(msg)
	return err
}

func (myData *data) paymentMethods() error {

	if err := myData.fullCheck(); err != nil {
		return err
	}

	var currencies list.List
	if err := myData.conn.GetCurrency(&currencies); err != nil {
		return err
	}
	if currencies.Len() <= 0 {
		msg := tgbotapi.NewMessage(myData.update.Message.Chat.ID, myData.loc.Get("ru", "currency_empty"))
		_, err := myData.api.Send(msg)
		return err
	}

	callbackBack, err := static.EncodeState(
		static.State{Action: "group", Data: myData.state.Data})
	if err != nil {
		return err
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.InlineKeyboardButton{Text: myData.loc.Get("ru", "button_back"), CallbackData: &callbackBack}))

	for el := currencies.Front(); el != nil; el = el.Next() {
		var cur = el.Value.(db.Currency)
		if cur.Name != "" {
			groupId := myData.state.Data

			var curName string
			switch cur.Name {
			case "RUB":
				curName = "🇷🇺₽"
			case "XTR":
				curName = "⭐"
			}

			curCallback, err := static.EncodePayment(
				static.PaymentState{Action: "sub_options", Data: db.PaymentData{GroupId: groupId, ProductId: -1, CurrencyId: cur.Id}})
			if err != nil {
				return err
			}
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.InlineKeyboardButton{
						Text: curName, CallbackData: &curCallback,
					}))
		}
	}

	msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID,
		myData.loc.Get("ru", "currency_choose"))
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err = myData.api.Send(msg)

	return err

}

func (myData *data) checkSubscribe() (bool, error) {
	group, err := myData.conn.GetGroupData(myData.state.Data)
	if err != nil {
		return false, err
	}

	return group.SubDateEnd.Truncate(24 * time.Hour).After(time.Now().Truncate(24 * time.Hour)), nil
}

func (myData *data) checkUserInGroup() (bool, error) {
	var users list.List
	if err := myData.conn.GetUsersFromGroup(myData.state.Data, &users); err != nil {
		return false, err
	}
	for el := users.Front(); el != nil; el = el.Next() {
		var user = el.Value.(db.UserData)
		if user.TgId == strconv.FormatInt(myData.update.CallbackQuery.From.ID, 10) {
			return true, nil
		}
	}
	return false, nil

}

func (myData *data) purchaseCallback() error {
	if myData.paymentState == nil {
		return errors.New("paymentState is nil")
	}

	if err := myData.fullCheck(); err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "subscribe_buy"))
	var productList list.List
	if err := myData.conn.GetPrices(&productList); err != nil {
		return err
	}

	if productList.Len() <= 0 {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "subscribe_empty"))
		if _, err := myData.api.Send(msg); err != nil {
			return err
		}
		return errors.New("Not allowed to purchase")
	}

	currencyCallback, err := static.EncodeState(static.State{Action: "subscribe_methods", Data: myData.paymentState.Data.GroupId})
	if err != nil {
		return err
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.InlineKeyboardButton{
				Text: myData.loc.Get("ru", "button_back"), CallbackData: &currencyCallback,
			}))

	curId := myData.paymentState.Data.CurrencyId
	count := 0
	for el := productList.Front(); el != nil; el = el.Next() {
		var product = el.Value.(db.Product)
		if product.CurrencyId != curId {
			continue
		}
		count++
		myData.paymentState.Data.ProductId = product.ProductId
		var buttonCallback, err = static.EncodePayment(
			static.PaymentState{Action: "invoice", Data: myData.paymentState.Data})
		if err != nil {
			return err
		}

		var productText string
		var curName string
		var curPrice int64
		switch product.CurrencyName {
		case "XTR":
			curName = "⭐"
			curPrice = product.Price
		case "RUB":
			curName = "₽"
			curPrice = product.Price / 100
		default:
			curName = product.CurrencyName
			curPrice = product.Price / 100
		}

		productText = fmt.Sprintf(myData.loc.Get("ru", "subscribe_button_text"), product.Name, curPrice, curName)

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.InlineKeyboardButton{Text: productText, CallbackData: &buttonCallback}))
	}
	if count <= 0 {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "subscribe_empty"))
		_, err := myData.api.Send(msg)
		return err
	}
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeMarkdownV2
	_, err = myData.api.Send(msg)
	return err
}

func (myData *data) fullCheck() error {
	var isSub = make(chan bool, 1)
	var isValidate = make(chan bool, 1)
	go func() {
		isExist, err := myData.checkUserInGroup()
		if err != nil {
			logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "user_callback",
				TgUserID:  strconv.FormatInt(myData.update.CallbackQuery.Message.From.ID, 10),
				TgGroupID: strconv.FormatInt(myData.update.CallbackQuery.Message.Chat.ID, 10),
				Error:     fmt.Sprintf("%s\n%s", "Error Checking User", err.Error()),
			})
			isValidate <- false
			close(isValidate)
			return
		}
		isValidate <- isExist
		close(isValidate)
	}()

	go func() {
		isSubscribe, err := myData.checkSubscribe()
		if err != nil {
			logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "user_callback",
				TgUserID:  strconv.FormatInt(myData.update.CallbackQuery.Message.From.ID, 10),
				TgGroupID: strconv.FormatInt(myData.update.CallbackQuery.Message.Chat.ID, 10),
				Error:     fmt.Sprintf("%s\n%s", "Error Checking Subscribe", err.Error()),
			})
			isSub <- false
			close(isSub)
			return
		}
		isSub <- isSubscribe
		close(isSub)
	}()

	if !<-isValidate {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "post_group_empty"))
		if _, err := myData.api.Send(msg); err != nil {
			return err
		}
		return errors.New("Havent Permission")
	}
	if <-isSub {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "subscribe_already"))
		if _, err := myData.api.Send(msg); err != nil {
			return err
		}
		return errors.New("Already Subscribe")
	}
	return nil
}

func (myData *data) invoiceCallback() error {
	if err := myData.fullCheck(); err != nil {
		return err
	}
	var products list.List
	if err := myData.conn.GetPrices(&products); err != nil {
		return err
	}
	var product *db.Product = nil
	for el := products.Front(); el != nil; el = el.Next() {
		buf := el.Value.(db.Product)
		if buf.ProductId == myData.paymentState.Data.ProductId &&
			buf.CurrencyId == myData.paymentState.Data.CurrencyId {
			newProduct := el.Value.(db.Product)
			product = &newProduct
			break
		}
	}
	if product == nil {
		return errors.New("Product not found")
	}
	var sendInvoice = make(chan bool, 1)
	var payload = uuid.New().String()
	var invoiceMessage tgbotapi.Message
	go func() {
		invoice := tgbotapi.InvoiceConfig{
			BaseChat:       tgbotapi.BaseChat{ChatID: myData.update.CallbackQuery.Message.Chat.ID},
			Title:          product.Name,
			Description:    product.Description,
			Payload:        payload,
			ProviderToken:  product.ProviderToken,
			StartParameter: "test",
			Currency:       product.CurrencyName,
			Prices: []tgbotapi.LabeledPrice{
				{Label: product.Name, Amount: int(product.Price)},
			},
			IsFlexible:          false,
			SuggestedTipAmounts: []int{},
			PhotoURL:            "https://cdn4.cdn-telegram.org/file/X49EI542lmGGeZG-5HKbyozULKtQiO6UOM1_i3fmZ9hPBbdKe1nzUGZiTjma6aS6V977Pip15jWsJjRP9p60eEuQEYQX3xAuGbkM3DGe-6vHXvLV1UfdLCNPkT81gLIi756CsXY5XYmDPHiRxk2lLPbF3-0CLp5bddZgTbHzX000QnTNOIJCqB2cwpXybuUhvuo0O82QzOFYBiRLBPzSvPXIgLmqAaS2GOYGLOB8wULllIcsLS3uqo7vRTHQww0isyQRZZ1OXq_j2qdTDaUL_8_T_WNAlRh0uUlrTpsA7dpIjQY_GrqSVuhECHFRU__fha8HPBB7JB4Cudg-PzeT3Q.jpg",
			PhotoWidth:          400,
			PhotoHeight:         400,
		}
		res, err := myData.api.Send(invoice)
		if err != nil {
			logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "payment",
				TgUserID:  strconv.FormatInt(myData.update.CallbackQuery.Message.From.ID, 10),
				TgGroupID: strconv.FormatInt(myData.update.CallbackQuery.Message.Chat.ID, 10),
				Error:     fmt.Sprintf("%s\n%s", "Error sending invoice", err.Error()),
			})
			sendInvoice <- false
			return
		}
		invoiceMessage = res
		sendInvoice <- true
	}()
	var chanGroupData = make(chan *db.GroupData, 1)
	var chanUserData = make(chan *db.UserData, 1)
	var chanProductList = make(chan *db.Product, 1)
	go func() {
		groupData, err := myData.conn.GetGroupData(myData.paymentState.Data.GroupId)
		if err != nil {
			chanGroupData <- nil
			return
		}
		chanGroupData <- &groupData
	}()

	go func() {
		userData, err := myData.conn.GetUserData(strconv.FormatInt(myData.update.CallbackQuery.From.ID, 10))
		if err != nil {
			chanUserData <- nil
			return
		}
		chanUserData <- &userData
	}()

	go func() {
		var products list.List
		err := myData.conn.GetPrices(&products)
		if err != nil || products.Len() <= 0 {
			chanProductList <- nil
			return
		}

		for el := products.Front(); el != nil; el = el.Next() {
			buf := el.Value.(db.Product)
			if buf.ProductId == myData.paymentState.Data.ProductId &&
				buf.CurrencyId == myData.paymentState.Data.CurrencyId {
				newProduct := el.Value.(db.Product)
				chanProductList <- &newProduct
				return
			}
		}
		chanProductList <- nil
	}()

	if !<-sendInvoice {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "error_send_invoice"))
		_, err := myData.api.Send(msg)
		return err
	}

	go func() {

		<-time.After(10 * time.Minute)
		is_paid, err := myData.conn.GetPaymentStatus(payload)
		if err != nil {
			logs.SendLog(logs.LogEntry{
				Level:     "error",
				EventType: "data_base",
				TgUserID:  strconv.FormatInt(myData.update.CallbackQuery.Message.From.ID, 10),
				TgGroupID: strconv.FormatInt(myData.update.CallbackQuery.Message.Chat.ID, 10),
				Error:     fmt.Sprintf("%s\n%s", "Error getting payment status", err.Error()),
			})
			return
		}
		if !is_paid {
			deleteMsg := tgbotapi.NewDeleteMessage(invoiceMessage.Chat.ID, invoiceMessage.MessageID)
			if _, err := myData.api.Request(deleteMsg); err != nil {
				go logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "telegram",
					TgUserID:  strconv.FormatInt(myData.update.CallbackQuery.Message.From.ID, 10),
					TgGroupID: strconv.FormatInt(myData.update.CallbackQuery.Message.Chat.ID, 10),
					Error:     fmt.Sprintf("%s\n%s", "Error deleting payment status", err.Error()),
				})
			}
			msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "invoice_timeout"))
			if _, err := myData.api.Send(msg); err != nil {
				logs.SendLog(logs.LogEntry{
					Level:     "error",
					EventType: "telegram",
					TgUserID:  strconv.FormatInt(myData.update.CallbackQuery.Message.From.ID, 10),
					TgGroupID: strconv.FormatInt(myData.update.CallbackQuery.Message.Chat.ID, 10),
					Error:     fmt.Sprintf("%s\n%s", "Error sending payment status", err.Error()),
				})
			}
		}

	}()

	userData := <-chanUserData
	groupData := <-chanGroupData
	productData := <-chanProductList

	if userData == nil || groupData == nil || productData == nil {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, myData.loc.Get("ru", "error_send_invoice"))
		_, err := myData.api.Send(msg)
		return err
	}

	paymentData := myData.paymentState.Data
	err := myData.conn.AddPayment(groupData.Id, userData.Id, paymentData.CurrencyId,
		int(productData.Price), groupData.TgId, userData.TgId, productData.CurrencyName, payload,
		productData.Description, productData.CurrencyName, productData.ProductId)
	if err != nil {
		return err
	}
	return nil
}

func SuccessfulPayment(update tgbotapi.Update, conn *db.DB) {
	var payload = update.Message.SuccessfulPayment.InvoicePayload
	var p = update.Message.SuccessfulPayment
	go func() {
		err := conn.UpdateSuccessfulPayment(p.InvoicePayload, p.ProviderPaymentChargeID, p.TelegramPaymentChargeID, true, false, time.Now())
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
			mProduct, err := conn.GetProductData(payload)
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
			mPayment, err := conn.GetPaymentDataFromInvoice(payload)
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
		err := conn.UpdateSubDate(groupId, newTimeSub)
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

func PreCheckoutQuery(update tgbotapi.Update, api *tgbotapi.BotAPI) {
	answer := tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: update.PreCheckoutQuery.ID,
		OK:                 true,
	}
	_, err := api.Request(answer)
	if err != nil {
		logs.SendLog(logs.LogEntry{
			Level:          "error",
			EventType:      "payment_invoice",
			Error:          err.Error(),
			InvoicePayload: update.PreCheckoutQuery.InvoicePayload,
			TgUserID:       strconv.FormatInt(update.PreCheckoutQuery.From.ID, 10),
		})
		msg := tgbotapi.NewMessage(update.PreCheckoutQuery.From.ID, "*Счет устрарел*\nПовтори процесс выбора группы -> выбора подписки")
		if _, err := api.Send(msg); err != nil {
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
}
