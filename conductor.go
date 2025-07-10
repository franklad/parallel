package parallel

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

type Process interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
}

type processError struct {
	process Process
	err     error
}

type Conductor struct {
	log       zerolog.Logger
	stop      chan os.Signal
	errors    chan processError
	processes []Process
}

func NewConductor(processes ...Process) *Conductor {
	log := zerolog.New(os.Stdout).With().Str("engine", "conductor").Logger()
	log.Info().Msg("initializing conductor engine")

	r := &Conductor{
		log:       log,
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
			c.log.Info().
				Str("process", process.Name()).
				Msg("starting process")

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
	c.log.Warn().Msg("received stop signal, stopping all processes")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, p := range c.processes {
		wg.Add(1)
		go func(process Process) {
			defer wg.Done()

			if err := process.Stop(ctx); err != nil {
				c.log.Error().
					Str("process", process.Name()).
					Err(err).
					Msg("failed to stop process")
			} else {
				c.log.Info().
					Str("process", process.Name()).
					Msg("stopped process")
			}
		}(p)
	}

	wg.Wait()
	signal.Stop(c.stop)
}

func (c *Conductor) Errors() <-chan processError {
	return c.errors
}

func (c *Conductor) monitor(ctx context.Context) {
	select {
	case err := <-c.errors:
		if err.err != nil {
			c.log.Error().
				Str("process", err.process.Name()).
				Err(err.err).
				Msg("process error")
		}

		c.stop <- syscall.SIGTERM
		return
	case <-ctx.Done():
		c.log.Warn().Msg("context cancelled")

		c.stop <- syscall.SIGTERM
		return
	}
}
