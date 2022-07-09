/*
 * Quantifier:
 * Takes normalized moisture values and translates them into discrete moisture levels
 * also reports direction of moisture level history. (up, steady, down)
 */

package quantifier

import (
	"fmt"
	"log"
	"time"

	"thomas-leister.de/plantmonitor/configmanager"
	"thomas-leister.de/plantmonitor/sensor"
)

type QuantificationLevel struct {
	Start                int           // Quantification Level Start value (0 < value < 100)
	End                  int           // ""
	Name                 string        // Level name, such as "low", "normal", "high"
	NotificationInterval time.Duration // Notification interval in seconds
}

type QuantificationResult struct {
	Value               int                 // old normalized sensor value
	QuantificationLevel QuantificationLevel // Old quantification level
}

type Quantifier struct {
	Current              QuantificationResult  // the current quantification result
	History              QuantificationResult  // old value and level for comparison / history
	QuantificationLevels []QuantificationLevel // All available quantification levels.
	Sensor               *sensor.Sensor        // Sensor for which to quantify (use for hysteresis)
}

func (q *Quantifier) Init(config *configmanager.Config, sensor *sensor.Sensor) {
	log.Println("Initializing quantifier ...")

	// Set to empty history
	q.History = QuantificationResult{}

	// Set sensor reference
	q.Sensor = sensor

	// Load quantification levels
	q.loadLevels(config)
}

func (q *Quantifier) loadLevels(config *configmanager.Config) {
	// Read all quantification levels from config and copy them into q.QuantificationLevels
	q.QuantificationLevels = make([]QuantificationLevel, 0)

	for _, level := range config.Levels {
		// Map values from config to QuantificationLevel attributes. Most attributes match 1:1, but some need extra care.
		newLevel := QuantificationLevel{}
		newLevel.Start = level.Start
		newLevel.End = level.End
		newLevel.Name = level.Name
		newLevel.NotificationInterval = time.Duration(level.NotificationInterval) * time.Second

		// Append new item to levels
		q.QuantificationLevels = append(q.QuantificationLevels, newLevel)
	}

	// Output table showing quantification levels and thresholds
	fmt.Printf("\nAvailable quantification levels:\n\n")
	printLevelTable(&q.QuantificationLevels)
	fmt.Printf("\n")
}

func (q *Quantifier) Reload(config *configmanager.Config) {
	// Reload levels
	log.Println("Quantifier: Reloading quantification levels")
	q.loadLevels(config)
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
	log.Printf("Sensor direction: %d | Hysteresis threshold shift: %d \n", q.Sensor.Normalized.Current.Direction, hysteresisMargin)

	// Quantify current value and check which level we reached
	currentLevel, err := q.Quantify(moistureValue, hysteresisMargin)
	if err != nil {
		return levelDirection, QuantificationLevel{}, fmt.Errorf("could not evaluate new moisture Value: %s", err)
	}

	// Save current value and level
	q.Current = QuantificationResult{Value: moistureValue, QuantificationLevel: currentLevel}
	log.Printf("Quantification result: moistureValue=%d QuantificationLevel=%s", q.Current.Value, q.Current.QuantificationLevel.Name)

	// Check history: Has was level increased of decreased?
	if q.HistoryExists() { // If history is not empty
		// We have history
		// Check if level has changed compared to previous
		if q.Current.QuantificationLevel.Name != q.History.QuantificationLevel.Name {
			// Level has changed. Up or down?
			if q.Current.Value > q.History.Value {
				levelDirection = 1
			} else if q.Current.Value < q.History.Value {
				levelDirection = -1
			}
		}
	} else {
		// We do not have history, yet.
		log.Println("We do not have quantifier history, yet! Assuming levelDirection=0 (steady)")
	}

	// Save old value to history
	q.History = q.Current

	// Return levelDirection, QuantificationLevel and error values
	return levelDirection, currentLevel, err
}

func (q *Quantifier) HistoryExists() bool {
	if (q.History != QuantificationResult{}) {
		return true
	} else {
		return false
	}
}
