/*
 * SPDX-License-Identifier: MIT
 * (valid for all sub-packages in original repository)
 */

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	configManagerPkg "thomas-leister.de/plantmonitor/configmanager"
	gifManagerPkg "thomas-leister.de/plantmonitor/gifmanager"
	messengerPkg "thomas-leister.de/plantmonitor/messenger"
	mqttManagerPkg "thomas-leister.de/plantmonitor/mqttmanager"
	quantifierPkg "thomas-leister.de/plantmonitor/quantifier"
	reminderPkg "thomas-leister.de/plantmonitor/reminder"
	sensorPkg "thomas-leister.de/plantmonitor/sensor"
	watchdogPkg "thomas-leister.de/plantmonitor/watchdog"
	xmppManagerPkg "thomas-leister.de/plantmonitor/xmppmanager"
)

/* Version string. Is manipulated by build script.*/
var versionString string = "0.0.0"

/* Global var for config*/
var config configManagerPkg.Config

func main() {
	var err error
	var quantifierHistoryExists = false

	mqttMessageChannel := make(chan mqttManagerPkg.MqttDecodedPayload)
	xmppMessageOutChannel := make(chan interface{})
	xmppMessageInChannel := make(chan xmppManagerPkg.XmppInMessage)

	// Welcome message and version
	log.Printf("Starting Plantmonitor %s ...", versionString)

	// Read config
	config, err = configManagerPkg.ReadConfig("config.yaml")
	if err != nil {
		log.Fatal("Could not parse config:", err)
	} else {
		log.Println("Config was read and parsed!")
	}

	// Init sensor
	sensor := sensorPkg.Sensor{}
	sensor.Init(&config)

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
	quantifier.Init(&config, &sensor)

	// Init messenger
	messenger := messengerPkg.Messenger{}
	err = messenger.Init(&config, xmppMessageOutChannel, xmppMessageInChannel, giphyclient, &sensor)
	if err != nil {
		log.Fatal("Could not initialize messenger:", err)
	}

	// Init reminder engine
	reminder := reminderPkg.Reminder{}
	reminder.Init(&messenger, &sensor)

	// Init watchdog
	watchdog := watchdogPkg.Watchdog{}
	watchdog.Init(&config, &messenger)

	/*
	 * Start signal handler routine
	 */
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP) // React to SIGHUP signals (daemon "reload")
	go func() {
		for range signalChan {
			fmt.Println("Got a HUP signal! Reloading configuration ...")

			// Read config
			config, err = configManagerPkg.ReadConfig("config.yaml")
			if err != nil {
				log.Fatal("Could not parse config:", err)
			} else {
				log.Println("Config was read and parsed!")

				// Reload parts of other services
				quantifier.Reload(&config)
				messenger.Reload(&config)
			}
		}
	}()

	// Start a new Goroutine which listens for new messages and sents them over the mqttMessageChannel
	go mqttclient.RunMQTTListener(mqttMessageChannel)

	// Start another Goroutine which sends XMPP messages when receiving new XmppTextMessage or XmppGifMessage strings
	go xmppclient.RunXMPPClient(xmppMessageOutChannel, xmppMessageInChannel)

	// Start Messenger responder: Responds to incoming XMPP messages
	go messenger.ResponderLoop()

	/*
	 * Watch the MQTT channel and receive new messages
	 */
	for mqttMessage := range mqttMessageChannel {
		log.Println("Received new sensor value via MQTT!")

		// Satisfy watchdog
		watchdog.Reset()

		// Decode MQTT message
		/*mqttDecodedPayload := mqttclient.ParseMqttMessage(mqttMessage)
		moistureRaw := mqttDecodedPayload.UplinkMessage.DecodedPayload.MoistureRaw*/

		// Get moistureRaw from mqttMessage
		moistureRaw := mqttMessage.MoistureRaw

		// Update current sensor value
		sensor.UpdateCurrentValue(int(moistureRaw))
		log.Printf("Raw sensor value: %d  |  Current normalized and filtered value: %d %% \n", moistureRaw, sensor.Normalized.Current.Value)

		// Save state before first value is evaluated, because then history will exist for sure ;)
		quantifierHistoryExists = quantifier.HistoryExists()

		// Put current sensor value into quantifier
		levelDirection, currentLevel, err := quantifier.EvaluateValue(sensor.Normalized.Current.Value)
		if err != nil {
			log.Panicf("Error happended during evaluation: %s", err)
			continue
		}

		/*
		 * Check if level has changed. Only notify
		 *     - on level change or
		 *     - if no history exists (first sensor value was read / quantified)
		 */
		if (levelDirection != 0) || (!quantifierHistoryExists) {
			// Send message via messenger
			messenger.ResolveLevelToMessage(sensor.Normalized.Current.Value, levelDirection, currentLevel)

			// Stop all reminders for the old level
			reminder.Stop()

			// If new level demands a reminder, set it:
			if currentLevel.NotificationInterval != 0 {
				reminder.Set(currentLevel)
			}
		}
	}

	log.Fatal("Plantmonitor failed. Exiting ...")
}
