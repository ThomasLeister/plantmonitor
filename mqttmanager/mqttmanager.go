package mqttmanager

import (
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"thomas-leister.de/plantmonitor/configmanager"
)

type MqttClient struct {
	Host               string
	Port               int
	Username           string
	Password           string
	Topic              string
	connectHandler     mqtt.OnConnectHandler
	connectLostHandler mqtt.OnConnectHandler
}

type MqttDecodedPayload struct {
	MoistureRaw uint16 `json:"moisture_raw"`
}

type MqttUplinkMessage struct {
	DecodedPayload MqttDecodedPayload `json:"decoded_payload"` //decoded_payload stores the already-decoded payload from TTN
}

type MqttPayload struct {
	UplinkMessage MqttUplinkMessage `json:"uplink_message"`
}

func (m *MqttClient) ParseMqttMessage(mqttMessage mqtt.Message) MqttPayload {
	var mqttPayload MqttPayload

	err := json.Unmarshal(mqttMessage.Payload(), &mqttPayload)
	if err != nil {
		panic(err)
	}

	return mqttPayload
}

func (m *MqttClient) ConnectHandler(client mqtt.Client) {
	log.Printf("MQTT: Connected to %s \n", m.Host)
}

func (m *MqttClient) ConnectLostHandler(client mqtt.Client, err error) {
	log.Printf("MQTT: Connection lost: %v \n", err)
}

func (m *MqttClient) Init(config *configmanager.Config) {
	m.Host = config.Mqtt.Host
	m.Port = config.Mqtt.Port
	m.Username = config.Mqtt.Username
	m.Password = config.Mqtt.Password
	m.Topic = config.Mqtt.Topic
}

func (m *MqttClient) RunMQTTListener(mqttMessageChannel chan mqtt.Message) {
	opts := mqtt.NewClientOptions()

	// Set options for connection
	opts.AddBroker(fmt.Sprintf("mqtts://%s:%d", m.Host, m.Port))
	opts.SetClientID("go_mqtt_client")
	opts.SetUsername(m.Username)
	opts.SetPassword(m.Password)

	// Set callback functions
	opts.SetDefaultPublishHandler(func(c mqtt.Client, m mqtt.Message) {
		mqttMessageChannel <- m
	})
	opts.OnConnect = m.ConnectHandler
	opts.OnConnectionLost = m.ConnectLostHandler

	// Create client
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to topic
	token := client.Subscribe(m.Topic, 1, nil)
	token.Wait()
	log.Printf("MQTT: Subscribed to topic %s \n", m.Topic)
}
