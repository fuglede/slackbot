package slackbot

type event interface {
	invoke(bot *SlackBot) error // invoke the callback associated to a given event on the bot
}

var eventTypeByEvent = map[string]event{
	"hello":   &Hello{},
	"message": &MessageIn{},
	"pong":    &pingMessage{},
}

// Hello represents the event sent when a connection is opened to the message server.
// Slack API doc: https://api.slack.com/events/hello
type Hello struct {
	Type string `json:"type"`
}

func (event Hello) invoke(bot *SlackBot) (err error) {
	if bot.OnHello != nil {
		err = bot.OnHello(event)
	}
	return
}

func (event pingMessage) invoke(bot *SlackBot) (err error) {
	bot.lastPong = event.LastPing
	return nil
}

// MessageIn represents the event sent when a general message was sent to a channel.
// Slack API doc: https://api.slack.com/events/message
type MessageIn struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	User    string `json:"user"`
	Text    string `json:"text"`
	Ts      string `json:"ts"`
}

func (event MessageIn) invoke(bot *SlackBot) (err error) {
	if bot.OnMessage != nil {
		err = bot.OnMessage(event)
	}
	return
}
