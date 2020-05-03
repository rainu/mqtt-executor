package main

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	LoadConfig()

	signals := make(chan os.Signal, 1)
	defer close(signals)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	client := MQTT.NewClient(Config.GetMQTTOpts())

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		zap.L().Fatal("Error while connecting to mqtt broker: %s", zap.Error(token.Error()))
	}

	for topicName, config := range Config.TopicConfigurations {
		client.Subscribe(topicName, byte(*Config.SubscribeQOS), createMessageHandler(config))
	}

	// wait for interrupt
	<-signals

	shutdown(client)
}

func createMessageHandler(topicConfig TopicConfiguration) MQTT.MessageHandler {
	return func(client MQTT.Client, message MQTT.Message) {
		zap.L().Info("Incoming message: ",
			zap.String("topic", message.Topic()),
			zap.ByteString("payload", message.Payload()),
		)

		cmdArgs, exists := topicConfig[string(message.Payload())]
		if !exists {
			zap.L().Warn("Command is not configured")
			return
		}

		go commandExecutor.ExecuteCommand(client, message, cmdArgs)
	}
}

func shutdown(client MQTT.Client) {
	zap.L().Info("Shutting down...")
	for topicName, _ := range Config.TopicConfigurations {
		client.Unsubscribe(topicName)
	}

	err := commandExecutor.Close(20 * time.Second)
	if err != nil {
		zap.L().Error("Timeout while waiting command execution!", zap.Error(err))
	}

	client.Disconnect(20 * 1000) //wait 10sek at most
}
