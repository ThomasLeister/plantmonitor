package reminder

import (
	"fmt"
	"time"

	"thomas-leister.de/plantmonitor/quantifier"
)

type Reminder struct {
	quitChannel        chan bool    // Control channel to end reminder loop
	xmppMessageChannel chan string  // Channel to xmpp service
	ticker             *time.Ticker // Ticker for notification loop
	tickerRunning      bool
}

func (r *Reminder) reminderNotificationLoop(quitChannel chan bool, ticker *time.Ticker, level quantifier.QuantificationLevel) {
	r.tickerRunning = true

	for {
		select {
		case <-quitChannel:
			ticker.Stop()
			r.tickerRunning = false
			fmt.Println("Timer: Stopped.")
			return
		case t := <-ticker.C:
			fmt.Println("Remembering ...", t)
			r.xmppMessageChannel <- level.ChatMessageReminder
		}
	}
}

func (r *Reminder) Init(xmppMessageChannel chan string) {
	r.xmppMessageChannel = xmppMessageChannel

	// Init quit channel
	r.quitChannel = make(chan bool)
}

func (r *Reminder) Set(currentLevel quantifier.QuantificationLevel) {
	fmt.Println("Setting timer")
	r.Stop()
	r.ticker = time.NewTicker(currentLevel.NotificationInterval)
	go r.reminderNotificationLoop(r.quitChannel, r.ticker, currentLevel)
}

func (r *Reminder) Stop() {
	if r.tickerRunning {
		fmt.Println("Stopping timer...")
		r.quitChannel <- true
	}
}
