package main

import (
	"encoding/json"
	"flag"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"os"
)

type config struct {
	Broker       *string
	SubscribeQOS *int
	PublishQOS   *int
	Username     *string
	Password     *string
	ClientId     *string

	TopicConfigFile     *string
	TopicConfigurations TopicConfigurations
}

var Config config

type TopicConfigurations map[string]TopicConfiguration
type TopicConfiguration map[string][]string

func LoadConfig() {
	Config = config{
		Broker:       flag.String("broker", "", "The broker URI. ex: tcp://127.0.0.1:1883"),
		SubscribeQOS: flag.Int("sub-qos", 0, "The Quality of Service for subscription 0,1,2 (default 0)"),
		PublishQOS:   flag.Int("pub-qos", 0, "The Quality of Service for publishing 0,1,2 (default 0)"),
		Username:     flag.String("user", "", "The User (optional)"),
		Password:     flag.String("password", "", "The password (optional)"),
		ClientId:     flag.String("client-id", "mqtt-executor", "The ClientID (optional)"),

		TopicConfigFile: flag.String("config", "./config.json", "The topic configuration file"),
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
	if *Config.TopicConfigFile == "" {
		zap.L().Fatal("Topic configuration file is missing!")
	} else {
		file, err := os.Open(*Config.TopicConfigFile)
		if err != nil {
			zap.L().Fatal("Error while opening topic configuration file: %s", zap.Error(err))
		}
		defer file.Close()

		err = json.NewDecoder(file).Decode(&Config.TopicConfigurations)
		if err != nil {
			zap.L().Fatal("Could not read topic configuration file: %s", zap.Error(err))
		}
	}
}

func (c *config) GetMQTTOpts() *MQTT.ClientOptions {
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

	return opts
}
