slackbot
========

This library provides a very partial Go implementation of the [Slack RTM API](https://api.slack.com/rtm) with an eye towards creating chatbots for Slack.

About a 1000 similar libraries exist, and if you came here searching for a implementation of the full API, you may as well head somewhere else; [this repository](https://github.com/nlopes/slack) looks promising!

Instead, in this rudimentary approach, all we do is connect to the WebSocket server and allow the user of a bot to set up simple read-and-respond scripts through callbacks, which happened to be what I needed when setting this up; the repository is kept here mainly for my personal bookkeeping, and to allow others who happen to come across it to get started.


Installation
------------

First of all, to get started you'll need an API token for your bot user; you can get that at https://your-team-name.slack.com/apps/manage/custom-integrations.

With that out of the way, an example script says more than a thousand words:

    package main

    import (
        "errors"
        "fmt"
        "log"
        "os"

        "github.com/fuglede/slackbot"
    )

    func main() {
        token := "xoxb-your-api-token-goes-here"

        // Create a new bot and let it log to stdout
        bot := slackbot.New(log.New(os.Stdout, "", 3))

        // Set up the bot to respond to the message "Hi". The callback
        // returns an error which we will catch later.
        bot.OnMessage = func(message slackbot.MessageIn) error {
            if message.Text == "Hi" {
                bot.SendMessage(message.Channel, "Hi!")
            }
            return errors.New("Error only for illustration")
        }

        // Connect and start listening for messages
        bot.Start(token)

        for {
            select {
            // Exit the application when the bot signals disconnection
            case <-bot.Done:
                return
            // Use the CallbackErrors channel to catch the "error" from above
            case err := <-bot.CallbackErrors:
                fmt.Println(err)
            }
        }
    }
