package main

import (
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"thomas-leister.de/plantmonitor/configmanager"
	"thomas-leister.de/plantmonitor/quantifier"
	"thomas-leister.de/plantmonitor/reminder"
	"thomas-leister.de/plantmonitor/giphy"
	"thomas-leister.de/plantmonitor/xmppmanager"
	"thomas-leister.de/plantmonitor/mqttmanager"
	"thomas-leister.de/plantmonitor/messenger"
)

/* Global var for config*/
var config configmanager.Config


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
	xmppMessageChannel := make(chan interface{})

	fmt.Println(("Starting Plantmonitor ..."))

	// Read config
	config, err = configmanager.ReadConfig("config.yaml")
	if err != nil {
		fmt.Println("Could not parse config:", err)
		os.Exit(1)
	} else {
		fmt.Println("Config was read and parsed!")
	}

	// Init xmppmanager 
	xmppclient := xmppmanager.XmppClient{}
	xmppclient.Init(&config)

	// Init mqttmanager
	mqttclient := mqttmanager.MqttClient{}
	mqttclient.Init(&config)

	// Init Giphy
	giphyClient := giphy.GiphyClient{}
	giphyClient.Init(config.Giphy.ApiKey)

	// Init quantifier
	myQuantifier := quantifier.Quantifier{}
	myQuantifier.Init(&config)

	// Init reminder engine
	myReminder := reminder.Reminder{}
	myReminder.Init(xmppMessageChannel)

	// Init messenger 
	myMessenger := messenger.Messenger{}
	myMessenger.Init(xmppMessageChannel, giphyClient)

	// Start a new Goroutine which listens for new messages and sents them over the mqttMessageChannel
	go mqttclient.RunMQTTListener(mqttMessageChannel)

	// Start another Goroutine which sends XMPP messages
	go xmppclient.RunXMPPClient(xmppMessageChannel)

	

	// Watch the channel and receive new messages
	for mqttMessage := range mqttMessageChannel {
		//fmt.Printf("Received message: %s from topic: %s\n", mqttMessage.Payload(), mqttMessage.Topic())
		mqttDecodedPayload := mqttclient.ParseMqttMessage(mqttMessage)
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

		// Send message via messenger
		myMessenger.ResolveLevelToMessage(normalizedMoistureValue, levelDirection, currentLevel)

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
