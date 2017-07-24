package slackbot

// MessageIn represents a general incoming message.
// Slack API doc: https://api.slack.com/events/message
type MessageIn struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	User    string `json:"user"`
	Text    string `json:"text"`
	Ts      string `json:"ts"`
}

// messageOut represents an outbound message
// Slack API doc: https://api.slack.com/rtm
type messageOut struct {
	ID      int32  `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}
