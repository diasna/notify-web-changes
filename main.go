package main

import (
	"encoding/json"
	"errors"
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
type Watch struct {
	Url      string `json:"url"`
	Selector string `json:"selector"`
}
type Configuration struct {
	Telegram  TelegramConfiguration `json:"telegram"`
	Interval  int                   `json:"interval_in_minutes"`
	WatchList []Watch               `json:"watch_list"`
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

func sendTelegramMessage(token string, chatId string, message string) bool {
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
	if res != nil {
		defer res.Body.Close()
		/*body, err := */ ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("cannot send telegram message, Status: %s", res.Status)
		}
		// log.Printf("Send notification response: %s\n", string(body))
		return true
	}
	return false
}

func extractNodes(url string, selector string) ([]string, error) {
	doc, err := htmlquery.LoadURL(url)
	if err != nil {
		log.Printf("cannot load url: %s", err.Error())
		return nil, errors.New("cannot load url: %s")
	}
	watch, err := htmlquery.QueryAll(doc, selector)
	if err != nil || watch == nil {
		log.Printf("cannot parse selector: %s", err.Error())
		return nil, errors.New("cannot parse selector: %s")
	}

	var extracted []string
	for _, n := range watch {
		if n != nil {
			extracted = append(extracted, fmt.Sprintf("%s(%s)\n", htmlquery.InnerText(n), htmlquery.SelectAttr(n, "class")))
		}
	}
	return extracted, nil
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

	var valueToWatch [10][10]string
	isNotificationSent := true

	for {
		time.Sleep(time.Duration(configuration.Interval) * time.Minute)
		var notificationMessage string
		var valueChanged bool
		for i, n := range configuration.WatchList {
			extracted, err := extractNodes(n.Url, n.Selector)
			if err != nil {
				sendTelegramMessage(configuration.Telegram.Token, configuration.Telegram.ChatId, "cannot parse")
				continue
			}
			for j, m := range extracted {
				if valueToWatch[i][j] != m {
					notificationMessage += m
					valueChanged = true
					log.Printf("%s => %s\n", valueToWatch[i][j], m)
				}
				valueToWatch[i][j] = m
			}
		}
		if valueChanged || !isNotificationSent {
			isNotificationSent = sendTelegramMessage(configuration.Telegram.Token, configuration.Telegram.ChatId, notificationMessage)
		} else {
			log.Println("No change")
		}
		valueChanged = false
	}
}
