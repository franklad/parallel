package parallel

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Process is an interface for components that can be started and stopped.
type Process interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
}

// Conductor manages the lifecycle of multiple Process components.
type Conductor struct {
	log       *zap.Logger
	stop      chan os.Signal
	processes []Process
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewConductor initializes a new Conductor with provided processes and sets up signal handling.
func NewConductor(ctx context.Context, processes ...Process) *Conductor {
	ctx, cancel := context.WithCancel(ctx)
	r := &Conductor{
		log:       zap.NewExample(),
		stop:      make(chan os.Signal, 1),
		processes: processes,
		ctx:       ctx,
		cancel:    cancel,
	}

	signal.Notify(r.stop, os.Interrupt, syscall.SIGTERM)

	return r
}

// Run starts all processes in separate goroutines.
func (r *Conductor) Run() *Conductor {
	for _, process := range r.processes {
		go func(p Process) {
			r.log.Info("running", zap.String("process", p.Name()))
			if err := p.Run(r.ctx); err != nil {
				r.log.Panic("failed to run", zap.String("process", p.Name()), zap.Error(err))
			}

			r.log.Info("ran", zap.String("process", p.Name()))
		}(process)
	}

	return r
}

// Stop stops all processes gracefully within a 5-second timeout.
func (r *Conductor) Stop() {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	for _, process := range r.processes {
		r.log.Info("stopping", zap.String("process", process.Name()))
		if err := process.Stop(ctx); err != nil {
			r.log.Panic("failed to stop", zap.String("process", process.Name()), zap.Error(err))
		}

		r.log.Info("stopped", zap.String("process", process.Name()))
	}

	r.cancel()
}

// ThenStop waits for an OS signal and then stops all processes.
func (r *Conductor) ThenStop() {
	<-r.stop
	r.Stop()
}
