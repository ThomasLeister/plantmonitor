/*
 * Reminder:
 * Implements goroutines with timer for reminding of cricital moisture levels
 */

package reminder

import (
	"fmt"
	"log"
	"time"

	"thomas-leister.de/plantmonitor/messenger"
	"thomas-leister.de/plantmonitor/quantifier"
	"thomas-leister.de/plantmonitor/sensor"
)

type Reminder struct {
	quitChannel   chan bool    // Control channel to end reminder loop
	ticker        *time.Ticker // Ticker for notification loop
	tickerRunning bool
	Sensor        *sensor.Sensor       // Sensor for retrieving the current moisture value
	Messenger     *messenger.Messenger // Messenger for sending reminder messages
}

func (r *Reminder) reminderNotificationLoop(quitChannel chan bool, ticker *time.Ticker, level quantifier.QuantificationLevel) {
	r.tickerRunning = true

	for {
		select {
		case <-quitChannel:
			ticker.Stop()
			r.tickerRunning = false
			log.Println("Reminder: Timer stopped.")
			return
		case t := <-ticker.C:
			fmt.Println("Reminder: Remembering user ...", t)
			r.Messenger.SendReminder(level, r.Sensor.Normalized.Current.Value)
		}
	}
}

func (r *Reminder) Init(messenger *messenger.Messenger, sensor *sensor.Sensor) {
	log.Println("Initializing reminder ...")

	r.Messenger = messenger
	r.Sensor = sensor

	// Init quit channel
	r.quitChannel = make(chan bool)
}

func (r *Reminder) Set(currentLevel quantifier.QuantificationLevel) {
	log.Println("Reminder: Setting timer")
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
