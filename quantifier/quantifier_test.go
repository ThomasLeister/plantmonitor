package quantifier

import (
	"log"
	"testing"

	configManagerPkg "thomas-leister.de/plantmonitor/configmanager"
	sensorPkg "thomas-leister.de/plantmonitor/sensor"
	_ "thomas-leister.de/plantmonitor/testing_init"
)

// Thresholds according to example config:
// 		0 <low> 30    ||    31 <normal> 66    ||    67 <high> 100

const SENSOR_NORMALIZED_VALUE_LOW = 20
const SENSOR_NORMALIZED_VALUE_NORMAL_LOWEREDGE = 31
const SENSOR_NORMALIZED_VALUE_NORMAL = 60
const SENSOR_NORMALIZED_VALUE_NORMAL_UPPEREDGE = 66
const SENSOR_NORMALIZED_VALUE_HIGH_LOWEREDGE = 67
const SENSOR_NORMALIZED_VALUE_HIGH = 90

const HYSTERESIS_MARGIN = 2

/*
 * Test case
 * Contains input value and expected outut values
 * NOTE: Order of execution is important due to hysteresis
 */
type TestCase struct {
	SensorNormalizedValue  int    // Input value for quantifier
	ExpectedLevelDirection int    // Expected level direction
	ExpectedLevelName      string // Expected level name
}

/* Global var for config*/
var config configManagerPkg.Config

/*
 * Test different sensor values and validate quantifier evaluation
 */
func TestEvaluateValue(t *testing.T) {
	var err error
	var testcases []TestCase
	var previousSensorDir int

	// Read config
	config, err = configManagerPkg.ReadConfig("./")
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

	// Create test cases
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_LOW, 0, "low"})        // TC 0: Start with low level
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL, +1, "normal"}) // TC 1: Enter normal level
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_HIGH, +1, "high"})     // TC 2: Enter high level
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL, -1, "normal"}) // TC 3: Back to normal level

	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL, 0, "normal"})     // TC 4: Do not increase sensor value and expect same level as before.
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL + 1, 0, "normal"}) // TC 5: Slightly increase sensor value and expect same level as before.

	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL_LOWEREDGE, 0, "normal"}) // TC 6: Test lower "normal" boundary. Should stay normal level.
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL_UPPEREDGE, 0, "normal"}) // TC 7: Test upper "normal" boundary. Should stay normal level.

	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL_LOWEREDGE - HYSTERESIS_MARGIN, 0, "normal"}) // TC 8: Check hysteresis margin downwards. Should keep level.
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL_UPPEREDGE + HYSTERESIS_MARGIN, 0, "normal"}) // TC 9: Check hysteresis margin upwards. Should keep level.

	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_NORMAL_UPPEREDGE + HYSTERESIS_MARGIN + 1, 1, "high"})  // TC 10: Check hysteresis break-through: Should increase level to "high".
	testcases = append(testcases, TestCase{SENSOR_NORMALIZED_VALUE_HIGH_LOWEREDGE - HYSTERESIS_MARGIN - 1, -1, "normal"}) // TC 11: Check hysteresis break-through: Should decrease level to "normal".

	/*
	 * Test case loop: quantify and validate
	 * Using the testcases
	 */
	for i, testcase := range testcases {
		log.Printf("Running TC %d", i)

		// Mock sensor value direction for hysteresis. This is usually set by Sensor.UpdateCurrentValue()
		if testcase.SensorNormalizedValue > previousSensorDir {
			sensor.Normalized.Current.Direction = +1
		} else if testcase.SensorNormalizedValue < previousSensorDir {
			sensor.Normalized.Current.Direction = -1
		} else if testcase.SensorNormalizedValue == previousSensorDir {
			sensor.Normalized.Current.Direction = 0
		}
		// Save current sensor value in history for next direction calculation
		previousSensorDir = testcase.SensorNormalizedValue

		// Evaluate testcase's sensor value
		levelDirection, currentLevel, err := quantifier.EvaluateValue(testcase.SensorNormalizedValue)

		// Check for eval errors
		if err != nil {
			t.Errorf("\tFailed to run testcase %d: %s", i, err)
			continue
		}

		// Check level direction
		if levelDirection != testcase.ExpectedLevelDirection {
			t.Errorf("\tTestcase %d failed: Expected level direction: %d. Got level direction %d\n", i, testcase.ExpectedLevelDirection, levelDirection)
			continue
		}

		// Check new level's name
		if currentLevel.Name != testcase.ExpectedLevelName {
			t.Errorf("\tTestcase %d failed: Expected level: %s. Got level %s\n", i, testcase.ExpectedLevelName, currentLevel.Name)
			continue
		}

		log.Printf("\tTC %d successful!\n", i)
	}

}
