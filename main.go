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

	"thomas-leister.de/plantmonitor/configmanager"
	"thomas-leister.de/plantmonitor/quantifier"
	"thomas-leister.de/plantmonitor/reminder"
)

/* Global var for config*/
var config configmanager.Config


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
	fmt.Printf("Connected to %s \n", config.Mqtt.Host)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v \n", err)
}

func runMQTTListener(mqttMessageChannel chan mqtt.Message) {
	opts := mqtt.NewClientOptions()

	// Set options for connection
	opts.AddBroker(fmt.Sprintf("mqtts://%s:%d", config.Mqtt.Host, config.Mqtt.Port))
	opts.SetClientID("go_mqtt_client")
	opts.SetUsername(config.Mqtt.Username)
	opts.SetPassword(config.Mqtt.Password)

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
	token := client.Subscribe(config.Mqtt.Topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic %s \n", config.Mqtt.Topic)
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
			Address: config.Xmpp.Host + ":" + strconv.Itoa(config.Xmpp.Port),
		},
		Jid:          config.Xmpp.Username,
		Credential:   xmpp.Password(config.Xmpp.Password),
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
		reply := stanza.Message{Attrs: stanza.Attrs{To: config.Xmpp.Recipient}, Body: xmppMessage}
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

func normalizeRawValue(rawValue int) int {
	// Normalize range
	rangeNormalizedValue := rawValue - config.Sensor.Adc.RawLowerBound

	// Normalize to percentage
	percentageValue := float32(rangeNormalizedValue) * (100 / (float32(config.Sensor.Adc.RawUpperBound) - float32(config.Sensor.Adc.RawLowerBound)))

	// Normalize meaning: Moisture rawValue is in fact "dryness" level: High => More dry. Low => more wet.
	// Let's invert that!
	percentageValueWetness := 100 - percentageValue

	// Return wetness percentage
	return int(percentageValueWetness)
}

func main() {
	var err error
	mqttMessageChannel := make(chan mqtt.Message)
	xmppMessageChannel := make(chan string)
	var historyExists bool = false

	fmt.Println(("Starting Plantmonitor ..."))

	// Read config
	config, err = configmanager.ReadConfig("config.yaml")
	if err != nil {
		fmt.Println("Could not parse config:", err)
		os.Exit(1)
	} else {
		fmt.Println("Config was read and parsed!")
	}

	// Init qauantifier
	myQuantifier := quantifier.Quantifier{}
	myQuantifier.Init(&config)

	// Init reminder engine
	myReminder := reminder.Reminder{}
	myReminder.Init(xmppMessageChannel)

	// Start a new Goroutine which listens for new messages and sents them over the mqttMessageChannel
	go runMQTTListener(mqttMessageChannel)

	// Start another Goroutine which sends XMPP messages
	go runXMPPClient(xmppMessageChannel)

	// Watch the channel and receive new messages
	for mqttMessage := range mqttMessageChannel {
		//fmt.Printf("Received message: %s from topic: %s\n", mqttMessage.Payload(), mqttMessage.Topic())
		mqttDecodedPayload := parseMqttMessage(mqttMessage)
		moistureRaw := mqttDecodedPayload.UplinkMessage.DecodedPayload.MoistureRaw

		// Normalize raw value to percentage (and invert value)
		normalizedMoistureValue := normalizeRawValue(int(moistureRaw))

		fmt.Printf("Raw value: %d  |  Normalized value: %d %% \n", moistureRaw, normalizedMoistureValue)

		// Put value into evaluation
		levelDirection, currentLevel, err := myQuantifier.EvaluateValue(normalizedMoistureValue)
		if err != nil {
			fmt.Printf("Error happended during evaluation.")
			break
		}

		if historyExists {
			// If value has changed and we have history, output a message
			if levelDirection == -1 {
				// Send normalized value and level name / level message
				fmt.Printf("Sending message: %s \n", currentLevel.ChatMessageDown)
				xmppMessageChannel <- fmt.Sprintf("%s \n\nBodenfeuchte: %d %%", currentLevel.ChatMessageDown, normalizedMoistureValue)
			} else if levelDirection == +1 {
				// Send normalized value and level name / level message
				fmt.Printf("Sending message: %s \n", currentLevel.ChatMessageUp)
				xmppMessageChannel <- fmt.Sprintf("%s \n\nBodenfeuchte: %d %%", currentLevel.ChatMessageUp, normalizedMoistureValue)
			} else if levelDirection == 0 {
				// Level has not changed (or there has been no history)
				fmt.Println("Level has not changed. Not sending any message (except for reminders).")
			}
		} else {
			// No history exists, yet (e.g. due to power-on). Just tell about the current state.
			fmt.Printf("Sending message: %s \n", currentLevel.ChatMessageInitial)
			xmppMessageChannel <- fmt.Sprintf("%s \n\nBodenfeuchte: %d %%", currentLevel.ChatMessageInitial, normalizedMoistureValue)
			historyExists = true
		}

		// Check for urgency.
		// If state demands urgent action, set a periodic reminder!
		if currentLevel.Urgency != quantifier.UrgencyLow {
			myReminder.Set(currentLevel)
		} else {
			// Do nothing. One message is enough.
			myReminder.Stop()
		}

	}

	fmt.Println("Plant monitor failed. Exiting ...")
}
