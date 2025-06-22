package logs

import (
	"CandallGo/config"
	"encoding/json"
	"log"
	"net"
)

type LogEntry struct {
	// info error fatal
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	TimeStamp string `json:"@timestamp"`

	// полезный контекст
	TgPostId  string `json:"tg_post_id,omitempty"`
	TgUserID  string `json:"tg_user_id,omitempty"`
	TgGroupID string `json:"tg_group_id,omitempty"`

	// system payment data_base user_callback bot_callback new_member
	EventType      string `json:"event_type,omitempty"` // subscription, payment, command
	Command        string `json:"command,omitempty"`    // если был вызов команды
	Error          string `json:"error,omitempty"`      // если ошибка
	Info           string `json:"info,omitempty"`
	InvoicePayload string `json:"payment_id,omitempty"` // если есть
	ProductID      int    `json:"product_id,omitempty"` // если есть
}

func SendLog(entry LogEntry) {
	logUrl := config.LoadConfig().LogUrl
	if logUrl == "" {
		log.Println(entry)
	}
	conn, err := net.Dial("tcp", logUrl)
	data, _ := json.Marshal(entry)
	if err != nil {
		log.Println(string(data))
		return
	}
	defer func() {
		_ = conn.Close()
	}()
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		log.Println("Error write to vector: ", err)
	}
}
