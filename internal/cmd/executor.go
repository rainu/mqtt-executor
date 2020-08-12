package cmd

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"os/exec"
	"strings"
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
	//register the context so that we have a chance to cancel the commands later
	ctx := c.registerContext(executionContext)
	c.openExecutions.Add(1)
	defer c.openExecutions.Done()
	defer c.releaseContext(ctx)

	command := exec.CommandContext(ctx, cmd, args...)
	out, execErr := command.CombinedOutput()

	if len(out) > 0 {
		//trim combined output
		out = []byte(strings.Trim(string(out), " \n"))
	}

	if execErr != nil && ctx.Err() == nil {
		zap.L().Error("Command execution failed.", zap.Error(execErr))
	} else if ctx.Err() != nil {
		zap.L().Info("Command execution cancelled.")
		return out, ctx.Err()
	}

	return out, execErr
}

func (c *CommandExecutor) registerContext(parentContext context.Context) context.Context {
	//lock to ensure the map is thread-safe
	c.lock.Lock()
	defer c.lock.Unlock()

	//wrap the given context so that we can later cancel the context (see Close func)
	ctx, cancelFunc := context.WithCancel(parentContext)
	c.usedContext[ctx] = cancelFunc

	return ctx
}

func (c *CommandExecutor) releaseContext(ctx context.Context) {
	//lock to ensure the map is thread-safe
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.usedContext, ctx)
}

func (c *CommandExecutor) Close(timeout time.Duration) error {
	wg := sync.WaitGroup{}

	//this lock ensures that no new commands can start
	c.lock.Lock()
	defer c.lock.Unlock()

	//cancel all in parallel
	for ctx, cancelFunc := range c.usedContext {
		wg.Add(1)

		//trigger cancel-func and wait for it
		go func(c context.Context, cf context.CancelFunc) {
			defer wg.Done()
			cf()       //call cancel
			<-c.Done() //wait for cancellation
		}(ctx, cancelFunc)
	}

	//wait for all commands to be stopped
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
