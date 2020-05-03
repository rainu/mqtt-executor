package mqtt

import (
	"context"
	"errors"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-executor/internal/cmd"
	"github.com/rainu/mqtt-executor/internal/mqtt/config"
	"sync"
	"time"
)

type SensorWorker struct {
	waitGroup  sync.WaitGroup
	cancelFunc context.CancelFunc

	Executor   *cmd.CommandExecutor
	MqttClient MQTT.Client
}

func (s *SensorWorker) Initialise(publishQOS byte, sensorConfigs []config.Sensor) {

	var ctx context.Context
	ctx, s.cancelFunc = context.WithCancel(context.Background())

	for _, sensorConf := range sensorConfigs {
		s.waitGroup.Add(1)
		go s.runSensor(ctx, publishQOS, sensorConf)
	}
}

func (s *SensorWorker) runSensor(ctx context.Context, publishQOS byte, sensorConf config.Sensor) {
	defer s.waitGroup.Done()

	//first one
	s.executeCommand(publishQOS, sensorConf)

	ticker := time.Tick(time.Duration(sensorConf.Interval))
	for {
		select {
		case <-ticker:
			s.executeCommand(publishQOS, sensorConf)
		case <-ctx.Done():
			return
		}
	}
}

func (s *SensorWorker) executeCommand(publishQOS byte, sensorConf config.Sensor) {
	output, execErr := s.Executor.ExecuteCommand(sensorConf.Command.Name, sensorConf.Command.Arguments)
	if execErr != nil {
		s.MqttClient.Publish(sensorConf.ResultTopic, publishQOS, false, "<FAILED> "+execErr.Error())
		return
	}

	s.MqttClient.Publish(sensorConf.ResultTopic, publishQOS, false, output)
}

func (s *SensorWorker) Close(timeout time.Duration) error {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	wgChan := make(chan bool)
	go func() {
		s.waitGroup.Wait()
		wgChan <- true
	}()

	//wait for WaitGroup or Timeout
	select {
	case <-wgChan:
		return nil
	case <-time.After(timeout):
		return errors.New("timeout exceeded")
	}
}
