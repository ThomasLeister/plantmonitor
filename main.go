package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
)

const MQTT_SERVER = "eu1.cloud.thethings.network"
const MQTT_PORT = 8883
const MQTT_USERNAME = "pico-lora-moisture@ttn"
const MQTT_PASSWORD = "NNSXS.QCVGWZBLXFLLNKL7SQARNKOPXFY6D7TTBNWA6EI.G7PBTBGTFXC56EPHZDZY532QHWHYCLZCU7KGQ43VZ2ZALETWFZ2Q"
const MQTT_TOPIC = "v3/pico-lora-moisture@ttn/devices/+/up"

const XMPP_SERVER = "xmpp.trashserver.net"
const XMPP_PORT = 5222
const XMPP_USERNAME = "fritzpflanze@trashserver.net"
const XMPP_PASSWORD = "fritzpflanze"
const XMPP_RECIPIENT = "thomas.privat@trashserver.net"

type MqttDecodedPayload struct {
	MoistureRaw uint16 `json:"moisture_raw"`
}

type MqttUplinkMessage struct {
	DecodedPayload MqttDecodedPayload `json:"decoded_payload"` //decoded_payload stores the already-decoded payload from TTN
}

type MqttPayload struct {
	UplinkMessage MqttUplinkMessage `json:"uplink_message"`
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Printf("Connected to %s \n", MQTT_SERVER)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v \n", err)
}

func runMQTTListener(mqttMessageChannel chan mqtt.Message) {
	opts := mqtt.NewClientOptions()

	// Set options for connection
	opts.AddBroker(fmt.Sprintf("mqtts://%s:%d", MQTT_SERVER, MQTT_PORT))
	opts.SetClientID("go_mqtt_client")
	opts.SetUsername(MQTT_USERNAME)
	opts.SetPassword(MQTT_PASSWORD)

	// Set callback functions
	opts.SetDefaultPublishHandler(func(c mqtt.Client, m mqtt.Message) {
		mqttMessageChannel <- m
	})
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	// Create client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to topic
	token := client.Subscribe(MQTT_TOPIC, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic %s \n", MQTT_TOPIC)
}

func handleXmppMessage(s xmpp.Sender, p stanza.Packet) {
	msg, ok := p.(stanza.Message)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", p)
		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Body = %s - from = %s\n", msg.Body, msg.From)
	reply := stanza.Message{Attrs: stanza.Attrs{To: msg.From}, Body: msg.Body}
	_ = s.Send(reply)
}

func xmppErrorHandler(err error) {
	fmt.Println(err.Error())
}

func runXMPPClient(xmppMessageChannel chan string) {
	xmppClientConfig := xmpp.Config{
		TransportConfiguration: xmpp.TransportConfiguration{
			Address: XMPP_SERVER + ":" + strconv.Itoa(XMPP_PORT),
		},
		Jid:          XMPP_USERNAME,
		Credential:   xmpp.Password(XMPP_PASSWORD),
		StreamLogger: nil,
		Insecure:     false,
	}

	router := xmpp.NewRouter()
	router.HandleFunc("message", handleXmppMessage)

	client, err := xmpp.NewClient(&xmppClientConfig, router, xmppErrorHandler)
	if err != nil {
		log.Fatalf("%+v", err)
	}

	// If you pass the client to a connection manager, it will handle the reconnect policy
	// for you automatically.
	cm := xmpp.NewStreamManager(client, nil)
	go cm.Run()

	// Wait for a new message to send (listen on channel)
	for xmppMessage := range xmppMessageChannel {
		fmt.Println("Sending message via XMPP")
		reply := stanza.Message{Attrs: stanza.Attrs{To: XMPP_RECIPIENT}, Body: xmppMessage}
		err := client.Send(reply)
		if err != nil {
			fmt.Println("Error sending: ", err)
		}
	}
}

func parseMqttMessage(mqttMessage mqtt.Message) MqttPayload {
	var mqttPayload MqttPayload

	err := json.Unmarshal(mqttMessage.Payload(), &mqttPayload)
	if err != nil {
		panic(err)
	}

	return mqttPayload
}

func main() {
	mqttMessageChannel := make(chan mqtt.Message)
	xmppMessageChannel := make(chan string)

	fmt.Println(("Starting Plantmonitor ..."))

	// Start a new Goroutine which listens for new messages and sents them over the mqttMessageChannel
	go runMQTTListener(mqttMessageChannel)

	// Start another Goroutine which sends XMPP messages
	go runXMPPClient(xmppMessageChannel)

	// Watch the channel and receive new messages
	for mqttMessage := range mqttMessageChannel {
		//fmt.Printf("Received message: %s from topic: %s\n", mqttMessage.Payload(), mqttMessage.Topic())
		mqttDecodedPayload := parseMqttMessage(mqttMessage)

		moistureRaw := mqttDecodedPayload.UplinkMessage.DecodedPayload.MoistureRaw
		fmt.Printf("Moisture raw value: %d\n", moistureRaw)
		xmppMessageChannel <- "New moisture value: " + strconv.Itoa(int(moistureRaw))
	}

	fmt.Println("Plant monitor failed. Exiting ...")
}
