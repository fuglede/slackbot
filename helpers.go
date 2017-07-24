package slackbot

import (
	"sync/atomic"

	"golang.org/x/net/websocket"
)

// SendMessage sends a given message to a given channel.
func (bot SlackBot) SendMessage(channel string, message string) error {
	atomic.AddInt32(&bot.messageID, 1)
	bot.logger.Printf("Sending message %s to channel %s\n", message, channel)
	messageOut := &messageOut{
		ID:      bot.messageID,
		Type:    "message",
		Channel: channel,
		Text:    message,
	}
	return websocket.JSON.Send(bot.ws, messageOut)
}

// messageOut represents an outbound message
// Slack API doc: https://api.slack.com/rtm
type messageOut struct {
	ID      int32  `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}
