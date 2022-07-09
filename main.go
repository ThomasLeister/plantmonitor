package main

import (
	"fmt"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	configManagerPkg "thomas-leister.de/plantmonitor/configmanager"
	quantifierPkg "thomas-leister.de/plantmonitor/quantifier"
	reminderPkg "thomas-leister.de/plantmonitor/reminder"
	giphyPkg "thomas-leister.de/plantmonitor/giphy"
	xmppManagerPkg "thomas-leister.de/plantmonitor/xmppmanager"
	mqttManagerPkg "thomas-leister.de/plantmonitor/mqttmanager"
	messengerPkg "thomas-leister.de/plantmonitor/messenger"
)

/* Global var for config*/
var config configManagerPkg.Config


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
	config, err = configManagerPkg.ReadConfig("config.yaml")
	if err != nil {
		fmt.Println("Could not parse config:", err)
		os.Exit(1)
	} else {
		fmt.Println("Config was read and parsed!")
	}

	// Init xmppmanager 
	xmppclient := xmppManagerPkg.XmppClient{}
	xmppclient.Init(&config)

	// Init mqttmanager
	mqttclient := mqttManagerPkg.MqttClient{}
	mqttclient.Init(&config)

	// Init Giphy
	giphyclient := giphyPkg.GiphyClient{}
	giphyclient.Init(config.Giphy.ApiKey)

	// Init quantifier
	quantifier := quantifierPkg.Quantifier{}
	quantifier.Init(&config)

	// Init reminder engine
	reminder := reminderPkg.Reminder{}
	reminder.Init(xmppMessageChannel)

	// Init messenger 
	messenger := messengerPkg.Messenger{}
	messenger.Init(xmppMessageChannel, giphyclient)

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
		levelDirection, currentLevel, err := quantifier.EvaluateValue(normalizedMoistureValue)
		if err != nil {
			fmt.Printf("Error happended during evaluation.")
			break
		}

		// Send message via messenger
		messenger.ResolveLevelToMessage(normalizedMoistureValue, levelDirection, currentLevel)

		// Check for urgency.
		// If state demands urgent action, set a periodic reminder!
		if currentLevel.Urgency != quantifierPkg.UrgencyLow {
			reminder.Set(currentLevel)
		} else {
			// Do nothing. One message is enough. Stop existing reminders.
			reminder.Stop()
		}
	}

	fmt.Println("Plant monitor failed. Exiting ...")
}
