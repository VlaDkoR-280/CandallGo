package logs

import (
	"CandallGo/config"
	"encoding/json"
	"log"
	"net"
	"time"
)

type LogEntry struct {
	Level     string `json:"level"`
	Msg       string `json:"msg"`
	TimeStamp string `json:"@timestamp"`
}

func SendLog(level, msg string) {
	entry := LogEntry{
		Level:     level,
		Msg:       msg,
		TimeStamp: time.Now().UTC().Format(time.RFC3339),
	}

	conn, err := net.Dial("tcp", config.LoadConfig().LogUrl)
	if err != nil {
		log.Println("Error connecting to Log Server:", err)
		return
	}
	defer conn.Close()
	data, _ := json.Marshal(entry)
	_, _ = conn.Write(append(data, '\n'))
}
