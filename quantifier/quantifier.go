package quantifier

import (
	"fmt"
	"strconv"
	"time"

	"thomas-leister.de/plantmonitor/configmanager"
)

type Urgency int64

const (
	UrgencyLow    Urgency = iota // Low urgency = Do not remember, e.g. it level is normal.
	UrgencyMedium                // Remember sometimes
	UrgencyHigh                  // Remember in short intervals, critical state.
)

type QuantificationLevel struct {
	Start                int           // Quantification Level Start value (0 < value < 100)
	End                  int           // ""
	Name                 string        // Level name, such as "low", "normal", "high"
	ChatMessageSteady   string        // Message to send if this level is steady
	ChatMessageUp        string        // Message to send if this level is reached from a lower level
	ChatMessageDown      string        // Message to send if this level is reached from a higher level
	ChatMessageReminder  string        // Reminder chat message text
	Urgency              Urgency       // Whether to remember humans of the (bad) state, in case this is a bad state. The higher the urgency, the higher the remember interval.
	NotificationInterval time.Duration // Notification interval in seconds
}

type QuantificationHistory struct {
	Value               int                 // old normalized value
	QuantificationLevel QuantificationLevel // Old quantification level
}

type Quantifier struct {
	QuantificationHistory QuantificationHistory // old value and level for comparison / history
	QuantificationLevels  []QuantificationLevel // All available quantification levels.
}

func (q *Quantifier) Init(config *configmanager.Config) {
	fmt.Println("Initializing quantifier ...")

	// Set to empty history
	q.QuantificationHistory = QuantificationHistory{}

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

		if level.Urgency == "low" {
			newLevel.Urgency = UrgencyLow
		} else if level.Urgency == "medium" {
			newLevel.Urgency = UrgencyMedium
		} else if level.Urgency == "high" {
			newLevel.Urgency = UrgencyHigh
		}

		// Append new item to levels
		q.QuantificationLevels = append(q.QuantificationLevels, newLevel)
	}
	
	// Read values back
	fmt.Println("Available levels:")
	fmt.Println("---------------------------------------------")
	for _, level := range q.QuantificationLevels {
		fmt.Printf("|" + level.Name + " | " + strconv.Itoa(level.Start) + " | " + strconv.Itoa(level.End) + " |\n")
	}
	fmt.Println("---------------------------------------------")
}

/*
 * Quantification function
 * Takes: value (0 < value < 100)
 * Returns: QuantificationLevel
 */
func (q *Quantifier) Quantify(value int) (QuantificationLevel, error) {
	for _, quantificationLevel := range q.QuantificationLevels {
		if (value >= quantificationLevel.Start) && (value <= quantificationLevel.End) {
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

	// First: Quantify current value and check which level we reached
	currentLevel, err := q.Quantify(moistureValue)
	if err != nil {
		return levelDirection, QuantificationLevel{}, fmt.Errorf("could not evaluate new moisture Value: %s", err)
	}

	// Check history: Has value increased or decreased?
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
		fmt.Println("We do not have history, yet!")
	}

	// Save new value and level to history
	q.QuantificationHistory = QuantificationHistory{moistureValue, currentLevel}

	// Return levelDirection, QuantificationLevel and error values
	return levelDirection, currentLevel, err
}
