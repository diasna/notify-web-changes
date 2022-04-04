package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

type TelegramConfiguration struct {
	Token  string `json:"token"`
	ChatId string `json:"chat_id"`
}
type MainConfiguration struct {
	Url      string `json:"url"`
	Selector string `json:"selector"`
	Interval int    `json:"interval_in_minutes"`
}
type Configuration struct {
	Main     MainConfiguration     `json:"main"`
	Telegram TelegramConfiguration `json:"telegram"`
}

func loadConfiguration(path string) Configuration {
	file, _ := os.Open(path)
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Fatalf("cannot load configuration: %s", err.Error())
	}
	return configuration
}

func sendTelegramMessage(token string, chatId string, message string) {
	var endpoint = fmt.Sprintf("https://api.telegram.org/%s/sendMessage", token)
	data := url.Values{}
	data.Set("chat_id", chatId)
	data.Set("text", message)

	client := &http.Client{}
	r, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("cannot send telegram message: %s", err.Error())
	}
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	res, err := client.Do(r)
	if err != nil {
		log.Printf("cannot send telegram message: %s", err.Error())
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("cannot send telegram message, Status: %s", res.Status)
	}
	log.Printf("Send notification response: %s\n", string(body))
}

func main() {
	var configFileLocation string
	flag.StringVar(&configFileLocation, "config", "config.json", "config file location")
	flag.Parse()

	if len(configFileLocation) <= 0 {
		log.Fatalf("missing argument config")
	}
	configuration := loadConfiguration(configFileLocation)

	sendTelegramMessage(configuration.Telegram.Token, configuration.Telegram.ChatId, "service notify-web-changes started")

	var oldValue = ""
	for {
		time.Sleep(time.Duration(configuration.Main.Interval) * time.Minute)
		doc, err := htmlquery.LoadURL(configuration.Main.Url)
		if err != nil {
			log.Printf("cannot load url: %s", err.Error())
			continue
		}
		watch, err := htmlquery.Query(doc, configuration.Main.Selector)
		if err != nil {
			log.Printf("cannot parse selector: %s", err.Error())
			continue
		}
		newValue := htmlquery.OutputHTML(watch, true)
		if oldValue == "" || oldValue == newValue {
			log.Println("No value change detected")
			oldValue = newValue
		} else {
			log.Printf("Value %s has change to %s", oldValue, newValue)
			oldValue = newValue
			sendTelegramMessage(configuration.Telegram.Token, configuration.Telegram.ChatId, newValue)
		}
	}
}
