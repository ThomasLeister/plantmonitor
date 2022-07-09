/*
 * Messenger package:
 * Translates moisture levels into text / GIF messages and sends them to XmppManager
 */
package messenger

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"

	"thomas-leister.de/plantmonitor/configmanager"
	"thomas-leister.de/plantmonitor/gifmanager"
	"thomas-leister.de/plantmonitor/quantifier"
	"thomas-leister.de/plantmonitor/xmppmanager"
)

type Messenger struct {
	XmppMessageChannel chan interface{}
	GiphyClient        gifmanager.GiphyClient
	Messages           *configmanager.Messages
}

/*
 * Init messenger and set
 * - xmppMessageChannel to use
 * - Giphy client to use
 */
func (m *Messenger) Init(config *configmanager.Config, xmppMessageChannel chan interface{}, giphyClient gifmanager.GiphyClient) {
	m.Messages = &config.Messages
	m.XmppMessageChannel = xmppMessageChannel
	m.GiphyClient = giphyClient
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
	m.XmppMessageChannel <- xmppmanager.XmppTextMessage(textMessage + " \nBodenfeuchte: " + strconv.Itoa(normalizedMoistureValue) + " %")

	// Send GIF (if set in config)
	if gifUrl != "" {
		m.XmppMessageChannel <- xmppmanager.XmppGifMessage(gifUrl)
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
	m.XmppMessageChannel <- xmppmanager.XmppTextMessage(textMessage + " \nBodenfeuchte: " + strconv.Itoa(normalizedMoistureValue) + " %")

	// Send GIF (if set in config)
	if gifUrl != "" {
		m.XmppMessageChannel <- xmppmanager.XmppGifMessage(gifUrl)
	}

	return nil
}
