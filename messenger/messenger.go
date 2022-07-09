/*
 * Messenger package:
 * Translates moisture levels into text / GIF messages and sends them to XmppManager
 */
package messenger

import (
	"fmt"
	"thomas-leister.de/plantmonitor/gifmanager"
	"thomas-leister.de/plantmonitor/quantifier"
	"thomas-leister.de/plantmonitor/xmppmanager"
)

type Messenger struct {
	HistoryExists      bool
	XmppMessageChannel chan interface{}
	GiphyClient        gifmanager.GiphyClient
}

/*
 * Init messenger and set
 * - xmppMessageChannel to use
 * - Giphy client to use
 */
func (m *Messenger) Init(xmppMessageChannel chan interface{}, giphyClient gifmanager.GiphyClient) {
	m.XmppMessageChannel = xmppMessageChannel
	m.GiphyClient = giphyClient
}

/*
 * Inputs:
 * - Direction of levels (up, stead, down +1, 0, -1)
 * - Current level
 * - Xmpp client instance to use for sending
 */
func (m *Messenger) ResolveLevelToMessage(normalizedMoistureValue int, levelDirection int, currentLevel quantifier.QuantificationLevel) error {
	fmt.Println("Resolving level and direction to message...")

	if m.HistoryExists {
		// If value has changed and we have history, output a message
		if levelDirection == -1 {
			// Send normalized value and level name / level message
			fmt.Printf("Sending message: %s \n", currentLevel.ChatMessageDown)
			m.XmppMessageChannel <- xmppmanager.XmppTextMessage(fmt.Sprintf("%s \n\nBodenfeuchte: %d %%", currentLevel.ChatMessageDown, normalizedMoistureValue))
		} else if levelDirection == +1 {
			// Send normalized value and level name / level message
			fmt.Printf("Sending message: %s \n", currentLevel.ChatMessageUp)
			m.XmppMessageChannel <- xmppmanager.XmppTextMessage(fmt.Sprintf("%s \n\nBodenfeuchte: %d %%", currentLevel.ChatMessageUp, normalizedMoistureValue))
		} else if levelDirection == 0 {
			// Level has not changed (or there has been no history)
			fmt.Println("Level has not changed. Not sending any message (except for reminders).")
		}
	} else {
		// No history exists, yet (e.g. due to power-on). Just tell about the current state.
		fmt.Printf("Sending message: %s \n", currentLevel.ChatMessageSteady)

		// Send message and GIF: "I'm back"
		m.XmppMessageChannel <- xmppmanager.XmppTextMessage("Ich bin wieder online!")
		gifUrl, err := m.GiphyClient.GetGifURL("I'm back")
		if err != nil {
			fmt.Println("Failed to retrieve GIF:", err)
		} else {
			m.XmppMessageChannel <- xmppmanager.XmppGifMessage(gifUrl)
		}

		// Send sensor data ("steady" message)
		m.XmppMessageChannel <- xmppmanager.XmppTextMessage(fmt.Sprintf("%s \n\nBodenfeuchte: %d %%", currentLevel.ChatMessageSteady, normalizedMoistureValue))

		m.HistoryExists = true
	}

	return nil
}
