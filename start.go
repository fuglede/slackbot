package slackbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

// Start opens a WebSocket connection to Slack and starts listening
// for messages.
func (bot *SlackBot) Start(token string) (err error) {
	url := "https://slack.com/api/rtm.connect?token=" + token
	bot.logger.Println("Getting websocket URL from Slack web API")
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf("API websocket URL request failed with code %d", resp.StatusCode)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}
	var msg connectMessage
	err = json.Unmarshal(body, &msg)
	if err != nil {
		return
	}

	if !msg.Ok {
		err = fmt.Errorf("Slack error: %s", msg.Error)
		return
	}
	bot.id = msg.Self.ID
	bot.name = msg.Self.Name

	bot.ws, err = websocket.Dial(msg.URL, "", "https://api.slack.com/")
	go bot.listen()
	return
}

// connectMessage represents a response sent by the Slack Web API method
// rtm.connect. It is documented at https://api.slack.com/methods/rtm.connect
type connectMessage struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
	URL   string `json:"url"`
	Team  struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Domain         string `json:"domain"`
		EnterpriseID   string `json:"enterprise_id"`
		EnterpriseName string `json:"enterprise_name"`
	} `json:"team"`
	Self struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"self"`
}

// listen will continuously parse messages from the bot's RTM connection
// and spawn handlers for each of them. It disconnects the bot if any
// errors occur.
func (bot SlackBot) listen() (err error) {
	for {
		event := json.RawMessage{}
		err = websocket.JSON.Receive(bot.ws, &event)
		if err != nil {
			log.Print("Error receiving JSON from websocket :", err)
			break
		}
		go bot.handleEvent(event)
	}
	bot.Disconnect()
	return
}

// Disconnect closes the WebSocket connection and signals completion
// on the Done channel.
func (bot SlackBot) Disconnect() error {
	bot.Done <- true
	return bot.ws.Close()
}

// event represents a generic message received from the Slack RTM API. It
// is documented at https://api.slack.com/rtm
type event struct {
	Type string `json:"type"`
}

// handleEvent parses a general Slack event into its specific type
// and calls the relevant callbacks.
func (bot SlackBot) handleEvent(rawEvent json.RawMessage) {
	var event event
	json.Unmarshal(rawEvent, &event)
	switch event.Type {
	case "hello":
		if bot.OnHello != nil {
			err := bot.OnHello()
			if err != nil {
				bot.CallbackErrors <- err
			}
		}
	case "message":
		var message MessageIn
		json.Unmarshal(rawEvent, &message)
		if bot.OnMessage != nil {
			err := bot.OnMessage(message)
			if err != nil {
				bot.CallbackErrors <- err
			}
		}
	}
}
