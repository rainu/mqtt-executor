package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadTopicConfiguration(t *testing.T) {
	deviceId := "D3V1C3"
	tests := []struct {
		name           string
		content        string
		expectedResult TopicConfigurations
		expectedError  string
	}{
		{
			name:          "empty config",
			expectedError: "could not read topic configuration file: EOF",
		},
		{
			name:    "empty json",
			content: "{}",
		},
		{
			name: "Availability",
			content: `{
				"availability": {
					"topic": "tele/__DEVICE_ID__/status",
					"payload": {
						"available": "ON",
						"unavailable": "OFF"
					}
				}
			}`, expectedResult: TopicConfigurations{
				Availability: &Availability{
					Topic: fmt.Sprintf("tele/%s/status", deviceId),
					Payload: availabilityPayload{
						Available:   "ON",
						Unavailable: "OFF",
					},
				},
			},
		},
		{
			name: "Availability default values",
			content: `{
				"availability": {
					"topic": "tele/__DEVICE_ID__/status"
				}
			}`, expectedResult: TopicConfigurations{
				Availability: &Availability{
					Topic: fmt.Sprintf("tele/%s/status", deviceId),
					Payload: availabilityPayload{
						Available:   "Online",
						Unavailable: "Offline",
					},
				},
			},
		},
		{
			name:          "Availability invalid topic",
			content:       `{ "availability": { "topic": "tele/+/status" } }`,
			expectedError: "invalid config: invalid availability topic: invalid character",
		},
		{
			name:          "Availability empty topic",
			content:       `{ "availability": { "topic": "" } }`,
			expectedError: "invalid config: invalid availability topic: must not be empty",
		},
		{
			name: "Sensor",
			content: `{
				"sensor": [{
					"name": "My sweat sensor",
					"topic": "tele/__DEVICE_ID__/status",
					"unit": "kWh",
					"interval": "13s",
					"icon": "hassio-icon",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`, expectedResult: TopicConfigurations{
				Sensor: []Sensor{{
					GeneralSensor: GeneralSensor{
						ResultTopic: fmt.Sprintf("tele/%s/status", deviceId),
						Interval:    *interval(13 * time.Second),
						Command: Command{
							Name:      "/usr/bin/bash",
							Arguments: []string{"echo"},
						},
					},
					Name: "My sweat sensor",
					Unit: "kWh",
					Icon: "hassio-icon",
				}},
			},
		},
		{
			name: "Sensor missing interval",
			content: `{
				"sensor": [{
					"name": "My sweat sensor",
					"topic": "tele/__DEVICE_ID__/status",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid sensor (#0): invalid duration",
		},
		{
			name: "Sensor missing name",
			content: `{
				"sensor": [{
					"topic": "tele/__DEVICE_ID__/status",
					"interval": "13s",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid sensor (#0): name must not be empty",
		},
		{
			name: "Sensor invalid topic",
			content: `{
				"sensor": [{
					"name": "My sweat sensor",
					"topic": "tele/+/status",
					"interval": "13s",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid sensor (#0): invalid topic: invalid character",
		},
		{
			name: "Sensor empty topic",
			content: `{
				"sensor": [{
					"name": "My sweat sensor",
					"interval": "13s",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid sensor (#0): invalid topic: must not be empty",
		},
		{
			name: "Sensor empty command",
			content: `{
				"sensor": [{
					"name": "My sweat sensor",
					"topic": "tele/status",
					"interval": "13s",
					"command": {
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid sensor (#0): command name must not be empty",
		},
		{
			name: "Sensor duplicate name",
			content: `{
				"sensor": [{
					"name": "My sweat sensor",
					"topic": "tele/status",
					"interval": "13s",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				},{
					"name": "My sweat sensor",
					"topic": "tele/status",
					"interval": "13s",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid sensor (#1): sensor with this name already exists",
		},
		{
			name: "Trigger",
			content: `{
				"trigger": [{
					"name": "My sweat sensor",
					"topic": "tele/__DEVICE_ID__/status",
					"icon": "hassio-icon",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`, expectedResult: TopicConfigurations{
				Trigger: []Trigger{{
					Name:  "My sweat sensor",
					Topic: fmt.Sprintf("tele/%s/status", deviceId),
					Icon:  "hassio-icon",
					Command: Command{
						Name:      "/usr/bin/bash",
						Arguments: []string{"echo"},
					},
				}},
			},
		},
		{
			name: "Trigger missing name",
			content: `{
				"trigger": [{
					"topic": "tele/__DEVICE_ID__/status",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid trigger (#0): name must not be empty",
		},
		{
			name: "Trigger invalid topic",
			content: `{
				"trigger": [{
					"name": "My sweat sensor",
					"topic": "tele/+/status",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid trigger (#0): invalid topic: invalid character",
		},
		{
			name: "Trigger empty topic",
			content: `{
				"trigger": [{
					"name": "My sweat sensor",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid trigger (#0): invalid topic: must not be empty",
		},
		{
			name: "Trigger empty command",
			content: `{
				"trigger": [{
					"name": "My sweat sensor",
					"topic": "tele/status",
					"command": {
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid trigger (#0): command name must not be empty",
		},
		{
			name: "Trigger duplicate name",
			content: `{
				"trigger": [{
					"name": "My sweat sensor",
					"topic": "tele/status",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				},{
					"name": "My sweat sensor",
					"topic": "tele/status",
					"command": {
						"name": "/usr/bin/bash",
						"arguments": ["echo"]
					}
				}]
			}`,
			expectedError: "invalid config: invalid trigger (#1): trigger with this name already exists",
		},
	}

	for _, tt := range tests {
		t.Run("TestLoadTopicConfiguration_"+tt.name, func(t *testing.T) {
			tmpFile := testFile(tt.content)
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			configuration, err := LoadTopicConfiguration(tmpFile.Name(), deviceId)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, configuration)
			}

		})
	}
}

func testFile(content string) *os.File {
	file, err := ioutil.TempFile("", "TestLoadTopicConfiguration")
	if err != nil {
		panic(err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		panic(err)
	}

	return file
}

func interval(d time.Duration) *Interval {
	i := Interval(d)
	return &i
}
