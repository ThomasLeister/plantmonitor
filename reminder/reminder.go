/*
 * Reminder:
 * Implements goroutines with timer for reminding of cricital moisture levels
 */

package reminder

import (
	"fmt"
	"time"
	"log"

	"thomas-leister.de/plantmonitor/quantifier"
	. "thomas-leister.de/plantmonitor/xmppmanager"
)

type Reminder struct {
	quitChannel        chan bool        // Control channel to end reminder loop
	xmppMessageChannel chan interface{} // Channel to xmpp service
	ticker             *time.Ticker     // Ticker for notification loop
	tickerRunning      bool
}

func (r *Reminder) reminderNotificationLoop(quitChannel chan bool, ticker *time.Ticker, level quantifier.QuantificationLevel) {
	r.tickerRunning = true

	for {
		select {
		case <-quitChannel:
			ticker.Stop()
			r.tickerRunning = false
			log.Println("Timer: Stopped.")
			return
		case t := <-ticker.C:
			fmt.Println("Timer: Remembering user ...", t)
			r.xmppMessageChannel <- XmppTextMessage(level.ChatMessageReminder)
		}
	}
}

func (r *Reminder) Init(xmppMessageChannel chan interface{}) {
	r.xmppMessageChannel = xmppMessageChannel

	// Init quit channel
	r.quitChannel = make(chan bool)
}

func (r *Reminder) Set(currentLevel quantifier.QuantificationLevel) {
	log.Println("Timer: Setting timer")
	r.Stop()
	r.ticker = time.NewTicker(currentLevel.NotificationInterval)
	go r.reminderNotificationLoop(r.quitChannel, r.ticker, currentLevel)
}

func (r *Reminder) Stop() {
	if r.tickerRunning {
		log.Println("Timer: Stopping timer...")
		r.quitChannel <- true
	}
}
