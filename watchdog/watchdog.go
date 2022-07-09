/*
 * Watchdog: Observes sensor data and notifies users if
 * no new sensor data has been received for a certain time.
 */

package watchdog

import (
	"log"
	"time"

	"thomas-leister.de/plantmonitor/configmanager"
	"thomas-leister.de/plantmonitor/messenger"
)

type Watchdog struct {
	Messenger    *messenger.Messenger
	Timer        *time.Timer
	TimerRunning bool
	Timeout      time.Duration
}

func (w *Watchdog) Init(config *configmanager.Config, messenger *messenger.Messenger) {
	log.Println("Initializing watchdog ...")

	w.Timeout = time.Duration(config.Watchdog.Timeout) * time.Second
	w.Messenger = messenger
}

// Initial start of watchdog
func (w *Watchdog) Start() {
	w.Timer = time.AfterFunc(w.Timeout, func() {
		log.Println("Watchdog triggered!")
		w.Messenger.SendSensorWarning(w.Timeout)
	})
	w.TimerRunning = true
}

/*
 * Resets timer and starts the timer
 * This function should be called if a new MQTT message has arrived.
 * If no further MQTT message follows in time, the timer will trigger.
 */
func (w *Watchdog) Reset() {
	if !w.TimerRunning {
		w.Start()
	} else {
		w.Timer.Stop()
		w.Timer.Reset(w.Timeout)
	}
}
