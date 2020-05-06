package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type TopicConfigurations struct {
	Availability *Availability `json:"availability,omitempty"`
	Trigger      []Trigger     `json:"trigger"`
	Sensor       []Sensor      `json:"sensor"`
}

type Availability struct {
	Topic   string `json:"topic"`
	Payload struct {
		Available   string `json:"available"`
		Unavailable string `json:"unavailable"`
	} `json:"payload"`
	Interval *Interval `json:"interval"`
}

type Trigger struct {
	Name    string  `json:"name"`
	Topic   string  `json:"topic"`
	Command Command `json:"command"`
}

type Sensor struct {
	Name        string   `json:"name"`
	ResultTopic string   `json:"topic"`
	Interval    Interval `json:"interval"`
	Unit        string   `json:"unit"`
	Command     Command  `json:"command"`
}

type Command struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments"`
}

func LoadTopicConfiguration(configFile, deviceId string) (TopicConfigurations, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return TopicConfigurations{}, fmt.Errorf("error while opening topic configuration file: %w", err)
	}
	defer file.Close()

	var topicConfig TopicConfigurations
	err = json.NewDecoder(file).Decode(&topicConfig)
	if err != nil {
		return TopicConfigurations{}, fmt.Errorf("could not read topic configuration file: %w", err)
	}

	err = topicConfig.validate()
	if err != nil {
		return TopicConfigurations{}, fmt.Errorf("invalid config: %w", err)
	}

	//replace DEVICE_ID
	if topicConfig.Availability != nil {
		topicConfig.Availability.Topic = strings.Replace(topicConfig.Availability.Topic, "__DEVICE_ID__", deviceId, -1)
		if topicConfig.Availability.Interval == nil {
			defaultInterval := Interval(30 * time.Second)
			topicConfig.Availability.Interval = &defaultInterval
		}
		if topicConfig.Availability.Payload.Available == "" {
			topicConfig.Availability.Payload.Available = "Online"
		}
		if topicConfig.Availability.Payload.Unavailable == "" {
			topicConfig.Availability.Payload.Unavailable = "Offline"
		}
	}
	for i := range topicConfig.Trigger {
		topicConfig.Trigger[i].Topic = strings.Replace(topicConfig.Trigger[i].Topic, "__DEVICE_ID__", deviceId, -1)
	}
	for i := range topicConfig.Sensor {
		topicConfig.Sensor[i].ResultTopic = strings.Replace(topicConfig.Sensor[i].ResultTopic, "__DEVICE_ID__", deviceId, -1)
	}

	return topicConfig, nil
}

func (t *TopicConfigurations) validate() error {
	//TODO
	//keine wildcards in Topic-Namen
	//Keine Sonderzeichen in Payloads
	//Trigger-Name muss einzigartig sein
	return nil
}
