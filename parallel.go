package parallel

import (
	"context"
	"os"
	"os/signal"
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

	signal.Notify(r.stop, os.Interrupt, syscall.SIGTERM)

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

			c.log.Info("successfully ran", zap.String("process", process.Name()))
		}(p)
	}

	return c
}

func (c *Conductor) ThenStop() {
	<-c.stop
	c.log.Info("received stop signal, stopping all processes")

	for _, process := range c.processes {
		process.Stop()
		c.log.Info("stopped", zap.String("process", process.Name()))
	}

	close(c.stop)
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
