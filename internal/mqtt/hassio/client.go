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
	Name                string  `json:"name"`
	AvailabilityTopic   string  `json:"avty_t,omitempty"`
	PayloadAvailable    string  `json:"pl_avail,omitempty"`
	PayloadNotAvailable string  `json:"pl_not_avail,omitempty"`
	UniqueId            string  `json:"uniq_id"`
	Device              *device `json:"dev,omitempty"`
}

type sensorConfig struct {
	generalConfig

	StateTopic      string `json:"stat_t"`
	MeasurementUnit string `json:"unit_of_meas,omitempty"`
	Icon            string `json:"ic,omitempty"`
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
		c.MqttClient.Publish(targetTopic, byte(0), false, payload)
	}

	//sensor
	for _, sensor := range config.Sensor {
		targetTopic := fmt.Sprintf("%ssensor/%s_%s/config", c.TopicPrefix, c.DeviceId, friendlyName(sensor.Name))
		payload := c.generatePayloadForSensor(config.Availability, sensor)
		c.MqttClient.Publish(targetTopic, byte(0), false, payload)
	}

	//trigger
	for _, trigger := range config.Trigger {
		targetTopic := fmt.Sprintf("%sswitch/%s/%s/config", c.TopicPrefix, c.DeviceId, friendlyName(trigger.Name))
		payload := c.generatePayloadForTriggerAction(config.Availability, trigger)
		c.MqttClient.Publish(targetTopic, byte(0), false, payload)
	}
}

func friendlyName(name string) string {
	return strings.Replace(name, " ", "_", -1)
}

func (c *Client) generatePayloadForStatus(availability *config.Availability) []byte {
	conf := sensorConfig{
		generalConfig: generalConfig{
			Name:                "Status",
			PayloadAvailable:    availability.Payload.Available,
			PayloadNotAvailable: availability.Payload.Unavailable,
			UniqueId:            fmt.Sprintf("%s_status", c.DeviceId),
			Device: &device{
				Name:         c.DeviceName,
				Ids:          []string{c.DeviceId},
				Manufacturer: "rainu",
				Model:        runtime.GOOS,
				Version:      "mqtt-executor",
			},
		},
		StateTopic: availability.Topic,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		conf.Device.Version += " " + info.Main.Version
	}

	payload, err := json.Marshal(conf)
	if err != nil {
		panic(err)
	}
	return payload
}

func (c *Client) generatePayloadForSensor(availability *config.Availability, sensor config.Sensor) []byte {
	bTrue := true
	conf := sensorConfig{
		generalConfig: generalConfig{
			Name:     sensor.Name,
			UniqueId: fmt.Sprintf("%s_%s", c.DeviceId, friendlyName(sensor.Name)),
			Device: &device{
				Ids: []string{c.DeviceId},
			},
		},
		StateTopic:      sensor.ResultTopic,
		MeasurementUnit: sensor.Unit,
		Icon:            "",
		ForceUpdate:     &bTrue,
	}
	if availability != nil {
		conf.AvailabilityTopic = availability.Topic
		conf.PayloadAvailable = availability.Payload.Available
		conf.PayloadNotAvailable = availability.Payload.Unavailable
	}

	payload, err := json.Marshal(conf)
	if err != nil {
		panic(err)
	}
	return payload
}

func (c *Client) generatePayloadForTriggerAction(availability *config.Availability, trigger config.Trigger) []byte {
	conf := triggerConfig{
		generalConfig: generalConfig{
			Name:     fmt.Sprintf("%s", trigger.Name),
			UniqueId: fmt.Sprintf("%s_%s", c.DeviceId, friendlyName(trigger.Name)),
			Device: &device{
				Ids: []string{c.DeviceId},
			},
		},
		CommandTopic: trigger.Topic,
		PayloadStart: mqtt.PayloadStart,
		PayloadStop:  mqtt.PayloadStop,
		StateTopic:   fmt.Sprintf("%s/%s", trigger.Topic, mqtt.TopicSuffixState),
		StateRunning: mqtt.PayloadStatusRunning,
		StateStopped: mqtt.PayloadStatusStopped,
	}
	if availability != nil {
		conf.AvailabilityTopic = availability.Topic
		conf.PayloadAvailable = availability.Payload.Available
		conf.PayloadNotAvailable = availability.Payload.Unavailable
	}

	payload, err := json.Marshal(conf)
	if err != nil {
		panic(err)
	}
	return payload
}
