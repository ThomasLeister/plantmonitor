package quantifier

import (
	"fmt"
	"strconv"
)

type QuantificationLevel struct {
	Start           int    // Quantification Level Start value (0 < value < 100)
	End             int    // ""
	Name            string // Level name, such as "low", "normal", "high"
	ChatMessageUp   string // Message to send if this level ist reached from a lower level
	ChatMessageDown string // Message to send if this level is reached from a higher level
}

type Quantifier struct {
	CurrentLevel         QuantificationLevel   // current / old level
	QuantificationLevels []QuantificationLevel // All available quantification levels.
}

func (q Quantifier) Init() {
	fmt.Println("Hello, I'm Quantifier!")

	q.CurrentLevel = QuantificationLevel{31, 66, "normal", "I'm feeling fine", "I'm back to normal"}

	//q.QuantificationLevels = make([]QuantificationLevel, 10)

	// Read all quantification levels from config and copy them into q.QuantificationLevels
	q.QuantificationLevels = []QuantificationLevel{
		QuantificationLevel{0, 30, "low", "", "Need more water! Pleeeeease!"},
		QuantificationLevel{31, 66, "normal", "I'm feeling fine", "I'm back to normal"},
		QuantificationLevel{67, 100, "hight", "I've got much water right now! Thanks!", ""},
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
func (q Quantifier) Quantify(value int) (QuantificationLevel, error) {
	for _, quantificationLevel := range q.QuantificationLevels {
		if (value >= quantificationLevel.Start) && (value <= quantificationLevel.End) {
			//This is the correct quantification level.
			return quantificationLevel, nil
		}
	}

	return QuantificationLevel{}, fmt.Errorf("cannot asign quantification level. Value out of range.\n")
}
