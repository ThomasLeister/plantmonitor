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
 * New matchLevels() function
 * Finds levels that match the current sensor value.
 * Either with overlap of levels (for hysteresis, "blurry mode") or without overlaps ("sharp mode")
 * Level of overlap is defined by hysteresisMargin.
 * hysteresisMargin = 0 	=> No overlap
 * hysteresisMargin = 2 	=> Overlap by 2 in each direction
 *
 * Returns zero, one or more levels that match the value.
 */

func (q *Quantifier) matchLevels(value int, hysteresisMargin int) []QuantificationLevel {
	var matchedLevels []QuantificationLevel

	// Loop through all levels and find matching ones
	for _, quantificationLevel := range q.QuantificationLevels {
		if (value >= quantificationLevel.Start-hysteresisMargin) && (value <= quantificationLevel.End+hysteresisMargin) {
			matchedLevels = append(matchedLevels, quantificationLevel)
		}
	}

	return matchedLevels
}

/*
 * New Quantify implementation
 */

func (q *Quantifier) QuantifyNew(value int, hysteresisMargin int) (QuantificationLevel, error) {
	var previousLevelMatched = false

	// First step: matchLevels using "blurry mode" (margin != 0) and check whether we're in an ambivalent range (two or more levels match)
	matchedLevels := q.matchLevels(value, hysteresisMargin)

	switch len(matchedLevels) {
	case 0:
		// No level could be matched. Value out or range
		return QuantificationLevel{}, fmt.Errorf("cannot assign quantification level. Value out of range")
	case 1:
		// Clear situation: This value can only be on this single level
		return matchedLevels[0], nil
	default:
		// We need further investigation. Multiple levels match the value.

		// Check if previous level is amongst matched levels
		for _, matchedLevel := range matchedLevels {
			if q.History.QuantificationLevel.Name == matchedLevel.Name {
				previousLevelMatched = true
				break
			}
		}

		if previousLevelMatched {
			// If previous level is amongst matching levels, just keep that level (=> Hysteresis applied)
			return q.History.QuantificationLevel, nil
		} else {
			// If previous level is _not_ amongst matching levels, we have no choise and need to apply "sharp" matching. (=> No hysteresis applied)
			matchedLevels = q.matchLevels(value, 0)

			// We expect only one level to be returned
			if len(matchedLevels) == 0 {
				// no level could be identified
				return QuantificationLevel{}, fmt.Errorf("cannot assign quantification level. Value out of range")
			} else if len(matchedLevels) > 1 {
				// Multiple levels could be identified. Overlap detected without blurry match! Something must be wrong in config.
				return QuantificationLevel{}, fmt.Errorf("multiple levels are matching. Overlap in level configuration?")
			} else {
				// A single suitable level was found. Return it.
				return matchedLevels[0], nil
			}
		}
	}
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

	// Calc hysteresis margin for hysteresis / "blurry" matching of levels
	hysteresisMargin := (q.Sensor.Normalized.NoiseMargin / 2)
	currentLevel, err := q.QuantifyNew(moistureValue, hysteresisMargin)
	if err != nil {
		return levelDirection, QuantificationLevel{}, fmt.Errorf("could not evaluate new moisture value: %s", err)
	}

	// Save current value and level
	q.Current = QuantificationResult{Value: moistureValue, QuantificationLevel: currentLevel}
	log.Printf("Quantification result: moistureValue=%d QuantificationLevel=%s", q.Current.Value, q.Current.QuantificationLevel.Name)

	// Check history: Has was level increased of decreased?
	if q.HistoryExists() {
		// We have history: Check if level has changed compared to previous
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
