package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/franklad/parallel"
)

type Process struct {
	done chan struct{}
}

func NewProcess() *Process {
	return &Process{
		done: make(chan struct{}),
	}
}

func (p *Process) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return nil
	case <-time.After(2 * time.Second):
		fmt.Println("Simulate work duration")
		return errors.New("simulated error after work duration")
	}
}

func (p *Process) Stop(ctx context.Context) error {
	close(p.done)
	return nil
}

func (p *Process) Name() string {
	return "ExampleProcess"
}

func main() {
	ctx := context.Background()
	conductor := parallel.NewConductor(NewProcess())
	conductor.Run(ctx).ThenStop()
}
