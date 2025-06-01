package payment

import (
	"CandallGo/internal/db"
	"CandallGo/internal/static"
	"container/list"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"log"
	"strconv"
	"time"
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
}

func PaymentCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update, state static.State, conn *db.DB) error {
	myData := &data{
		api:    bot,
		update: update,
		conn:   conn,
		state:  state,
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
		msg := tgbotapi.NewMessage(myData.update.Message.Chat.ID, "На данный момент нет доступных способов оплаты, попробуйте позже!")
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
			tgbotapi.InlineKeyboardButton{Text: "Назад", CallbackData: &callbackBack}))

	for el := currencies.Front(); el != nil; el = el.Next() {
		var cur = el.Value.(db.Currency)
		if cur.Name != "" {
			groupId := myData.state.Data

			curCallback, err := static.EncodePayment(
				static.PaymentState{Action: "sub_options", Data: db.PaymentData{GroupId: groupId, ProductId: -1, CurrencyId: cur.Id}})
			if err != nil {
				return err
			}
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.InlineKeyboardButton{
						Text: cur.Name, CallbackData: &curCallback,
					}))
		}
	}

	msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID,
		"Выберите валюту в оплате\n *Варианты подписки*")
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

	msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "Выберите подписку")
	var productList list.List
	if err := myData.conn.GetPrices(&productList); err != nil {
		return err
	}

	for el := productList.Front(); el != nil; el = el.Next() {

	}
	if productList.Len() <= 0 {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "На данный момент нет доступных способов оплаты, попробуйте позже")
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
				Text: "К выбору валюты", CallbackData: &currencyCallback,
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
		if product.CurrencyName != "XTR" {
			productText = fmt.Sprintf("%d : %s", product.Price/100, product.Name)
		} else {
			productText = fmt.Sprintf("%d : %s", product.Price, product.Name)
		}

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.InlineKeyboardButton{Text: productText, CallbackData: &buttonCallback}))
	}
	if count <= 0 {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "Нет доступных методов оплаты")
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
			log.Println("Error checkUserInGroup:", err)
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
			log.Println("Error checkSubscribe:", err)
			isSub <- false
			close(isSub)
			return
		}
		isSub <- isSubscribe
		close(isSub)
	}()

	if !<-isValidate {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "Похоже вы уже не состоите в группе")
		if _, err := myData.api.Send(msg); err != nil {
			return err
		}
		return errors.New("Havent Permission")
	}
	if <-isSub {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "Уже оформлена подписка")
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
		}
		res, err := myData.api.Send(invoice)
		if err != nil {
			log.Println("Error sending invoice:", err)
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
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "Ошибка отправки формы для оплаты, попробуйте позже")
		_, err := myData.api.Send(msg)
		return err
	}

	go func() {

		<-time.After(10 * time.Minute)
		is_paid, err := myData.conn.GetPaymentStatus(payload)
		if err != nil {
			log.Println("Error getting payment status:", err)
			return
		}
		if !is_paid {
			deleteMsg := tgbotapi.NewDeleteMessage(invoiceMessage.Chat.ID, invoiceMessage.MessageID)
			if _, err := myData.api.Request(deleteMsg); err != nil {
				log.Println("Error deleting payment status:", err)
			}
			msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "Один из запросов на платежей был удален, на платеж дается 10 минут. Повторите попытку, если не пришло письмо об успешной оплате")
			if _, err := myData.api.Send(msg); err != nil {
			}
		}

	}()

	userData := <-chanUserData
	groupData := <-chanGroupData
	productData := <-chanProductList

	if userData == nil || groupData == nil || productData == nil {
		msg := tgbotapi.NewMessage(myData.update.CallbackQuery.Message.Chat.ID, "Ошибка записи платежа, попробуйте позже")
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
