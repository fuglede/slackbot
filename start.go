package slackbot

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
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
	bot.ws, err = bot.dial(msg.URL)
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

// dial opens a WebSocket connection at a given URL, stripping all SNI information
// in the process.
func (bot SlackBot) dial(url string) (*websocket.Conn, error) {
	// Rather than connecting directly to the host provided by Slack, we resolve
	// its IP and connect to that instead. Effectively, this strips SNI information
	// from the TLS packets, which allows the bot to work in environments in which
	// firewalls employ packet inspection to block frames based on SNI. This only
	// works because the server at the other end does not actually make use of SNI,
	// so that removing it becomes safe.
	hostRegExp := regexp.MustCompile("//([^/]+)/")
	host := hostRegExp.FindStringSubmatch(url)[1]
	addrs, err := net.LookupHost(host)
	if err != nil {
		return nil, fmt.Errorf("could not resolve address of %s: %v", host, err)
	}
	ip := addrs[0]
	newURL := strings.Replace(url, host, ip, 1)
	bot.logger.Println("Connecting to WebSocket at " + url)
	config, err := websocket.NewConfig(newURL, "https://api.slack.com/")
	// As we have removed the hostname, the Go TLS package will not know what to
	// validate the certificate DNS names against, so we have to provide a custom
	// verifier based on the hostname we threw away. In the particular case of
	// Slack, this happens to be rather straightforward as no intermediate certificates
	// are provided; that is, the leaf is signed directly by the CA.
	config.TlsConfig = &tls.Config{
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: verifier(host),
	}
	return websocket.DialConfig(config)
}

// verifier produces a certificate validating callback in which it is required that the first
// certificate has as its DNSName a given host.
func verifier(host string) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		opts := x509.VerifyOptions{DNSName: host}
		rawCert := rawCerts[0]
		cert, err := x509.ParseCertificate(rawCert)

		if err != nil {
			return err
		}
		_, err = cert.Verify(opts)
		return err
	}
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
	event, exists := eventTypeByEvent[firstPassEvent.Type]
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
