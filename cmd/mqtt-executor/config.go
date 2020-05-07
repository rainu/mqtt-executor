package main

import (
	"flag"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	internalConf "github.com/rainu/mqtt-executor/internal/mqtt/config"
	"go.uber.org/zap"
)

type applicationConfig struct {
	Broker       *string
	SubscribeQOS *int
	PublishQOS   *int
	Username     *string
	Password     *string
	ClientId     *string
	DeviceName   *string
	DeviceId     *string

	HomeassistantEnable *bool
	HomeassistantTopic  *string

	TopicConfigFile     *string
	TopicConfigurations internalConf.TopicConfigurations
}

var Config applicationConfig

func LoadConfig() {
	deviceId, _ := machineid.ID()
	if len(deviceId) > 8 {
		deviceId = deviceId[:8]
	}

	Config = applicationConfig{
		Broker:       flag.String("broker", "", "The broker URI. ex: tcp://127.0.0.1:1883"),
		SubscribeQOS: flag.Int("sub-qos", 0, "The Quality of Service for subscription 0,1,2 (default 0)"),
		PublishQOS:   flag.Int("pub-qos", 0, "The Quality of Service for publishing 0,1,2 (default 0)"),
		Username:     flag.String("user", "", "The User (optional)"),
		Password:     flag.String("password", "", "The password (optional)"),
		ClientId:     flag.String("client-id", "mqtt-executor", "The ClientID (optional)"),
		DeviceName:   flag.String("device-name", fmt.Sprintf("MQTTExecutor - %s", deviceId), "The name of this device (optional)"),
		DeviceId:     flag.String("device-id", deviceId, "A unique device id (optional)"),

		HomeassistantEnable: flag.Bool("home-assistant", false, "Enable home assistant support (optional)"),
		HomeassistantTopic:  flag.String("ha-discovery-prefix", "homeassistant/", "The mqtt topic prefix for homeassistant's discovery (optional)"),
		TopicConfigFile:     flag.String("config", "./config.json", "The topic configuration file"),
	}
	flag.Parse()

	if *Config.Broker == "" {
		zap.L().Fatal("Broker is missing!")
	}
	if *Config.SubscribeQOS != 0 && *Config.SubscribeQOS != 1 && *Config.SubscribeQOS != 2 {
		zap.L().Fatal("Invalid qos level!")
	}
	if *Config.PublishQOS != 0 && *Config.PublishQOS != 1 && *Config.PublishQOS != 2 {
		zap.L().Fatal("Invalid qos level!")
	}
	if *Config.DeviceId == "" {
		zap.L().Fatal("Invalid device id!")
	}
	if *Config.TopicConfigFile == "" {
		zap.L().Fatal("Topic configuration file is missing!")
	} else {
		configuration, err := internalConf.LoadTopicConfiguration(*Config.TopicConfigFile, *Config.DeviceId)
		if err != nil {
			zap.L().Fatal("Error while read topic configuration: %s", zap.Error(err))
		}
		Config.TopicConfigurations = configuration
	}
}

func (c *applicationConfig) GetMQTTOpts(
	onConn MQTT.OnConnectHandler,
	onLost MQTT.ConnectionLostHandler) *MQTT.ClientOptions {

	opts := MQTT.NewClientOptions()

	opts.AddBroker(*c.Broker)
	if *c.Username != "" {
		opts.SetUsername(*c.Username)
	}
	if *c.Password != "" {
		opts.SetPassword(*c.Password)
	}
	if *c.ClientId != "" {
		opts.SetClientID(*c.ClientId)
	}

	if c.TopicConfigurations.Availability != nil {
		opts.WillEnabled = true
		opts.WillQos = byte(*c.PublishQOS)
		opts.WillPayload = []byte(c.TopicConfigurations.Availability.Payload.Unavailable)
		opts.WillTopic = c.TopicConfigurations.Availability.Topic
	}

	opts.SetOnConnectHandler(onConn)
	opts.SetConnectionLostHandler(onLost)

	return opts
}
