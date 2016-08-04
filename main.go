package main

import (

	r "github.com/dancannon/gorethink"
	"log"
	"io/ioutil"
	"encoding/json"
	"strings"
	"fmt"
	"net/http"
	"bytes"
)

type Config struct {
	Slack struct{ Host, Channel, Nickname string }
	DB    struct{ Host string }
}

type Issue struct {
	Description, Type string
}

func sendNotification(message string, config Config) {
	url := config.Slack.Host
	payload := []byte(fmt.Sprintf(`{"text": "%v", "channel": "%s"}`, message, config.Slack.Channel))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Unable to send notification:", err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
}

func main() {
	quit := make(chan bool, 1)
	var config Config

	file, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal("Unable to read configuration file:", err)
	}
	if err := json.Unmarshal(file, &config); err != nil {
		log.Fatal("Unable to parse configuration:", err)
	}

	db, err := r.Connect(r.ConnectOpts{Address: config.DB.Host})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	issues, err := r.DB("rethinkdb").Table("current_issues").Filter(
		r.Row.Field("critical").Eq(true)).Changes().Field("new_val").Run(db)

	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	go func() {
		var issue Issue
		for issues.Next(&issue) {
			if issue.Type != "" {
				text := strings.Split(issue.Description, "\n")[0]
				message := fmt.Sprintf("(%s)\n %s", issue.Type, text)
				sendNotification(message, config)
			}
		}
	}()

	// block main goroutine forever
	<-quit
}
