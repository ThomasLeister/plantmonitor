package quantifier

import (
	"log"
	"testing"

	configManagerPkg "thomas-leister.de/plantmonitor/configmanager"
	sensorPkg "thomas-leister.de/plantmonitor/sensor"
)

// Thresholds according to example config:
// 0  <low>  30 | 31  <normal>  66 | 67 <high> 100

const SENSOR_NORMALIZED_VALUE_LOW = 20
const SENSOR_NORMALIZED_VALUE_LOW_EDGE_HIGHER = 30
const SENSOR_NORMALIZED_VALUE_NORMAL = 60
const SENSOR_NORMALIZED_VALUE_HIGH = 90
const SENSOR_NORMALIZED_VALUE_HIGH_EDGE_LOWER = 67

/* Global var for config*/
var config configManagerPkg.Config

/*
 *
 * TC-5 and TC-6: Test hysteresis by using edge values
 */
func TestEvaluateValue(t *testing.T) {
	var err error
	var levelDirection int
	var currentLevel QuantificationLevel

	// Read config
	config, err = configManagerPkg.ReadConfig("../config.yaml")
	if err != nil {
		log.Fatal("Could not parse config:", err)
	} else {
		log.Println("Config was read and parsed!")
	}

	// Init sensor
	sensor := sensorPkg.Sensor{}
	sensor.Init(&config)

	// Init quantifier
	quantifier := Quantifier{}
	quantifier.Init(&config, &sensor)

	/*
	 * Evaluate first value to create history
	 */
	sensor.Normalized.Current.Direction = 0
	levelDirection, currentLevel, err = quantifier.EvaluateValue(SENSOR_NORMALIZED_VALUE_LOW)
	if err != nil {
		log.Panic("Error happended during evaluation.")
	}

	// Check if level is correct: We expect steady, because we don't have history.
	if levelDirection != 0 {
		t.Errorf("TC-1: Expected levelDirection == 0")
	}
	if currentLevel.Name != "low" {
		t.Errorf("TC-1: Expected currentLevel == \"low\"")
	}

	/*
	 * Evaluate second value for testing
	 */
	sensor.Normalized.Current.Direction = 1
	levelDirection, currentLevel, err = quantifier.EvaluateValue(SENSOR_NORMALIZED_VALUE_NORMAL)
	if err != nil {
		log.Panic("Error happended during evaluation.")
	}

	// Check if level is correct: We expect rising
	if levelDirection != +1 {
		t.Errorf("TC-2: Expected levelDirection == +1")
	}
	if currentLevel.Name != "normal" {
		t.Errorf("TC-2: Expected currentLevel == \"normal\"")
	}

	/*
	 * Evaluate third value for testing
	 */
	sensor.Normalized.Current.Direction = 1
	levelDirection, currentLevel, err = quantifier.EvaluateValue(SENSOR_NORMALIZED_VALUE_HIGH)
	if err != nil {
		log.Panic("Error happended during evaluation.")
	}

	// Check if level is correct: We expect rising
	if levelDirection != +1 {
		t.Errorf("TC-3: Expected levelDirection == +1")
	}
	if currentLevel.Name != "high" {
		t.Errorf("TC-3: Expected currentLevel == \"high\"")
	}

	/*
	 * Evaluate fourth value for testing
	 */
	sensor.Normalized.Current.Direction = -1
	levelDirection, currentLevel, err = quantifier.EvaluateValue(SENSOR_NORMALIZED_VALUE_NORMAL)
	if err != nil {
		log.Panic("Error happended during evaluation.")
	}

	// Check if level is correct: We expect rising
	if levelDirection != -1 {
		t.Errorf("TC-4: Expected levelDirection == -1")
	}
	if currentLevel.Name != "normal" {
		t.Errorf("TC-4: Expected currentLevel == \"normal\"")
	}

	/*
	 * Evaluate fifth value for testing
	 * Input: low level highter edge value
	 * Expected output: Should not (yet) reach "low" level due to Hysteresis (shift: sensor.Normalized.NoiseMargin | default: -2)
	 */
	sensor.Normalized.Current.Direction = -1
	levelDirection, currentLevel, err = quantifier.EvaluateValue(SENSOR_NORMALIZED_VALUE_LOW_EDGE_HIGHER)
	if err != nil {
		log.Panic("Error happended during evaluation.")
	}

	// Check if level is correct: We expect rising
	if levelDirection != 0 {
		t.Errorf("TC-5: Expected levelDirection == 0")
	}
	if currentLevel.Name != "normal" {
		t.Errorf("TC-5: Expected currentLevel == \"normal\"")
	}

	/*
	 * Evaluate sixth value for testing
	 * Input: high level lower edge value
	 * Expected output: Should not (yet) reach "high" level due to Hysteresis (shift: sensor.Normalized.NoiseMargin | default: -2)
	 */
	sensor.Normalized.Current.Direction = +1
	levelDirection, currentLevel, err = quantifier.EvaluateValue(SENSOR_NORMALIZED_VALUE_HIGH_EDGE_LOWER)
	if err != nil {
		log.Panic("Error happended during evaluation.")
	}

	// Check if level is correct: We expect rising
	if levelDirection != 0 {
		t.Errorf("TC-6: Expected levelDirection == 0")
	}
	if currentLevel.Name != "normal" {
		t.Errorf("TC-6: Expected currentLevel == \"normal\"")
	}
}
