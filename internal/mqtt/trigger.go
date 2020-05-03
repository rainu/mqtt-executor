package mqtt

import (
	"context"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-executor/internal/cmd"
	"github.com/rainu/mqtt-executor/internal/mqtt/config"
	"go.uber.org/zap"
	"time"
)

const (
	TopicSuffixState     = "STATE"
	TopicSuffixResult    = "RESULT"
	PayloadStatusRunning = "RUNNING"
	PayloadStatusStopped = "STOPPED"
	PayloadStop          = "STOP"
)

type Trigger struct {
	triggerConfigs []config.Trigger

	Executor   *cmd.CommandExecutor
	MqttClient MQTT.Client
}

func (t *Trigger) Initialise(subscribeQOS, publishQOS byte, triggerConfigs []config.Trigger) {

	for _, triggerConf := range triggerConfigs {
		t.triggerConfigs = append(t.triggerConfigs, triggerConf)
		t.MqttClient.Subscribe(triggerConf.Topic, subscribeQOS, t.createTriggerHandler(publishQOS, triggerConf))
	}

	//TODO: publish stopped state for each command (inital state)
}

func (t *Trigger) createTriggerHandler(publishQOS byte, triggerConfig config.Trigger) MQTT.MessageHandler {
	return func(client MQTT.Client, message MQTT.Message) {
		zap.L().Info("Incoming message: ",
			zap.String("topic", message.Topic()),
			zap.ByteString("payload", message.Payload()),
		)

		action, exists := triggerConfig.Actions[string(message.Payload())]
		if !exists {
			zap.L().Warn("Command is not configured")
			return
		}
		cmd := action.Command

		go t.executeCommand(publishQOS, message.Topic(), string(message.Payload()), cmd)
	}
}

func (t *Trigger) executeCommand(publishQOS byte, topic, action string, command config.Command) {
	stateTopic := fmt.Sprintf("%s/%s/%s", topic, action, TopicSuffixState)
	resultTopic := fmt.Sprintf("%s/%s/%s", topic, action, TopicSuffixResult)

	t.MqttClient.Publish(stateTopic, publishQOS, false, PayloadStatusRunning)
	defer t.MqttClient.Publish(stateTopic, publishQOS, false, PayloadStatusStopped)

	output, execErr := t.Executor.ExecuteCommandWithContext(command.Name, command.Arguments, context.Background())
	if execErr != nil {
		t.MqttClient.Publish(resultTopic, publishQOS, false, "<FAILED> "+execErr.Error())
		return
	}

	t.MqttClient.Publish(resultTopic, publishQOS, false, output)
}

func (t *Trigger) Close(timeout time.Duration) error {
	for _, triggerConf := range t.triggerConfigs {
		t.MqttClient.Unsubscribe(triggerConf.Topic)
	}

	return nil
}
