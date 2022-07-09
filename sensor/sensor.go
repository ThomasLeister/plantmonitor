/*
 * package calibration offers functions to retrieve
 * calibrated / normalized values from any sensor raw value
 */

package sensor

import (
	"fmt"
	"log"
	"math"
	"time"

	"thomas-leister.de/plantmonitor/configmanager"
)

type Sensor struct {
	Adc struct {
		RawLowerBound  int
		RawUpperBound  int
		RawNoiseMargin int
	}
	Normalized struct {
		MvgAvg struct {
			Average      int   // Calculated moving average. Pushed into .Current.Value and .History.LastValue
			SensorValues []int // Single values
			MaxSeriesLen int   // Max lenght of value series: How many values to store for moving average?
		}

		/* .Current and .History are both sourced from .MovingAvg */
		Current struct {
			Value     int
			Direction int // Direction after UpdateCurrentValue() ...  up: +1 | steady: 0 | down: -1
		}
		History struct {
			Valid     bool // Whether History exists / is valid: If there is no history,
			LastValue int  // Last known sensor value (normalized) for sensor direction
		}
		NoiseMargin int
	}
	LastUpdated time.Time // Time of last sensor value update
}

func (s *Sensor) Init(config *configmanager.Config) {
	log.Println("Initializing sensor ...")

	s.Adc.RawLowerBound = config.Sensor.Adc.RawLowerBound
	s.Adc.RawUpperBound = config.Sensor.Adc.RawUpperBound
	s.Adc.RawNoiseMargin = config.Sensor.Adc.RawNoiseMargin

	// Normalize noise margin.
	s.Normalized.NoiseMargin = int(float32(config.Sensor.Adc.RawNoiseMargin) * (100 / (float32(s.Adc.RawUpperBound) - float32(s.Adc.RawLowerBound))))
	log.Printf("Sensor: Noise margin is %d %%", s.Normalized.NoiseMargin)

	// Set size of moving average buffer. Min size: 1 (mvg avg filter disabled)
	s.Normalized.MvgAvg.MaxSeriesLen = config.Sensor.MvgAvgLen
	if s.Normalized.MvgAvg.MaxSeriesLen <= 0 {
		s.Normalized.MvgAvg.MaxSeriesLen = 1
	}
	log.Printf("Sensor: Moving average filter length is: %d", s.Normalized.MvgAvg.MaxSeriesLen)
}

/*
 * Feeds new raw sensor value into sensor
 * Saves old value to history
 * Normalizes new value
 * Saves new value to sensor struct
 */
func (s *Sensor) UpdateCurrentValue(currentRaw int) {
	// Back up old value to history
	s.Normalized.History.LastValue = s.Normalized.Current.Value

	// Normalize new value
	currentNormalized := s.normalizeRawValue(currentRaw)
	log.Printf("Normalized value: %d \n", currentNormalized)

	// Feed new value into mean avg filter
	s.mvgAvgAddValue(currentNormalized)

	// Retrieve new mean avg and set it as current value
	s.Normalized.Current.Value = s.Normalized.MvgAvg.Average
	log.Printf("Current value after mvg avg filter: %d \n", s.Normalized.Current.Value)

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

	// Save timestamp of sensor update
	s.LastUpdated = time.Now()
}

/*
 * Calculates normalizes value in a range from 0 - 100 (%).
 * Input: RAW ADC sensor value
 * Putput: Normalization, invertion of value ("dryness" => "wetness")
 * Output: Returns sensor moisture percentage
 */
func (s *Sensor) normalizeRawValue(rawValue int) int {
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

/*
 * Add a new normalized value to moving average filter
 * and calculate and save updated moving average
 */
func (s *Sensor) mvgAvgAddValue(newValue int) {
	/*
	 * Add new value via append as long as there's still capacity
	 * If there's no capacity left, shift everthing to left to make space
	 */
	if len(s.Normalized.MvgAvg.SensorValues) < s.Normalized.MvgAvg.MaxSeriesLen {
		fmt.Println("mvg avg: Filling up ...")
		s.Normalized.MvgAvg.SensorValues = append(s.Normalized.MvgAvg.SensorValues, newValue)
	} else {
		// Shift left
		log.Println("mvg avg: Shifting ...")
		for i := range s.Normalized.MvgAvg.SensorValues {
			if i != s.Normalized.MvgAvg.MaxSeriesLen-1 {
				s.Normalized.MvgAvg.SensorValues[i] = s.Normalized.MvgAvg.SensorValues[i+1]
			} else {
				// Reached last index: Put new value here
				s.Normalized.MvgAvg.SensorValues[i] = newValue
			}
		}
	}

	/*
	 * Recalc average
	 */
	var sum int
	var validValuesCnt = len(s.Normalized.MvgAvg.SensorValues)

	// Sum up
	for _, value := range s.Normalized.MvgAvg.SensorValues {
		sum += value
	}
	// ... and devide
	s.Normalized.MvgAvg.Average = int(math.Round(float64(sum) / float64(validValuesCnt)))
}
