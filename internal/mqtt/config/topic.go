package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

type TopicConfigurations struct {
	Availability *Availability `json:"availability,omitempty"`
	Trigger      []Trigger     `json:"trigger"`
	Sensor       []Sensor      `json:"sensor"`
}

type Availability struct {
	Topic   string              `json:"topic"`
	Payload availabilityPayload `json:"payload"`
}
type availabilityPayload struct {
	Available   string `json:"available"`
	Unavailable string `json:"unavailable"`
}

type Trigger struct {
	Name    string  `json:"name"`
	Topic   string  `json:"topic"`
	Icon    string  `json:"icon"`
	Command Command `json:"command"`
}

type Sensor struct {
	Name        string   `json:"name"`
	ResultTopic string   `json:"topic"`
	Retained    bool     `json:"retained"`
	Interval    Interval `json:"interval"`
	Unit        string   `json:"unit"`
	Icon        string   `json:"icon"`
	Command     Command  `json:"command"`
}

type Command struct {
	Name      string   `json:"name"`
	Arguments []string `json:"arguments"`
}

func LoadTopicConfiguration(configFilePath, deviceId string) (TopicConfigurations, error) {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return TopicConfigurations{}, fmt.Errorf("error while opening topic configuration file: %w", err)
	}
	defer configFile.Close()

	var topicConfig TopicConfigurations
	err = json.NewDecoder(configFile).Decode(&topicConfig)
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
	if t.Availability != nil {
		if err := checkTopicName(t.Availability.Topic); err != nil {
			return fmt.Errorf("invalid availability topic: %w", err)
		}
	}

	sensorNames := map[string]bool{}
	for i, sensor := range t.Sensor {
		if err := validateSensor(sensor); err != nil {
			return fmt.Errorf("invalid sensor (#%d): %w", i, err)
		}

		if _, exists := sensorNames[sensor.Name]; exists {
			return fmt.Errorf("invalid sensor (#%d): sensor with this name already exists", i)
		}
		sensorNames[sensor.Name] = true
	}

	triggerNames := map[string]bool{}
	for i, trigger := range t.Trigger {
		if err := validateTrigger(trigger); err != nil {
			return fmt.Errorf("invalid trigger (#%d): %w", i, err)
		}

		if _, exists := triggerNames[trigger.Name]; exists {
			return fmt.Errorf("invalid trigger (#%d): trigger with this name already exists", i)
		}
		triggerNames[trigger.Name] = true
	}

	return nil
}

func validateSensor(sensor Sensor) error {
	if sensor.Name == "" {
		return errors.New("name must not be empty")
	}
	if time.Duration(sensor.Interval).Nanoseconds() == 0 {
		return errors.New("invalid duration")
	}
	if err := checkTopicName(sensor.ResultTopic); err != nil {
		return fmt.Errorf("invalid topic: %w", err)
	}
	if sensor.Command.Name == "" {
		return errors.New("command name must not be empty")
	}
	return nil
}

func validateTrigger(trigger Trigger) error {
	if trigger.Name == "" {
		return errors.New("name must not be empty")
	}
	if err := checkTopicName(trigger.Topic); err != nil {
		return fmt.Errorf("invalid topic: %w", err)
	}
	if trigger.Command.Name == "" {
		return errors.New("command name must not be empty")
	}
	return nil
}

var topicRegex = regexp.MustCompile(`^[a-zA-Z0-9_/]*$`)

func checkTopicName(topic string) error {
	if strings.Trim(topic, " ") == "" {
		return errors.New("must not be empty")
	}
	if !topicRegex.MatchString(topic) {
		return errors.New("invalid character")
	}

	return nil
}
