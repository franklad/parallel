package parallel

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

type Process interface {
	Run(ctx context.Context) error
	Stop()
	Name() string
}

type processError struct {
	process Process
	err     error
}

type Conductor struct {
	log       *zap.Logger
	stop      chan os.Signal
	errors    chan processError
	processes []Process
}

func NewConductor(processes ...Process) *Conductor {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	r := &Conductor{
		log:       logger,
		stop:      make(chan os.Signal),
		errors:    make(chan processError, len(processes)),
		processes: processes,
	}

	signal.Notify(r.stop, syscall.SIGINT, syscall.SIGTERM)

	return r
}

func (c *Conductor) Run(ctx context.Context) *Conductor {
	go c.monitor(ctx)

	for _, p := range c.processes {
		go func(process Process) {
			c.log.Info("running", zap.String("process", process.Name()))

			if err := process.Run(ctx); err != nil {
				c.errors <- processError{
					process: process,
					err:     err,
				}

				return
			}
		}(p)
	}

	return c
}

func (c *Conductor) ThenStop() {
	<-c.stop
	c.log.Info("received stop signal, stopping all processes")

	var wg sync.WaitGroup
	for _, p := range c.processes {
		wg.Add(1)
		go func(process Process) {
			defer wg.Done()

			process.Stop()
			c.log.Info("process stopped", zap.String("process", process.Name()))
		}(p)
	}
	wg.Wait()

	signal.Stop(c.stop)
	close(c.stop)
	close(c.errors)
}

func (c *Conductor) Errors() <-chan processError {
	return c.errors
}

func (c *Conductor) monitor(ctx context.Context) {
	select {
	case err := <-c.errors:
		c.log.Error("error occurred", zap.String("process", err.process.Name()), zap.Error(err.err))
		c.stop <- syscall.SIGTERM
	case <-ctx.Done():
		c.log.Warn("context canceled")
		c.stop <- syscall.SIGTERM
	}
}
