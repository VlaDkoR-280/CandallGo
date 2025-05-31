package main

import (
	"CandallGo/config"
	"CandallGo/internal/telegram"
	"log"
)

func main() {
	cfg := config.LoadConfig()
	//log.Println("url = ", cfg.DbUrl)
	//mdb, err := pgxpool.New(context.Background(), cfg.DbUrl)
	//if err != nil {
	//	log.Fatal("1: ", err)
	//}
	//defer mdb.Close()
	//var count int
	//err1 := mdb.QueryRow(context.Background(), "SELECT COUNT(*) FROM user_data").Scan(&count)
	//
	//if err1 != nil {
	//	log.Fatal("2: ", err1)
	//}
	//
	//log.Println("Successfully added user:", count)

	//mdb, err := db.Connect(cfg.DbUrl)
	//if err != nil {
	//	log.Fatal("Error connecting to database:", err)
	//}
	//
	//log.Println("Connected to database")
	//
	//
	//log.Fatal(err)
	//defer mdb.Close()
	bot, err := telegram.NewBot(cfg.BotToken)
	if err != nil {
		log.Fatal("Bot connect Error", err)
	}
	defer bot.Close()
	bot.Start()

}
