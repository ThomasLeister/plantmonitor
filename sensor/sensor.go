/*
 * package calibration offers functions to retrieve
 * calibrated / normalized values from any sensor raw value
 */

package sensor

import (
	"log"

	"thomas-leister.de/plantmonitor/configmanager"
)

type Sensor struct {
	Adc struct {
		RawLowerBound  int
		RawUpperBound  int
		RawNoiseMargin int
	}
	Normalized struct {
		Current struct {
			Value     int
			Direction int // Direction after UpdateCurrentValue() ...  up: +1 | steady: 0 | down: -1
		}
		History struct {
			Valid     bool // Whether History exists / is valid: If there is no history,
			LastValue int  // Last known sensor value (normalized)
		}
		NoiseMargin int
	}
}

func (s *Sensor) Init(config *configmanager.Config) {
	s.Adc.RawLowerBound = config.Sensor.Adc.RawLowerBound
	s.Adc.RawUpperBound = config.Sensor.Adc.RawUpperBound
	s.Adc.RawNoiseMargin = config.Sensor.Adc.RawNoiseMargin

	// Normalize noise margin.
	s.Normalized.NoiseMargin = int(float32(config.Sensor.Adc.RawNoiseMargin) * (100 / (float32(s.Adc.RawUpperBound) - float32(s.Adc.RawLowerBound))))
	log.Printf("Sensor noise margin is %d %%", s.Normalized.NoiseMargin)
}

func (s *Sensor) UpdateCurrentValue(currentRaw int) {
	// Back up old value to history
	s.Normalized.History.LastValue = s.Normalized.Current.Value

	// Update new current value
	s.Normalized.Current.Value = s.NormalizeRawValue(currentRaw)

	// Based on that: Update direction
	if s.Normalized.History.Valid {
		// If there has been sensor history: Calc value direction
		if s.Normalized.Current.Value > s.Normalized.History.LastValue {
			s.Normalized.Current.Direction = +1
		} else if s.Normalized.Current.Value < s.Normalized.History.LastValue {
			s.Normalized.Current.Direction = -1
		} else if s.Normalized.Current.Value == s.Normalized.History.LastValue {
			s.Normalized.Current.Direction = 0
		}

	} else {
		log.Println("There has not been any sensor history. Assuming 'steady' as direction")
		s.Normalized.Current.Direction = 0
	}

	// History is valid after 1st UpdateCurrentValue() run
	s.Normalized.History.Valid = true
}

func (s *Sensor) NormalizeRawValue(rawValue int) int {
	// Normalize range
	rangeNormalizedValue := rawValue - s.Adc.RawLowerBound

	// Normalize to percentage
	percentageValue := float32(rangeNormalizedValue) * (100 / (float32(s.Adc.RawUpperBound) - float32(s.Adc.RawLowerBound)))

	// Safety first: We canot accept values < 0 or > 100.
	if percentageValue < 0 {
		percentageValue = 0
	} else if percentageValue > 100 {
		percentageValue = 100
	}

	// Normalize meaning: Moisture rawValue is in fact "dryness" level: High => More dry. Low => more wet.
	// Let's invert that!
	percentageValueWetness := 100 - percentageValue

	// Return wetness percentage
	return int(percentageValueWetness)
}
