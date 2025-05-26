package main

import (
	"CandallGo/config"
	"fmt"
)

func main() {
	cfg := config.LoadConfig()
	fmt.Println(cfg.BotToken)
}
