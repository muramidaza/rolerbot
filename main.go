package main

import (
	"fmt"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"os"
	"strconv"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"	
)

var db *sql.DB
var param string
var err error
var reply string
var query string
var count int
var answer string
var id int64

type Config struct {
	TelegramBotToken string
	Host string
	DBName string
	User string
	Password string
}

func main() {
	//Читаем файл с настройками
	file, _ := os.Open("config.json")
	log.Printf("load file")
	decoder := json.NewDecoder(file)
	conf := Config{}
	err := decoder.Decode(&conf)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(conf.TelegramBotToken)
	
	//Подключаемся к БД
	param := conf.User + ":" + conf.Password + "@tcp(" + conf.Host + ":3306)/" + conf.DBName
	db, err = sql.Open("mysql", param)
	if err != nil {
		panic(err.Error())    
	}
	defer db.Close()
	
	//Подключаемся к Telegram
	bot, err := tgbotapi.NewBotAPI(conf.TelegramBotToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}
	
	for update := range updates {

		//Читаем данные о пользователе, написавшем сообщение
		UserID := update.Message.From.ID
		ChatID := update.Message.Chat.ID
		FirstName := update.Message.From.FirstName
		LastName := update.Message.From.LastName
		UserName := FirstName + " " + LastName
		Text := update.Message.Text
		log.Printf("%s %d %d %s", UserName, UserID, ChatID, Text)

		//Определяем, есть ли пользователь в БД - если нет, то добавляем его в БД
		//если есть - рассылаем его сообщение всем пользователям из БД
		query = "SELECT count(*) FROM golangbot.users WHERE userid=" + strconv.Itoa(UserID);
		log.Printf(query)
		result, err := db.Query(query)
		if err != nil {
			log.Panic(err)
		}

		for result.Next() {
			err = result.Scan(&count)
			if err != nil {
				log.Panic(err)
			}
		}
		log.Printf("%d", count)
		
		if count == 0 {
			//Добавление пользователя в БД
			_, err = db.Exec("INSERT INTO users(userid, firstname, lastname) VALUES(?, ?, ?)", UserID, FirstName, LastName)
			if err != nil {
				log.Panic(err)
			}
			answer := "Добро пожаловать, " + FirstName + " " + LastName + "!"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, answer)
			bot.Send(msg)			
		} else {
				//Удаление пользователя
				if Text == "DELETEME" {
				_, err = db.Exec("DELETE FROM users WHERE userid=?", UserID)
				if err != nil {
					log.Panic(err)
				}
				answer := "Вы удалены из списка рассылки"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, answer)
				bot.Send(msg)
				log.Printf("Пользователь %d удален", UserID)
			} else {
				query = "select userid from golangbot.users";
				log.Printf(query)
				result, err := db.Query(query)
				if err != nil {
					log.Panic(err)
				}
				//Рассылка всем пользователям из базы данных
				for result.Next() {
					err = result.Scan(&id)
					if err != nil {
						log.Panic(err)
					}
					log.Printf("%d", id)
					msg := tgbotapi.NewMessage(id, Text)
					msg.ReplyToMessageID = update.Message.MessageID
					bot.Send(msg)				
				}
			}
		}
	}
}