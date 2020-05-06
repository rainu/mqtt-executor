package cmd

import (
	"context"
	"errors"
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

func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		lock:        sync.RWMutex{},
		usedContext: map[context.Context]context.CancelFunc{},
	}
}

func (c *CommandExecutor) ExecuteCommand(cmd string, args []string) ([]byte, error) {
	return c.ExecuteCommandWithContext(cmd, args, context.Background())
}

func (c *CommandExecutor) ExecuteCommandWithContext(cmd string, args []string, executionContext context.Context) ([]byte, error) {
	ctx := c.registerContext(executionContext)
	c.openExecutions.Add(1)
	defer c.openExecutions.Done()
	defer c.releaseContext(ctx)

	command := exec.CommandContext(ctx, cmd, args...)
	out, execErr := command.CombinedOutput()

	if execErr != nil {
		zap.L().Error("Command execution failed.", zap.Error(execErr))
	}

	return out, execErr
}

func (c *CommandExecutor) registerContext(parentContext context.Context) context.Context {
	c.lock.Lock()
	defer c.lock.Unlock()

	ctx, cancelFunc := context.WithCancel(parentContext)
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