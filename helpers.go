package slackbot

import (
	"sync/atomic"

	"golang.org/x/net/websocket"
)

// SendMessage sends a given message to a given channel.
func (bot SlackBot) SendMessage(channel string, message string) {
	atomic.AddInt32(&bot.messageID, 1)
	messageOut := &messageOut{
		ID:      bot.messageID,
		Type:    "message",
		Channel: channel,
		Text:    message,
	}
	websocket.JSON.Send(bot.ws, messageOut)
}
