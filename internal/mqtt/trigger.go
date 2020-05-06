package mqtt

import (
	"context"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-executor/internal/cmd"
	"github.com/rainu/mqtt-executor/internal/mqtt/config"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

const (
	TopicSuffixState     = "STATE"
	TopicSuffixResult    = "RESULT"
	PayloadStatusRunning = "RUNNING"
	PayloadStatusStopped = "STOPPED"
	PayloadStart         = "START"
	PayloadStop          = "STOP"
)

type Trigger struct {
	lock            sync.RWMutex
	runningCommands map[string]context.CancelFunc
	triggerConfigs  []config.Trigger
	publishQOS      byte

	Executor   *cmd.CommandExecutor
	MqttClient MQTT.Client
}

func (t *Trigger) Initialise(subscribeQOS, publishQOS byte, triggerConfigs []config.Trigger) {
	t.publishQOS = publishQOS
	t.runningCommands = map[string]context.CancelFunc{}

	for _, triggerConf := range triggerConfigs {
		t.triggerConfigs = append(t.triggerConfigs, triggerConf)
		t.MqttClient.Subscribe(triggerConf.Topic, subscribeQOS, t.createTriggerHandler(triggerConf))
		t.publishStatus(triggerConf.Topic, PayloadStatusStopped)
	}
}

func (t *Trigger) createTriggerHandler(triggerConfig config.Trigger) MQTT.MessageHandler {
	return func(client MQTT.Client, message MQTT.Message) {
		zap.L().Info("Incoming message: ",
			zap.String("topic", message.Topic()),
			zap.ByteString("payload", message.Payload()),
		)

		action := strings.ToUpper(string(message.Payload()))

		switch action {
		case PayloadStart:
			if t.isCommandRunning(triggerConfig) {
				zap.L().Warn("Command is already running. Skip execution!", zap.String("trigger", triggerConfig.Name))
				return
			}

			go t.executeCommand(message.Topic(), triggerConfig)
		case PayloadStop:
			if !t.isCommandRunning(triggerConfig) {
				return
			}
			t.interruptCommand(triggerConfig)
			t.unregisterCommand(triggerConfig)
		default:
			zap.L().Warn("Invalid payload")
		}
	}
}

func (t *Trigger) isCommandRunning(trigger config.Trigger) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	_, exist := t.runningCommands[trigger.Name]
	return exist
}

func (t *Trigger) registerCommand(trigger config.Trigger) context.Context {
	t.lock.Lock()
	defer t.lock.Unlock()

	ctx, cancelFunc := context.WithCancel(context.Background())
	t.runningCommands[trigger.Name] = cancelFunc

	return ctx
}

func (t *Trigger) unregisterCommand(trigger config.Trigger) {
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.runningCommands, trigger.Name)
}

func (t *Trigger) interruptCommand(trigger config.Trigger) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	//execute corresponding cancel func
	t.runningCommands[trigger.Name]()
}

func (t *Trigger) executeCommand(topic string, trigger config.Trigger) {
	ctx := t.registerCommand(trigger)
	defer t.unregisterCommand(trigger)

	t.publishStatus(topic, PayloadStatusRunning)
	defer t.publishStatus(topic, PayloadStatusStopped)

	output, execErr := t.Executor.ExecuteCommandWithContext(trigger.Command.Name, trigger.Command.Arguments, ctx)
	if execErr != nil {
		if execErr == context.Canceled {
			t.publishResult(topic, "<INTERRUPTED>")
		} else {
			t.publishResult(topic, "<FAILED>;"+execErr.Error())
		}
		return
	}

	t.publishResult(topic, output)
}

func (t *Trigger) publishStatus(parentTopic, status string) MQTT.Token {
	stateTopic := t.buildStateTopic(parentTopic)
	return t.MqttClient.Publish(stateTopic, t.publishQOS, false, status)
}

func (t *Trigger) publishResult(parentTopic string, result interface{}) MQTT.Token {
	resultTopic := t.buildResultTopic(parentTopic)
	return t.MqttClient.Publish(resultTopic, t.publishQOS, false, result)
}

func (t *Trigger) buildStateTopic(parentTopic string) string {
	return fmt.Sprintf("%s/%s", parentTopic, TopicSuffixState)
}

func (t *Trigger) buildResultTopic(parentTopic string) string {
	return fmt.Sprintf("%s/%s", parentTopic, TopicSuffixResult)
}

func (t *Trigger) Close(timeout time.Duration) error {
	for _, triggerConf := range t.triggerConfigs {
		t.MqttClient.Unsubscribe(triggerConf.Topic)
	}

	return nil
}
