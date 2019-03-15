package slackbot

type event interface {
	invoke(bot *SlackBot) error // invoke the callback associated to a given event on the bot
}

func makeEventByType(eventType string) (event, bool) {
	var eventTypeByEvent = map[string]event{
		"dnd_updated_user": &DndUpdatedUser{},
		"hello":            &Hello{},
		"message":          &MessageIn{},
		"pong":             &pongMessage{},
		"presence_change":  &PresenceChange{},
	}
	event, exists := eventTypeByEvent[eventType]
	return event, exists
}

// DndUpdatedUser represents the event sent when o not Disturb settings change for a team member
// Slack API doc: https://api.slack.com/events/dnd_updated_user
type DndUpdatedUser struct {
	Type      string `json:"type"`
	User      string `json:"user"`
	DndStatus struct {
		DndEnabled     bool `json:"dnd_enabled"`
		NextDndStartTs int  `json:"next_dnd_start_ts"`
		NextDndEndTs   int  `json:"next_dnd_end_ts"`
	} `json:"dnd_status"`
}

func (event DndUpdatedUser) invoke(bot *SlackBot) (err error) {
	if bot.OnDndUpdatedUser != nil {
		err = bot.OnDndUpdatedUser(event)
	}
	return
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

type pongMessage struct {
	ReplyTo int32  `json:"reply_to"`
	Type    string `json:"type"`
}

func (event pongMessage) invoke(bot *SlackBot) (err error) {
	bot.lastPong = event.ReplyTo
	return nil
}

// MessageIn represents the event sent when a general message was sent to a channel.
// Slack API doc: https://api.slack.com/events/message
type MessageIn struct {
	Type    string `json:"type"`
	Hidden  bool   `json:"hidden"`
	Channel string `json:"channel"`
	User    string `json:"user"`
	Text    string `json:"text"`
	Ts      string `json:"ts"`
}

func (event MessageIn) invoke(bot *SlackBot) (err error) {
	// The Slack API defines some messages as "Hidden". This includes edits and deletes,
	// which we will want to ignore here.
	if bot.OnMessage != nil && !event.Hidden {
		err = bot.OnMessage(event)
	}
	return
}

// PresenceChange represents the event sent when a team member's presence has changed.
// Slack API doc: https://api.slack.com/events/presence_change
type PresenceChange struct {
	Type     string `json:"type"`
	User     string `json:"user"`
	Presence string `json:"presence"`
}

func (event PresenceChange) invoke(bot *SlackBot) (err error) {
	if bot.OnPresenceChange != nil {
		err = bot.OnPresenceChange(event)
	}
	return
}
