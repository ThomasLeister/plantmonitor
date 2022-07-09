/*
 * Configmanager:
 * Reads configuration from YAML config file
 */

package configmanager

import (
	"os"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Xmpp struct {
		Host string	`yaml:"host"`
		Port int	`yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Recipient string `yaml:"recipient"`
	} `yaml:"xmpp"`

	Mqtt struct {
		Host string `yaml:"host"`
		Port int `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Topic string `yaml:"topic"`
	} `yaml:"mqtt"`

	Giphy struct {
		ApiKey string `yaml:"api_key"`
	}

	Sensor struct {
		Adc struct {
			RawLowerBound int `yaml:"raw_lower_bound"`
			RawUpperBound int `yaml:"raw_upper_bound"`
		} `yaml:"adc"`
	} `yaml:"sensor"`

	Levels []struct {
		Start int `yaml:"start"`
		End int `yaml:"end"`
		Name string `yaml:"name"`
		ChatMessageSteady string `yaml:"chat_message_steady"`
		ChatMessageUp string `yaml:"chat_message_up"`
		ChatMessageDown string `yaml:"chat_message_down"`
		ChatMessageReminder string `yaml:"chat_message_reminder"`
		Urgency string `yaml:"urgency"`
		NotificationInterval int `yaml:"notification_interval"`
	}`yaml:"levels"`
}


func ReadConfig(configPath string) (Config, error) {
	config := Config{}

	// Open config file
    file, err := os.Open(configPath)
    if err != nil {
        return config, err
    }
    defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)
	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return config, err
	}

	return config, err
}