/*
 * Quantifier:
 * Takes normalized moisture values and translates them into discrete moisture levels
 * also reports direction of moisture level history. (up, steady, down)
 */

package quantifier

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"thomas-leister.de/plantmonitor/configmanager"
	"thomas-leister.de/plantmonitor/sensor"
)

type QuantificationLevel struct {
	Start                int           // Quantification Level Start value (0 < value < 100)
	End                  int           // ""
	Name                 string        // Level name, such as "low", "normal", "high"
	ChatMessageSteady    string        // Message to send if this level is steady
	ChatMessageUp        string        // Message to send if this level is reached from a lower level
	ChatMessageDown      string        // Message to send if this level is reached from a higher level
	ChatMessageReminder  string        // Reminder chat message text
	NotificationInterval time.Duration // Notification interval in seconds
}

type QuantificationHistory struct {
	Value               int                 // old normalized value
	QuantificationLevel QuantificationLevel // Old quantification level
}

type Quantifier struct {
	QuantificationHistory QuantificationHistory // old value and level for comparison / history
	QuantificationLevels  []QuantificationLevel // All available quantification levels.
	Sensor                *sensor.Sensor        // Sensor for which to quantify (use for hysteresis)
}

func (q *Quantifier) Init(config *configmanager.Config, sensor *sensor.Sensor) {
	log.Println("Initializing quantifier ...")

	// Set to empty history
	q.QuantificationHistory = QuantificationHistory{}

	// Set sensor reference
	q.Sensor = sensor

	// Read all quantification levels from config and copy them into q.QuantificationLevels
	q.QuantificationLevels = make([]QuantificationLevel, 0)

	for _, level := range config.Levels {
		// Map values from config to QuantificationLevel attributes. Most attributes match 1:1, but some need extra care.
		newLevel := QuantificationLevel{}
		newLevel.Start = level.Start
		newLevel.End = level.End
		newLevel.Name = level.Name
		newLevel.ChatMessageSteady = level.ChatMessageSteady
		newLevel.ChatMessageUp = level.ChatMessageUp
		newLevel.ChatMessageDown = level.ChatMessageDown
		newLevel.ChatMessageReminder = level.ChatMessageReminder
		newLevel.NotificationInterval = time.Duration(level.NotificationInterval) * time.Second

		// Append new item to levels
		q.QuantificationLevels = append(q.QuantificationLevels, newLevel)
	}

	// Read values back
	fmt.Println("Available quantification levels:")
	fmt.Println("---------------------------------------------")
	for _, level := range q.QuantificationLevels {
		fmt.Printf("|" + level.Name + " | " + strconv.Itoa(level.Start) + " | " + strconv.Itoa(level.End) + " |\n")
	}
	fmt.Println("---------------------------------------------")
}

/*
 * Quantification function
 * Params:
 *   - Normalized sensor value (0 <= value <= 100)
 * 	 - Hysteresis margin: Increase / decrease level thresholds by a hysteresis margin (depending on sensor history / "direction")
 * Returns: QuantificationLevel
 */
func (q *Quantifier) Quantify(value int, hysteresisMargin int) (QuantificationLevel, error) {
	for _, quantificationLevel := range q.QuantificationLevels {
		if (value >= quantificationLevel.Start+hysteresisMargin) && (value <= quantificationLevel.End+hysteresisMargin) {
			//This is the correct quantification level.
			return quantificationLevel, nil
		}
	}

	return QuantificationLevel{}, fmt.Errorf("cannot asign quantification level - Value out of range")
}

/*
 * Evaluate new value:
 * - Quantify new value (which level does the new value correspond to?)
 * - Check history: Has the level increased or decreased since last time?
 * - Return suitable message for the current level, depending on history, e.g. ChatMessageUp or ChatMessageDown
 *
 * levelDirection: -1 = decreasing | 0 = stable | +1 = increasing
 */
func (q *Quantifier) EvaluateValue(moistureValue int) (int, QuantificationLevel, error) {
	var levelDirection int = 0
	var err error

	// Calculate hysteresis margin. Its polarity depends on the direction of normalized sensor values. It's absolute value by settings.
	hysteresisMargin := q.Sensor.Normalized.Current.Direction * (q.Sensor.Normalized.NoiseMargin / 2)
	log.Printf("Sensor direction: %d | Hysteresis margin: %d \n", q.Sensor.Normalized.Current.Direction, hysteresisMargin)

	// Quantify current value and check which level we reached
	currentLevel, err := q.Quantify(moistureValue, hysteresisMargin)
	if err != nil {
		return levelDirection, QuantificationLevel{}, fmt.Errorf("could not evaluate new moisture Value: %s", err)
	}

	// Check history: Has was level increased of decreased?
	if (q.QuantificationHistory != QuantificationHistory{}) {
		// We have history
		// Check if level has changed compared to previous
		if q.QuantificationHistory.QuantificationLevel.Name == currentLevel.Name {
			// Level has not changed
			levelDirection = 0
		} else {
			// Level has changed. Up or down? Check old value
			if moistureValue > q.QuantificationHistory.Value {
				levelDirection = 1
			} else if moistureValue < q.QuantificationHistory.Value {
				levelDirection = -1
			}
		}
	} else {
		// We do not have history, yet. Let's say there has not been a change in value... and save the current value to history
		log.Println("We do not have history, yet!")
	}

	// Save new value and level to history
	q.QuantificationHistory = QuantificationHistory{moistureValue, currentLevel}

	// Return levelDirection, QuantificationLevel and error values
	return levelDirection, currentLevel, err
}
