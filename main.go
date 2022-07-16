package main

import (
	"encoding/json"
	"errors"
	. "fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// output text
//const any_error = "some mistake has occurred"
const incorrectFormatInput = "Incorrect command format!"
const errorCoversation = "Conversion error!"
const forAdd = "Added: %s %f. Balance: %f."
const forSub = "Subtracted: %s %f. Balance: %f."
const forDel = "Values for %s cleaned."
const forShow = "- for %s: %f.  (%.2f %s)\n"
const forShowSum = "Summa in %s: %.2f"
const notZeroCode = "Incorrect currency."
const noCurrency = "Incorrect currency."
const noCurrencyInWallet = "This currency is not in your wallet."
const urlBinance = "https://www.binance.com/api/v3/ticker/price?symbol=%s%s"

type binanceResp struct {
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}
type wallet map[string]float64

var db = map[int64]wallet{}

func main() {
	bot, err := tgbotapi.NewBotAPI("tg_token")
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		var text string
		var isInWallet bool
		chatId := update.Message.Chat.ID
		msgArr := strings.Split(update.Message.Text, " ")

		if _, ok := db[update.Message.Chat.ID]; !ok {
			db[chatId] = wallet{}
		}
		switch msgArr[0] {
		case "ADD":
			_, err = checkInput(chatId, msgArr, 3, true)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatId, err.Error()))
				continue
			}
			summa, err := strconv.ParseFloat(msgArr[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatId, errorCoversation))
				continue
			}
			db[chatId][msgArr[1]] += summa
			text = Sprintf(forAdd, msgArr[1], summa, db[chatId][msgArr[1]])
		case "SUB":
			isInWallet, err := checkInput(chatId, msgArr, 3, false)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatId, err.Error()))
				continue
			}
			if !isInWallet {
				bot.Send(tgbotapi.NewMessage(chatId, noCurrencyInWallet))
				continue
			}
			summa, err := strconv.ParseFloat(msgArr[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, errorCoversation))
				continue
			}
			db[chatId][msgArr[1]] -= summa
			text = Sprintf(forSub, msgArr[1], summa, db[chatId][msgArr[1]])
		case "DEL":
			isInWallet, err = checkInput(chatId, msgArr, 2, false)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatId, err.Error()))
				continue
			}
			if !isInWallet {
				bot.Send(tgbotapi.NewMessage(chatId, noCurrencyInWallet))
				continue
			}
			delete(db[chatId], msgArr[1])
			text = Sprintf(forDel, msgArr[1])
		case "SHOW":
			money := "USDT"
			_, err = checkInput(chatId, msgArr, 2, false)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(chatId, err.Error()))
				continue
			}

			var price, summa float64
			if len(msgArr) == 2 {
				money = msgArr[1]
			}
			text = "Balance:\n"
			for key, value := range db[chatId] {
				price, err = getPrice(key, money)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatId, err.Error()))
					continue
				}
				price *= value
				summa += price
				text += Sprintf(forShow, key, value, price, money)
			}
			text += Sprintf(forShowSum, money, summa)
		default:
			text = "I don't know this command." + update.Message.Text
		}
		msg := tgbotapi.NewMessage(chatId, text)
		bot.Send(msg)
	}
}

func checkInput(chatId int64, arr []string, needLen int, isCurrency bool) (isInWallet bool, err error) {
	switch arr[0] {
	case "SHOW":
		if len(arr) > needLen {
			err = errors.New(incorrectFormatInput)
			return
		}
		if len(arr) == 1 {
			return
		}
		return
	default:
		if len(arr) != needLen {
			err = errors.New(incorrectFormatInput)
			return
		}
	}
	_, isInWallet = db[chatId][arr[1]]
	if isCurrency {
		var resp *http.Response
		resp, err = http.Get(Sprintf(urlBinance, arr[1], "USDT"))
		if err != nil {
			return
		}
		defer resp.Body.Close()

		var jsonResp binanceResp
		err = json.NewDecoder(resp.Body).Decode(&jsonResp)
		if err != nil {
			return
		}
		if jsonResp.Code != 0 {
			err = errors.New("noCurrency")
			return
		}
	}
	return
}

func getPrice(coin, money string) (price float64, err error) {
	resp, err := http.Get(Sprintf(urlBinance, coin, money))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonResp binanceResp
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return
	}
	if jsonResp.Code != 0 {
		err = errors.New(notZeroCode)
		return
	}
	price = jsonResp.Price
	return
}
