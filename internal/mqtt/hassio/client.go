package hassio

import (
	"encoding/json"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-executor/internal/mqtt"
	"github.com/rainu/mqtt-executor/internal/mqtt/config"
	"go.uber.org/zap"
	"runtime"
	"runtime/debug"
	"strings"
)

type generalConfig struct {
	Name                string `json:"name"`
	AvailabilityTopic   string `json:"avty_t,omitempty"`
	PayloadAvailable    string `json:"pl_avail,omitempty"`
	PayloadNotAvailable string `json:"pl_not_avail,omitempty"`
	UniqueId            string `json:"uniq_id"`
	Icon                string `json:"ic,omitempty"`
	Device              device `json:"dev,omitempty"`
}

type sensorConfig struct {
	generalConfig

	StateTopic      string `json:"stat_t"`
	MeasurementUnit string `json:"unit_of_meas,omitempty"`
	ForceUpdate     *bool  `json:"frc_upd,omitempty"`
}

type triggerConfig struct {
	generalConfig

	CommandTopic string `json:"cmd_t"`
	StateTopic   string `json:"stat_t"`
	PayloadStart string `json:"pl_on"`
	PayloadStop  string `json:"pl_off"`
	StateRunning string `json:"stat_on"`
	StateStopped string `json:"stat_off"`
}

type device struct {
	Name         string   `json:"name,omitempty"`
	Ids          []string `json:"ids"`
	Model        string   `json:"mdl,omitempty"`
	Manufacturer string   `json:"mf,omitempty"`
	Version      string   `json:"sw,omitempty"`
}

type AvailabilityConfig struct {
	Topic               string
	AvailablePayload    string
	NotAvailablePayload string
}

type Client struct {
	DeviceName  string
	DeviceId    string
	TopicPrefix string
	MqttClient  MQTT.Client
}

func (c *Client) PublishDiscoveryConfig(config config.TopicConfigurations) {
	zap.L().Info("Initialise homeassistant config.")

	//status
	if config.Availability != nil {
		targetTopic := fmt.Sprintf("%ssensor/%s_status/config", c.TopicPrefix, c.DeviceId)
		payload := c.generatePayloadForStatus(config.Availability)
		c.MqttClient.Publish(targetTopic, byte(1), true, payload)
	}

	//sensor
	for _, sensor := range config.Sensor {
		targetTopic := fmt.Sprintf("%ssensor/%s_%s/config", c.TopicPrefix, c.DeviceId, friendlyName(sensor.Name))
		payload := c.generatePayloadForSensor(config.Availability, sensor)
		c.MqttClient.Publish(targetTopic, byte(1), true, payload)
	}

	//trigger
	for _, trigger := range config.Trigger {
		targetTopic := fmt.Sprintf("%sswitch/%s/%s/config", c.TopicPrefix, c.DeviceId, friendlyName(trigger.Name))
		payload := c.generateSwitchPayloadForTriggerAction(config.Availability, trigger)
		c.MqttClient.Publish(targetTopic, byte(1), true, payload)

		//publish the trigger-result as sensor data
		targetTopic = fmt.Sprintf("%ssensor/%s_%s/result/config", c.TopicPrefix, c.DeviceId, friendlyName(trigger.Name))
		payload = c.generateResultPayloadForTriggerAction(config.Availability, trigger)
		c.MqttClient.Publish(targetTopic, byte(1), true, payload)

		//publish the trigger-state as sensor data
		targetTopic = fmt.Sprintf("%ssensor/%s_%s/state/config", c.TopicPrefix, c.DeviceId, friendlyName(trigger.Name))
		payload = c.generateStatePayloadForTriggerAction(config.Availability, trigger)
		c.MqttClient.Publish(targetTopic, byte(1), true, payload)
	}
}

func friendlyName(name string) string {
	return strings.Replace(name, " ", "_", -1)
}

func (c *Client) buildDevice() device {
	return device{
		Name:         c.DeviceName,
		Ids:          []string{c.DeviceId},
		Manufacturer: "rainu",
		Model:        runtime.GOOS,
		Version:      "mqtt-executor",
	}
}

func (c *Client) generatePayloadForStatus(availability *config.Availability) []byte {
	conf := sensorConfig{
		generalConfig: generalConfig{
			Name:                "Status",
			PayloadAvailable:    availability.Payload.Available,
			PayloadNotAvailable: availability.Payload.Unavailable,
			UniqueId:            fmt.Sprintf("%s_status", c.DeviceId),
			Device:              c.buildDevice(),
		},
		StateTopic: availability.Topic,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		conf.Device.Version += " " + info.Main.Version
	}

	payload, err := json.Marshal(conf)
	if err != nil {
		//the "marshalling" is relatively safe - it should never appear at runtime
		panic(err)
	}
	return payload
}

func (c *Client) generatePayloadForSensor(availability *config.Availability, sensor config.Sensor) []byte {
	bTrue := true
	conf := sensorConfig{
		generalConfig: generalConfig{
			Name:     sensor.Name,
			Icon:     sensor.Icon,
			UniqueId: fmt.Sprintf("%s_%s", c.DeviceId, friendlyName(sensor.Name)),
			Device:   c.buildDevice(),
		},
		StateTopic:      sensor.ResultTopic,
		MeasurementUnit: sensor.Unit,
		ForceUpdate:     &bTrue,
	}
	addAvailability(&conf.generalConfig, availability)

	payload, err := json.Marshal(conf)
	if err != nil {
		//the "marshalling" is relatively safe - it should never appear at runtime
		panic(err)
	}
	return payload
}

func (c *Client) generateSwitchPayloadForTriggerAction(availability *config.Availability, trigger config.Trigger) []byte {
	conf := triggerConfig{
		generalConfig: generalConfig{
			Name:     fmt.Sprintf("%s", trigger.Name),
			Icon:     trigger.Icon,
			UniqueId: fmt.Sprintf("%s_%s", c.DeviceId, friendlyName(trigger.Name)),
			Device:   c.buildDevice(),
		},
		CommandTopic: trigger.Topic,
		PayloadStart: mqtt.PayloadStart,
		PayloadStop:  mqtt.PayloadStop,
		StateTopic:   fmt.Sprintf("%s/%s", trigger.Topic, mqtt.TopicSuffixState),
		StateRunning: mqtt.PayloadStatusRunning,
		StateStopped: mqtt.PayloadStatusStopped,
	}
	addAvailability(&conf.generalConfig, availability)

	payload, err := json.Marshal(conf)
	if err != nil {
		//the "marshalling" is relatively safe - it should never appear at runtime
		panic(err)
	}
	return payload
}

func (c *Client) generateResultPayloadForTriggerAction(availability *config.Availability, trigger config.Trigger) []byte {
	conf := sensorConfig{
		generalConfig: generalConfig{
			Name:     fmt.Sprintf("%s - Result", trigger.Name),
			Icon:     trigger.Icon,
			UniqueId: fmt.Sprintf("%s_%s_result", c.DeviceId, friendlyName(trigger.Name)),
			Device:   c.buildDevice(),
		},
		StateTopic: fmt.Sprintf("%s/%s", trigger.Topic, mqtt.TopicSuffixResult),
	}
	addAvailability(&conf.generalConfig, availability)

	payload, err := json.Marshal(conf)
	if err != nil {
		//the "marshalling" is relatively safe - it should never appear at runtime
		panic(err)
	}
	return payload
}

func (c *Client) generateStatePayloadForTriggerAction(availability *config.Availability, trigger config.Trigger) []byte {
	conf := sensorConfig{
		generalConfig: generalConfig{
			Name:     fmt.Sprintf("%s - State", trigger.Name),
			Icon:     trigger.Icon,
			UniqueId: fmt.Sprintf("%s_%s_state", c.DeviceId, friendlyName(trigger.Name)),
			Device:   c.buildDevice(),
		},
		StateTopic: fmt.Sprintf("%s/%s", trigger.Topic, mqtt.TopicSuffixState),
	}
	addAvailability(&conf.generalConfig, availability)

	payload, err := json.Marshal(conf)
	if err != nil {
		//the "marshalling" is relatively safe - it should never appear at runtime
		panic(err)
	}
	return payload
}

func addAvailability(config *generalConfig, availability *config.Availability) {
	if availability != nil {
		config.AvailabilityTopic = availability.Topic
		config.PayloadAvailable = availability.Payload.Available
		config.PayloadNotAvailable = availability.Payload.Unavailable
	}
}
