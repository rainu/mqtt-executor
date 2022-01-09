package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-executor/internal/cmd"
	"github.com/rainu/mqtt-executor/internal/mqtt"
	"github.com/rainu/mqtt-executor/internal/mqtt/hassio"
	"go.uber.org/zap"
)

var commandExecutor *cmd.CommandExecutor
var statusWorker mqtt.StatusWorker
var sensorWorker mqtt.SensorWorker
var trigger mqtt.Trigger

func main() {
	LoadConfig()
	commandExecutor = cmd.NewCommandExecutor()
	trigger.Executor = commandExecutor
	sensorWorker.Executor = commandExecutor

	//reacting to signals (interrupt)
	signals := make(chan os.Signal, 1)
	defer close(signals)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	client := MQTT.NewClient(Config.GetMQTTOpts(
		handleOnConnection,
		handleOnConnectionLost,
	))
	statusWorker.MqttClient = client
	trigger.MqttClient = client
	sensorWorker.MqttClient = client

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		zap.L().Fatal("Error while connecting to mqtt broker: %s", zap.Error(token.Error()))
	}

	//if hassio is enabled -> publish the hassio mqtt-discovery configs
	if *Config.HomeassistantEnable {
		haClient := hassio.Client{
			DeviceName:  *Config.DeviceName,
			DeviceId:    *Config.DeviceId,
			TopicPrefix: *Config.HomeassistantTopic,
			MqttClient:  client,
		}
		haClient.PublishDiscoveryConfig(Config.TopicConfigurations)
	}

	if Config.TopicConfigurations.Availability != nil {
		statusWorker.Initialise(*Config.TopicConfigurations.Availability)
	}

	//register trigger and sensors
	trigger.Initialise(byte(*Config.SubscribeQOS), byte(*Config.PublishQOS), Config.TopicConfigurations.Trigger)
	sensorWorker.Initialise(byte(*Config.PublishQOS), Config.TopicConfigurations.Sensors())

	// wait for interrupt
	<-signals

	shutdown(client)
}

var handleOnConnection = func(client MQTT.Client) {
	if !trigger.IsInitialised() {
		return
	}

	zap.L().Info("Reinitialise...")
	trigger.ReInitialise()
}

var handleOnConnectionLost = func(client MQTT.Client, err error) {
	zap.L().Warn("Connection lost to broker.", zap.Error(err))
}

func shutdown(client MQTT.Client) {
	zap.L().Info("Shutting down...")

	type closable interface {
		Close(time.Duration) error
	}
	closeables := []closable{&statusWorker, &sensorWorker, &trigger, commandExecutor}

	//most operating systems wait a maximum of 30 seconds

	wg := sync.WaitGroup{}
	wg.Add(len(closeables))
	timeout := 20 * time.Second

	for _, c := range closeables {
		go func(c closable) {
			defer wg.Done()

			if err := c.Close(timeout); err != nil {
				zap.L().Error("Timeout while waiting for graceful shutdown!", zap.Error(err))
			}
		}(c)
	}
	wg.Wait()

	//we have to disconnect at last because one closeable unsubscripe all topics
	client.Disconnect(10 * 1000) //wait 10sek at most
}
