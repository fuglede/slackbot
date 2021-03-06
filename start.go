package slackbot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

// Start opens a WebSocket connection to Slack and starts listening
// for messages.
func (bot *SlackBot) Start(token string) (err error) {
	msg, err := bot.getConnectionInformation(token)
	if err != nil {
		return
	}
	bot.id = msg.Self.ID
	bot.name = msg.Self.Name
	bot.ws, err = websocket.Dial(msg.URL, "", "https://api.slack.com/")
	if err != nil {
		return
	}
	bot.logger.Println("Connected. Listening for events.")
	go bot.listen()
	// The standard Go WebSocket library does not support WebSocket pings,
	// but Slack provides a custom heartbeat mechanism that we use here instead
	go bot.sendPings()
	return
}

// getConnectionInformation performs the initial call to the Slack HTTP API,
// which gets us the bot's ID and name, as well as a URL for opening a
// WebSocket connection.
func (bot SlackBot) getConnectionInformation(token string) (msg connectMessage, err error) {
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
	if err = json.Unmarshal(body, &msg); err != nil {
		return
	}

	if !msg.Ok {
		err = fmt.Errorf("Slack error: %s", msg.Error)
	}
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
func (bot *SlackBot) listen() (err error) {
	for {
		event := json.RawMessage{}
		err = websocket.JSON.Receive(bot.ws, &event)
		if err != nil {
			bot.logger.Print("Error receiving JSON from websocket :", err)
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
	if !bot.disconnected {
		bot.logger.Println("Disconnecting.")
		bot.Done <- true
		bot.disconnected = true
		return bot.ws.Close()
	}
	return errors.New("bot is already disconnected")
}

// typeOnlyEvent represents a generic message received from the Slack RTM API. It
// is documented at https://api.slack.com/rtm
type typeOnlyEvent struct {
	Type string `json:"type"`
}

// handleEvent parses a general Slack event into its specific type
// and calls the relevant callbacks.
func (bot *SlackBot) handleEvent(rawEvent json.RawMessage) {
	// We unmarshal in two steps. First, we get the type of the event.
	var firstPassEvent typeOnlyEvent
	json.Unmarshal(rawEvent, &firstPassEvent)
	bot.logger.Println("Received event: " + string(rawEvent))
	// Now we have the type and can unmarshal into that type
	event, exists := makeEventByType(firstPassEvent.Type)
	if !exists {
		return
	}
	json.Unmarshal(rawEvent, &event)
	err := event.invoke(bot)
	if err != nil {
		bot.CallbackErrors <- err
	}
}

// sendPings sends a ping every minute, ensures that pongs are returned,
// and disconnects when they are not.
func (bot *SlackBot) sendPings() (err error) {
	for {
		// If more than three minutes passed since the last pong, disconnect.
		if bot.lastPing-bot.lastPong > 2 {
			return bot.Disconnect()
		}
		bot.lastPing++
		pingMessage := pingMessage{ID: bot.lastPing, Type: "ping"}
		websocket.JSON.Send(bot.ws, pingMessage)
		time.Sleep(time.Minute)
	}
}

// pingMessage represents the message used for pinging/ponging. It is documented
// at https://api.slack.com/rtm
type pingMessage struct {
	ID   int32  `json:"id"`
	Type string `json:"type"`
}
