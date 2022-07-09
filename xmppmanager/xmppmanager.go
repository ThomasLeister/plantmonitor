/*
 * XmppManager: Manages XMPP connection and
 * offers xmppMessageChannel for sending various types of XMPP Messages:
 * 		- XmppTextMessage or
 * 		- XmppGifMessage
 */

package xmppmanager

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
	"thomas-leister.de/plantmonitor/configmanager"
)

type XmppTextMessage struct {
	Recipients []string
	Text       string
}

type XmppGifMessage struct {
	Recipients []string
	Url        string
}

// General incoming XMPP message
type XmppInMessage struct {
	From string
	Body string
}

type XmppClient struct {
	Host                  string
	Port                  int
	Username              string
	Password              string
	Recipients            []string
	XmppMessageOutChannel chan interface{}
	XmppMessageInChannel  chan XmppInMessage
}

func (x *XmppClient) HandleXmppMessage(s xmpp.Sender, p stanza.Packet) {
	msg, ok := p.(stanza.Message)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", p)
		return
	}

	inMsg := XmppInMessage{From: msg.From, Body: msg.Body}

	// Just feed messages with Body into messenger responder. Not "typing" notifications etc.
	if msg.Body != "" {
		x.XmppMessageInChannel <- inMsg
	}
}

func (x *XmppClient) XmppErrorHandler(err error) {
	fmt.Println(err.Error())
}

func (x *XmppClient) Init(config *configmanager.Config) error {
	log.Println("Initializing xmppmanager ...")

	x.Host = config.Xmpp.Host
	x.Port = config.Xmpp.Port
	x.Username = config.Xmpp.Username
	x.Password = config.Xmpp.Password
	x.Recipients = config.Xmpp.Recipients

	return nil
}

func (x *XmppClient) RunXMPPClient(xmppMessageOutChannel chan interface{}, xmppMessageInChannel chan XmppInMessage) {
	x.XmppMessageOutChannel = xmppMessageOutChannel
	x.XmppMessageInChannel = xmppMessageInChannel

	xmppClientConfig := xmpp.Config{
		TransportConfiguration: xmpp.TransportConfiguration{
			Address: x.Host + ":" + strconv.Itoa(x.Port),
		},
		Jid:          x.Username,
		Credential:   xmpp.Password(x.Password),
		StreamLogger: nil,
		Insecure:     false,
	}

	router := xmpp.NewRouter()
	router.HandleFunc("message", x.HandleXmppMessage)

	client, err := xmpp.NewClient(&xmppClientConfig, router, x.XmppErrorHandler)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// If you pass the client to a connection manager, it will handle the reconnect policy
	// for you automatically.
	cm := xmpp.NewStreamManager(client, nil)
	go cm.Run()

	// Wait for a new message to send (listen on channel)
	for xmppMessage := range xmppMessageOutChannel {
		xmppMessageStanza := stanza.Message{}
		var recipients []string

		// Find out stanza type (TextMessage or GifMessage)
		switch xmppMessage.(type) {
		case XmppTextMessage:
			log.Println("XMPP: Sending a text message")
			m := xmppMessage.(XmppTextMessage)

			recipients = m.Recipients
			xmppMessageStanza = stanza.Message{
				Body: string(m.Text),
			}

		case XmppGifMessage:
			log.Println("XMPP: Sending a GIF message")
			m := xmppMessage.(XmppGifMessage)

			recipients = m.Recipients
			xmppMessageStanza = stanza.Message{
				Extensions: []stanza.MsgExtension{
					stanza.OOB{
						URL:  string(m.Url),
						Desc: "GIF with meme",
					},
				},
			}

		default:
			log.Println("ERROR: Type of message to send is unknown. Send one of XmppTextMessage or XmppGifMessage!")
			continue // Quit this for() round
		}

		// If recipients are undefined, broadcast to all users
		if len(recipients) == 0 {
			recipients = x.Recipients
		}

		// For each recipient: Set recipient in stanza and send message
		for _, recipient := range recipients {
			xmppMessageStanza.Attrs = stanza.Attrs{To: recipient}

			err := client.Send(xmppMessageStanza)
			if err != nil {
				log.Println("ERROR: Could not send stanza to: ", err)
			}
		}
	}
}
