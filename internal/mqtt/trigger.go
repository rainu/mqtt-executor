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
	t.triggerConfigs = triggerConfigs //safe the configs so that we can unsubscribe later (see Close func)

	for _, triggerConf := range triggerConfigs {
		t.MqttClient.Subscribe(triggerConf.Topic, subscribeQOS, t.createTriggerHandler(triggerConf))

		//publish the stopped state on startup
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
			//ensure that only one trigger runs at the same time
			if t.isCommandRunning(triggerConfig) {
				zap.L().Warn("Command is already running. Skip execution!", zap.String("trigger", triggerConfig.Name))
				return
			}

			go t.executeCommand(message.Topic(), triggerConfig)
		case PayloadStop:
			if !t.isCommandRunning(triggerConfig) {
				//no command running -> no action
				return
			}
			t.interruptCommand(triggerConfig)
			t.unregisterCommand(triggerConfig)
		default:
			zap.L().Warn("Invalid payload. Do nothing.")
		}
	}
}

func (t *Trigger) isCommandRunning(trigger config.Trigger) bool {
	//we only need read access
	t.lock.RLock()
	defer t.lock.RUnlock()

	_, exist := t.runningCommands[trigger.Name]
	return exist
}

func (t *Trigger) registerCommand(trigger config.Trigger) context.Context {
	//we need write access
	t.lock.Lock()
	defer t.lock.Unlock()

	ctx, cancelFunc := context.WithCancel(context.Background())
	t.runningCommands[trigger.Name] = cancelFunc

	return ctx
}

func (t *Trigger) unregisterCommand(trigger config.Trigger) {
	//we need write access
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.runningCommands, trigger.Name)
}

func (t *Trigger) interruptCommand(trigger config.Trigger) {
	//we only need read access
	t.lock.RLock()
	defer t.lock.RUnlock()

	//execute corresponding cancel func
	t.runningCommands[trigger.Name]()
}

func (t *Trigger) executeCommand(topic string, trigger config.Trigger) {
	ctx := t.registerCommand(trigger)  //register at begin
	defer t.unregisterCommand(trigger) //unregister at end

	t.publishStatus(topic, PayloadStatusRunning)       //publish that we are now running
	defer t.publishStatus(topic, PayloadStatusStopped) //at the end we are stopped

	output, execErr := t.Executor.ExecuteCommandWithContext(trigger.Command.Name, trigger.Command.Arguments, ctx)
	if execErr != nil {
		if execErr == context.Canceled {
			//this can happen if a STOPPED-Message was incoming or the application is shutting down
			t.publishResult(topic, "<INTERRUPTED>")
		} else {
			//program execution failed (status code != 0)
			t.publishResult(topic, "<FAILED>;"+execErr.Error())
		}
		return
	}

	//publish the program's output (stdout & stderr)
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
	//unsubscribe to all mqtt-topics (ignore the timeout!)
	for _, triggerConf := range t.triggerConfigs {
		t.MqttClient.Unsubscribe(triggerConf.Topic)
	}

	return nil
}
