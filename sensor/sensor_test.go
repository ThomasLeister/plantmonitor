package sensor

import (
	"log"
	"testing"

	configManagerPkg "thomas-leister.de/plantmonitor/configmanager"
	_ "thomas-leister.de/plantmonitor/testing_init"
)

/* Global var for config*/
var config configManagerPkg.Config

func TestNormalizeRawValue(t *testing.T) {
	var err error

	// Define test cases
	var testData = map[int]int{
		3624: 0,
		2557: 50,
		2493: 53,
		2472: 54,
		1491: 100,
	}

	// Read config
	config, err = configManagerPkg.ReadConfig("./")
	if err != nil {
		log.Fatal("Could not parse config:", err)
	} else {
		log.Println("Config was read and parsed!")
	}

	// Init sensor
	sensor := Sensor{}
	sensor.Init(&config)

	// Loop through testcases
	for input, expected := range testData {
		if result := sensor.NormalizeRawValue(input); result != expected {
			t.Errorf("Expected value for raw value %d: %d. But got %d", input, expected, result)
		}
	}
}
