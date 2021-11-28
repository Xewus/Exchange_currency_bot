package main

import (
	"encoding/json"
	"errors"
	. "fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"strconv"
	"strings"
)
// output text
const any_error = "some mistake has occurred"
const incorrect_format_input = "Incorrect command format!"
const error_coversation = "Conversion error!"
const for_add = "Added: %s %f. Balance: %f"
const for_cut = "Subtracted: %s %f. Balance: %f"
const for_del = "Values for %s cleaned. Balance: %f"
const for_show = "- for %s: %f"
const not_zero_code = "Incorrect currency"
const no_currency = "Incorrect currency"

type binanceResp struct {
	Price float64 `json:"price,string"`
	Code int64 `json:"code"`
}

type wallet map[string]float64

var db = map[int64]wallet{}

func main() {
	bot, err := tgbotapi.NewBotAPI("1979585498:AAHAtPDheiR4rxmf5BIcR1Y-p0QAEONRsGU")
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
		msgArr := strings.Split(update.Message.Text, " ")

		switch msgArr[0] {
		case "ADD":
			err := checkInput(msgArr)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			summa, err := strconv.ParseFloat(msgArr[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, error_coversation))
				continue
			}
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}
				db[update.Message.Chat.ID][msgArr[1]] += summa
			text = Sprintf(for_add, msgArr[1], summa, db[update.Message.Chat.ID][msgArr[1]])
		case "SUB":

			err := checkInput(msgArr)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			summa, err := strconv.ParseFloat(msgArr[2], 64)
			if err != nil{
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, error_coversation))
				continue
			}
			db[update.Message.Chat.ID][msgArr[1]] -= summa
			text = Sprintf(for_cut, msgArr[1], summa, db[update.Message.Chat.ID][msgArr[1]])
		case "DEL":
			db[update.Message.Chat.ID][msgArr[1]] = 0
			text = Sprintf(for_del, msgArr[1], db[update.Message.Chat.ID][msgArr[1]])
		case "SHOW":
			var priceUsd, sumRub, usd float64
			text = "Balance:\n"
			for key, value := range db[update.Message.Chat.ID] {
				priceUsd, err = getPrice(key)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
					continue
				}
				usd = value * priceUsd
				sumRub += usd
				text += (Sprintf(for_show, key, value) + Sprintf(" (%.2f RUB)\n", usd))
		    }
			text += Sprintf("Summa: %.2f RUB", sumRub)
		default:
			text = "I don't know this command." + update.Message.Text
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)

		bot.Send(msg)
	}
}

func checkInput(arr []string) (err error) {
	if (arr[0] == "ADD" || arr[0] == "SUB") && len(arr) != 3 {
		err = errors.New(incorrect_format_input)
		return
	}

	resp, err := http.Get(Sprintf("https://www.binance.com/api/v3/ticker/price?symbol=%sUSDT", arr[1]))
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
		err = errors.New(no_currency)
		return
	}
	return
}

func  getPrice(coin string) (price float64, err error) {
	respUsd, err :=http.Get(Sprintf("https://www.binance.com/api/v3/ticker/price?symbol=%sUSDT", coin))
	if err != nil {
		return
	}
	respRub, err :=http.Get("https://www.binance.com/api/v3/ticker/price?symbol=USDTRUB")
	if err != nil {
		return
	}
	defer respUsd.Body.Close()
	defer respRub.Body.Close()

	var jsonResp binanceResp
	err = json.NewDecoder(respUsd.Body).Decode(&jsonResp)
	if err != nil {
		return
	}
	if jsonResp.Code != 0 {
		err = errors.New(not_zero_code)
		return
	}
	price = jsonResp.Price

	err = json.NewDecoder(respRub.Body).Decode(&jsonResp)
	if err != nil {
		return
	}
	if jsonResp.Code != 0 {
		err = errors.New(not_zero_code)
		return
	}
	price *= jsonResp.Price
	return
}
