/*
 * Messenger package:
 * Translates moisture levels into text / GIF messages and sends them to XmppManager
 */
package messenger

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"text/template"
	"time"

	"thomas-leister.de/plantmonitor/configmanager"
	"thomas-leister.de/plantmonitor/gifmanager"
	"thomas-leister.de/plantmonitor/quantifier"
	"thomas-leister.de/plantmonitor/sensor"
	"thomas-leister.de/plantmonitor/xmppmanager"
)

type Messenger struct {
	XmppMessageOutChannel chan interface{}
	XmppMessageInChannel  chan xmppmanager.XmppInMessage // XMPP channel for incoming messages
	GiphyClient           gifmanager.GiphyClient
	Messages              *configmanager.Messages
	Sensor                *sensor.Sensor

	Templates struct {
		CurrentStateAnswer   *template.Template
		WarningSensorOffline *template.Template
	}
}

type CurrentStateAnswerParams struct {
	SensorValue int
	LastUpdated time.Time
}

type WarningSensorOfflineParams struct {
	Timeout time.Duration
}

func (m *Messenger) ResponderLoop() {
	for xmppMessage := range m.XmppMessageInChannel {
		log.Printf("Retrieved a message from %s!", xmppMessage.From)

		// Set recipients (= sender of this message)
		recipients := []string{xmppMessage.From}

		// Cimplify body message to be able to understand intention
		simpleBodyString := strings.TrimSpace(strings.ToLower(xmppMessage.Body))

		if simpleBodyString != "" {
			if simpleBodyString == "help" {
				log.Println("Sending help menu")
				m.XmppMessageOutChannel <- xmppmanager.XmppTextMessage{Recipients: recipients, Text: m.Messages.Answers.AvailableCommands}
			} else if simpleBodyString == "wie geht's dir?" {
				// If we have valid data, send them
				log.Println("Sending health info")
				if m.Sensor.Normalized.History.Valid {
					var messageStringBuffer bytes.Buffer

					answerParams := CurrentStateAnswerParams{
						SensorValue: m.Sensor.Normalized.Current.Value,
						LastUpdated: m.Sensor.LastUpdated,
					}

					err := m.Templates.CurrentStateAnswer.Execute(&messageStringBuffer, answerParams)
					if err != nil {
						panic(err)
					}

					// Respond via out channel
					m.XmppMessageOutChannel <- xmppmanager.XmppTextMessage{
						Recipients: recipients,
						Text:       messageStringBuffer.String(),
					}
				} else {
					m.XmppMessageOutChannel <- xmppmanager.XmppTextMessage{Recipients: recipients, Text: m.Messages.Answers.SensorDataUnavailable}
				}
			} else {
				log.Println("Sending help info")
				m.XmppMessageOutChannel <- xmppmanager.XmppTextMessage{Recipients: recipients, Text: m.Messages.Answers.UnknownCommand}
			}
		} else {
			log.Println("[Dropped message because it does not contain body]")
		}
	}
}

/*
 * Init messenger and set
 * - xmppMessageChannel to use
 * - Giphy client to use
 */
func (m *Messenger) Init(config *configmanager.Config, xmppMessageOutChannel chan interface{}, xmppMessageInChannel chan xmppmanager.XmppInMessage, giphyClient gifmanager.GiphyClient, sensor *sensor.Sensor) error {
	var err error

	m.Messages = &config.Messages
	m.XmppMessageOutChannel = xmppMessageOutChannel
	m.XmppMessageInChannel = xmppMessageInChannel
	m.GiphyClient = giphyClient
	m.Sensor = sensor

	// Load message strings and parse templates
	m.Templates.CurrentStateAnswer, err = template.New("").Parse(config.Messages.Answers.CurrentState)
	if err != nil {
		return fmt.Errorf("failed to parse template for messages.answers.current_state: %s", err)
	}

	m.Templates.WarningSensorOffline, err = template.New("").Parse(config.Messages.Warnings.SensorOffline)
	if err != nil {
		return fmt.Errorf("failed to parse template for messages.warnings.sensor_offline: %s", err)
	}

	return nil
}

/*
 * Input:
 * 	- A level name
 *  - Level direction (+1, 0 , -1)
 *  - Whether this is a reminder (bool)
 */
func (m *Messenger) GetMessage(levelName string, levelDirection int, reminder bool) (string, string, error) {
	var levelDirectionString string = "steady"
	var gifUrl string = ""
	var err error

	if !reminder {
		if levelDirection == 1 {
			levelDirectionString = "up"
		} else if levelDirection == 0 {
			levelDirectionString = "steady"
		} else if levelDirection == -1 {
			levelDirectionString = "down"
		}
	} else {
		levelDirectionString = "reminder"
	}

	// Build message type identifier, e.g. normal_steady, normal_up, high_reminder, ... (just as in YAML config)
	messageTypeString := levelName + "_" + levelDirectionString

	// Get messages array
	messageType := m.Messages.Levels[messageTypeString]

	// Choose one random message from the messages array
	messages := messageType.Messages
	messagesNum := len(messages)
	randomMessage := messages[rand.Intn(messagesNum)]

	// Choose a GIF
	gifKeywords := messageType.GifKeywords
	if gifKeywords != "" {
		gifUrl, err = m.GiphyClient.GetGifURL(gifKeywords)
		if err != nil {
			fmt.Errorf("Could not retrieve GIF URL from gifmanager: %s", err)
		}
	}

	return randomMessage, gifUrl, nil
}

/*
 * Inputs:
 * - Direction of levels (up, stead, down +1, 0, -1)
 * - Current level
 * - Xmpp client instance to use for sending
 */
func (m *Messenger) ResolveLevelToMessage(normalizedMoistureValue int, levelDirection int, currentLevel quantifier.QuantificationLevel) error {
	log.Println("Resolving level and direction to message...")

	// Send a text message and GIF (if any GIF keywords are defined)
	textMessage, gifUrl, err := m.GetMessage(currentLevel.Name, levelDirection, false)
	if err != nil {
		fmt.Errorf("Could not get and suitable message from config for level %s and direction %d: %s", currentLevel.Name, levelDirection, err)
	}
	log.Printf("Sending message: \"%s\" \n", textMessage)

	// Send text message
	m.XmppMessageOutChannel <- xmppmanager.XmppTextMessage{Text: textMessage + " \nBodenfeuchte: " + strconv.Itoa(normalizedMoistureValue) + " %"}

	// Send GIF (if set in config)
	if gifUrl != "" {
		m.XmppMessageOutChannel <- xmppmanager.XmppGifMessage{Url: gifUrl}
	}

	return nil
}

/*
 * Inputs:
 * - Level to remind of
 * - Current Moisture level
 */
func (m *Messenger) SendReminder(currentLevel quantifier.QuantificationLevel, normalizedMoistureValue int) error {
	log.Println("Resolving level and direction to message...")

	// Send a text message and GIF (if any GIF keywords are defined)
	textMessage, gifUrl, err := m.GetMessage(currentLevel.Name, 0, true)
	if err != nil {
		fmt.Errorf("Could not get and suitable reminder message from config for level %s: %s", currentLevel.Name, err)
	}
	log.Printf("Sending message: \"%s\" \n", textMessage)

	// Send text message
	m.XmppMessageOutChannel <- xmppmanager.XmppTextMessage{Text: textMessage + " \nBodenfeuchte: " + strconv.Itoa(normalizedMoistureValue) + " %"}

	// Send GIF (if set in config)
	if gifUrl != "" {
		m.XmppMessageOutChannel <- xmppmanager.XmppGifMessage{Url: gifUrl}
	}

	return nil
}

func (m *Messenger) SendSensorWarning(interval time.Duration) {
	var messageStringBuffer bytes.Buffer
	log.Println("Sending sensor availability warning")

	warningParams := WarningSensorOfflineParams{
		Timeout: interval,
	}

	err := m.Templates.WarningSensorOffline.Execute(&messageStringBuffer, warningParams)
	if err != nil {
		panic(err)
	}

	// Respond via out channel
	m.XmppMessageOutChannel <- xmppmanager.XmppTextMessage{
		Text: messageStringBuffer.String(),
	}
}
