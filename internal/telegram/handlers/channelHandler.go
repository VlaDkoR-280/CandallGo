package handlers

import (
	"CandallGo/config"
	"CandallGo/internal/db"
	"CandallGo/internal/localization"
	"container/list"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

func ChannelHandler(api *tgbotapi.BotAPI, conn *db.DB, update tgbotapi.Update, loc *localization.Local) error {
	if update.ChannelPost != nil && update.ChannelPost.Chat.ID == config.LoadConfig().ChannelId {
		handler := Handler{
			api: api, conn: conn, update: update, loc: loc,
		}

		return handler.channelForward()
	}
	return nil
}

func (h *Handler) channelForward() error {
	channelPost := h.update.ChannelPost
	var text string
	if channelPost.Poll != nil {
		text = channelPost.Poll.Question
	} else {
		text = channelPost.Text
	}

	if strings.Contains(text, "#forwards") {
		var groups list.List
		err := h.conn.GetAllGroups(&groups)
		if err != nil {
			return err
		}

		for el := groups.Front(); el != nil; el = el.Next() {
			var group = el.Value.(db.GroupData)
			go func() {
				if !group.IsGroup {
					return
				}
				chatId, err := strconv.Atoi(group.TgId)
				if err != nil {
					return
				}
				forward := tgbotapi.NewForward(int64(chatId), channelPost.Chat.ID, channelPost.MessageID)
				if _, err := h.api.Send(forward); err != nil {
					log.Println(err)
				}
			}()
		}

	}
	return nil
}
