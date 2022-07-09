package main

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	configManagerPkg "thomas-leister.de/plantmonitor/configmanager"
	gifManagerPkg "thomas-leister.de/plantmonitor/gifmanager"
	messengerPkg "thomas-leister.de/plantmonitor/messenger"
	mqttManagerPkg "thomas-leister.de/plantmonitor/mqttmanager"
	quantifierPkg "thomas-leister.de/plantmonitor/quantifier"
	reminderPkg "thomas-leister.de/plantmonitor/reminder"
	xmppManagerPkg "thomas-leister.de/plantmonitor/xmppmanager"
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

	log.Println(("Starting Plantmonitor ..."))

	// Read config
	config, err = configManagerPkg.ReadConfig("config.yaml")
	if err != nil {
		log.Fatal("Could not parse config:", err)
	} else {
		log.Println("Config was read and parsed!")
	}

	// Init xmppmanager
	xmppclient := xmppManagerPkg.XmppClient{}
	xmppclient.Init(&config)

	// Init mqttmanager
	mqttclient := mqttManagerPkg.MqttClient{}
	mqttclient.Init(&config)

	// Init Giphy
	giphyclient := gifManagerPkg.GiphyClient{}
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

	// Start another Goroutine which sends XMPP messages when receiving new XmppTextMessage or XmppGifMessage strings
	go xmppclient.RunXMPPClient(xmppMessageChannel)

	/*
	 * Watch the MQTT channel and receive new messages
	 */
	for mqttMessage := range mqttMessageChannel {
		mqttDecodedPayload := mqttclient.ParseMqttMessage(mqttMessage)
		moistureRaw := mqttDecodedPayload.UplinkMessage.DecodedPayload.MoistureRaw

		// Normalize raw value to percentage (and invert value)
		normalizedMoistureValue := normalizeRawValue(int(moistureRaw))

		log.Printf("Raw value: %d  |  Normalized value: %d %% \n", moistureRaw, normalizedMoistureValue)

		// Put normalized value into quantifier evaluation
		levelDirection, currentLevel, err := quantifier.EvaluateValue(normalizedMoistureValue)
		if err != nil {
			log.Panic("Error happended during evaluation.")
			break
		}

		// Send message via messenger
		messenger.ResolveLevelToMessage(normalizedMoistureValue, levelDirection, currentLevel)

		// Check for urgency and set reminder accordingly
		if currentLevel.Urgency != quantifierPkg.UrgencyLow {
			reminder.Set(currentLevel)
		} else {
			reminder.Stop() // Do nothing. One message is enough. Stop existing reminders.
		}
	}

	log.Fatal("Plant monitor failed. Exiting ...")
}
