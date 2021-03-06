// Package slackbot implements part of the Slack Real Time Messaging (RTM) API,
// with the intent of exposing a simple interface for creating a chatbot that
// can be provided with callbacks to respond to given events on a Slack channel.
package slackbot

import (
	"log"

	"golang.org/x/net/websocket"
)

// SlackBot represents a single connection of a Slack bot user
// to the Slack Real Time Messaging (RTM) API. The bot should be
// initialized through a call to New after which a connection can
// be started through a call to Start.
type SlackBot struct {
	CallbackErrors chan error // Signals errors seen on user-defined callbacks
	Done           chan bool  // Signals that the bot has disconnected

	// Event callbacks. These should be defined by the client and will
	// be called when the bot encounters the relevant events.
	OnDndUpdatedUser func(event DndUpdatedUser) error // Do not disturb settings changed for a team member
	OnHello          func(event Hello) error          // The client has successfully connected to the server
	OnMessage        func(event MessageIn) error      // A message was sent to a channel
	OnPresenceChange func(event PresenceChange) error // A team member's presence changed

	id           string // The Slack ID of the bot itself
	name         string // The name identifying the bot on Slack
	messageID    int32  // Counter to ensure that messages are sent with unique IDs
	lastPing     int32  // Counter to ensure that pings are sent with unique IDs
	lastPong     int32  // ID of the last pong message received
	disconnected bool   // Is true if the WebSocket connection has been closed

	logger *log.Logger     // Logger used for status reports
	ws     *websocket.Conn // The WebSocket connection on which all communication happens
}

// New creates a new SlackBot with a predefined logger.
func New(logger *log.Logger) *SlackBot {
	return &SlackBot{
		CallbackErrors: make(chan error),
		Done:           make(chan bool),
		logger:         logger,
		messageID:      0,
		disconnected:   false,
	}
}
