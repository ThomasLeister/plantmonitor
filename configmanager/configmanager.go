/*
 * Configmanager:
 * Reads configuration from YAML config file
 */

package configmanager

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type MessageType struct {
	Messages    []string `yaml:"messages"`
	GifKeywords string   `yaml:"gif_keywords,omitempty"`
}

type Messages struct {
	Online  []string               `yaml:"online"`
	Levels  map[string]MessageType `yaml:"levels"`
	Answers struct {
		CurrentState          string `yaml:"current_state"`
		UnknownCommand        string `yaml:"unknown_command"`
		AvailableCommands     string `yaml:"available_commands"`
		SensorDataUnavailable string `yaml:"sensor_data_unavailable"`
	} `yaml:"answers"`
	Warnings struct {
		SensorOffline string `yaml:"sensor_offline"`
	} `yaml:"warnings"`
}

type Config struct {
	Xmpp struct {
		Host       string   `yaml:"host"`
		Port       int      `yaml:"port"`
		Username   string   `yaml:"username"`
		Password   string   `yaml:"password"`
		Recipients []string `yaml:"recipients"`
	} `yaml:"xmpp"`

	Mqtt struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Topic    string `yaml:"topic"`
		ClientId string `yaml:"client_id"`
	} `yaml:"mqtt"`

	Watchdog struct {
		Timeout int `yaml:"timeout"`
	} `yaml:"watchdog"`

	Giphy struct {
		ApiKey string `yaml:"api_key"`
	}

	Sensor struct {
		Adc struct {
			RawLowerBound  int `yaml:"raw_lower_bound"`
			RawUpperBound  int `yaml:"raw_upper_bound"`
			RawNoiseMargin int `yaml:"raw_noise_margin"`
		} `yaml:"adc"`
	} `yaml:"sensor"`

	Levels []struct {
		Start                int    `yaml:"start"`
		End                  int    `yaml:"end"`
		Name                 string `yaml:"name"`
		NotificationInterval int    `yaml:"notification_interval"`
	} `yaml:"levels"`

	LangCode string `yaml:"lang_code"`

	Messages Messages // Not part of config.yaml, but language config will be put here.
}

func ReadConfig(configFilePath string) (Config, error) {
	config := Config{}

	log.Println("Initializing configmanager ...")

	/*
	 * Parse main config file config.yaml
	 */
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	// Decode config file
	configDecoder := yaml.NewDecoder(configFile)
	if err := configDecoder.Decode(&config); err != nil {
		return config, err
	}

	/*
	 * Parse language config file lang_<lang>.yaml
	 */

	langFile, err := os.Open("lang_" + config.LangCode + ".yaml")
	if err != nil {
		return config, err
	}
	defer langFile.Close()

	// Decode language file
	langFileDecoder := yaml.NewDecoder(langFile)
	if err := langFileDecoder.Decode(&config.Messages); err != nil {
		return config, err
	}

	return config, err
}
