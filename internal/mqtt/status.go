package mqtt

import (
	"context"
	"errors"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rainu/mqtt-executor/internal/mqtt/config"
	"sync"
	"time"
)

type StatusWorker struct {
	waitGroup  sync.WaitGroup
	cancelFunc context.CancelFunc

	MqttClient MQTT.Client
}

func (s *StatusWorker) Initialise(availabilityConfigs config.Availability) {

	var ctx context.Context
	ctx, s.cancelFunc = context.WithCancel(context.Background())

	s.waitGroup.Add(1)
	go s.runStatus(ctx, availabilityConfigs)
}

func (s *StatusWorker) runStatus(ctx context.Context, availabilityConfig config.Availability) {
	defer s.waitGroup.Done()
	defer func() {
		token := s.MqttClient.Publish(availabilityConfig.Topic, byte(0), false, availabilityConfig.Payload.Unavailable)
		token.Wait()
	}()

	//first one
	s.MqttClient.Publish(availabilityConfig.Topic, byte(0), false, availabilityConfig.Payload.Available)

	ticker := time.Tick(time.Duration(*availabilityConfig.Interval))
	for {
		select {
		case <-ticker:
			s.MqttClient.Publish(availabilityConfig.Topic, byte(0), false, availabilityConfig.Payload.Available)
		case <-ctx.Done():
			return
		}
	}
}

func (s *StatusWorker) Close(timeout time.Duration) error {
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
