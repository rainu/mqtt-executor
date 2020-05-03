package main

import (
	"context"
	"errors"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"os/exec"
	"sync"
	"time"
)

type CommandExecutor struct {
	lock           sync.RWMutex
	usedContext    map[context.Context]context.CancelFunc
	openExecutions sync.WaitGroup
}

var commandExecutor CommandExecutor

func init() {
	commandExecutor = CommandExecutor{
		lock:        sync.RWMutex{},
		usedContext: map[context.Context]context.CancelFunc{},
	}
}

func (c *CommandExecutor) ExecuteCommand(client MQTT.Client, message MQTT.Message, commandAndArgs []string) {
	ctx := c.createContext()
	c.openExecutions.Add(1)
	defer c.openExecutions.Done()

	command := exec.CommandContext(ctx, commandAndArgs[0], commandAndArgs[1:]...)
	out, execErr := command.CombinedOutput()
	if execErr != nil {
		zap.L().Error("Command execution failed.", zap.Error(execErr))
		client.Publish(fmt.Sprintf("%s/RESULT", message.Topic()), byte(*Config.PublishQOS), false, "<FAILED> "+execErr.Error())
		return
	}

	client.Publish(fmt.Sprintf("%s/RESULT", message.Topic()), byte(*Config.PublishQOS), false, out)
	c.releaseContext(ctx)
}

func (c *CommandExecutor) createContext() context.Context {
	c.lock.Lock()
	defer c.lock.Unlock()

	ctx, cancelFunc := context.WithCancel(context.Background())
	c.usedContext[ctx] = cancelFunc

	return ctx
}

func (c *CommandExecutor) releaseContext(ctx context.Context) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.usedContext, ctx)
}

func (c *CommandExecutor) Close(timeout time.Duration) error {
	wg := sync.WaitGroup{}

	c.lock.Lock()
	defer c.lock.Unlock()

	for ctx, cancelFunc := range c.usedContext {
		wg.Add(1)

		go func(c context.Context, cf context.CancelFunc) {
			defer wg.Done()
			cf()         //call cancel
			<-ctx.Done() //wait for cancellation
		}(ctx, cancelFunc)
	}

	wgChan := make(chan bool)
	go func() {
		wg.Wait()
		c.openExecutions.Wait()

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
