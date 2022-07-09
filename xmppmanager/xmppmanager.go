package xmppmanager 

import (
	"fmt"
	"os"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
	"strconv"
	"log"
	"thomas-leister.de/plantmonitor/configmanager"
)

type XmppTextMessage string

type XmppGifMessage string

type XmppClient struct {
	Host string 
	Port int 
	Username string 
	Password string 
	Recipient string
}

func (x *XmppClient) HandleXmppMessage(s xmpp.Sender, p stanza.Packet) {
	msg, ok := p.(stanza.Message)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", p)
		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Body = %s - from = %s\n", msg.Body, msg.From)
	reply := stanza.Message{Attrs: stanza.Attrs{To: msg.From}, Body: msg.Body}
	_ = s.Send(reply)
}

func (x *XmppClient) XmppErrorHandler(err error) {
	fmt.Println(err.Error())
}

func (x *XmppClient) Init(config *configmanager.Config) error {
	x.Host = config.Xmpp.Host
	x.Port = config.Xmpp.Port
	x.Username = config.Xmpp.Username
	x.Password = config.Xmpp.Password
	x.Recipient = config.Xmpp.Recipient

	return nil
}

func (x *XmppClient) RunXMPPClient(xmppMessageChannel chan interface{}) {
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
	for xmppMessage := range xmppMessageChannel {
		xmppMessageStanza := stanza.Message{}

		// Find out stanza type (TextMessage or GifMessage)
		switch xmppMessage.(type) {
		case XmppGifMessage:
			fmt.Println("XMPP: Sending a GIF message")
			gm := xmppMessage.(XmppGifMessage)

			xmppMessageStanza = stanza.Message{
				Attrs: stanza.Attrs{To: x.Recipient}, 
				Extensions: []stanza.MsgExtension{
					stanza.OOB{
						URL: string(gm),
						Desc: "meme",
					},
				},
			}

		case XmppTextMessage:
			fmt.Println("XMPP: Sending a text message")
			tm := xmppMessage.(XmppTextMessage)
			xmppMessageStanza = stanza.Message{Attrs: stanza.Attrs{To: x.Recipient}, Body: string(tm)}
		
		default:
			fmt.Println("ERROR: Type of message to send is unknown. Send one of XmppTextMessage or XmppGifMessage!")
			continue // Quit this for() round
		}

		err := client.Send(xmppMessageStanza)
		if err != nil {
			fmt.Println("Error sending: ", err)
		}
	}
}
