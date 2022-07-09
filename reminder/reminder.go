/*
 * Reminder:
 * Implements goroutines with timer for reminding of cricital moisture levels
 */

package reminder

import (
	"fmt"
	"log"
	"sync"
	"time"

	"thomas-leister.de/plantmonitor/messenger"
	"thomas-leister.de/plantmonitor/quantifier"
	"thomas-leister.de/plantmonitor/sensor"
)

type Reminder struct {
	quitChannel   chan bool // Control channel to end reminder loop
	tickerRunning bool
	Sensor        *sensor.Sensor       // Sensor for retrieving the current moisture value
	Messenger     *messenger.Messenger // Messenger for sending reminder messages
	wg            sync.WaitGroup
}

/*
 * Reminder Notification Loop:
 * Is running as a Goroutine if a ticker / reminder is active.
 * Is _not_ running if no reminder is running.
 * Goroutine / ticker can be quit by putting "true" into quitChannel
 */
func (r *Reminder) reminderNotificationLoop(quitChannel chan bool, notificationInterval time.Duration, level quantifier.QuantificationLevel) {
	log.Println("Reminder: Started reminder loop")

	// Set ticker
	ticker := time.NewTicker(notificationInterval)

	// Send a done signal to waitgroup if this loop has quit
	defer r.wg.Done()

	for {
		select {
		case <-quitChannel:
			ticker.Stop()
			log.Println("Reminder: Ticker stopped. Quitting goroutine ...")
			return
		case t := <-ticker.C:
			fmt.Println("Reminder: Remembering user ...", t)
			r.Messenger.SendReminder(level, r.Sensor.Normalized.Current.Value)
		}
	}
}

func (r *Reminder) Init(messenger *messenger.Messenger, sensor *sensor.Sensor) {
	log.Println("Reminder: Initializing reminder ...")

	r.Messenger = messenger
	r.Sensor = sensor

	// Init quit channel
	r.quitChannel = make(chan bool)
}

/*
 * Stop any running reminder
 * and launch a new reminder goroutine
 */
func (r *Reminder) Set(currentLevel quantifier.QuantificationLevel) {
	r.Stop()

	// Create a new reminder loop
	log.Println("Reminder: Creating a new reminder goroutine")
	r.wg.Add(1)
	go r.reminderNotificationLoop(r.quitChannel, currentLevel.NotificationInterval, currentLevel)
	r.tickerRunning = true
}

/*
 * Just stop the reminder Goroutine
 * and don't start a new one.
 */
func (r *Reminder) Stop() {
	if r.tickerRunning {
		log.Println("Reminder: Stopping current reminder goroutine")
		r.quitChannel <- true

		// Wait until goroutine has quit
		r.wg.Wait()
		r.tickerRunning = false
		log.Println("Reminder: Reminder goroutine was quit")
	}
}
