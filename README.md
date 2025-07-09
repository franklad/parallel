# Parallel Package

The `parallel` package provides a `Conductor` type for orchestrating multiple processes concurrently in Go, with support for graceful shutdown and error handling. It leverages Go's concurrency primitives and integrates with the `context` package for cancellation and the `zerolog` package for logging.

## Features
- Concurrent execution of multiple processes implementing the `Process` interface.
- Graceful shutdown handling for SIGINT and SIGTERM signals.
- Error monitoring and propagation for failed processes.
- Context-aware process management with timeout support during shutdown.

## Installation
To use the `parallel` package, ensure you have the `zerolog` dependency installed:

```bash
go get github.com/rs/zerolog
```

Then, import the package in your Go code:

```go
import "path/to/parallel"
```

## Usage
The `parallel` package revolves around the `Process` interface and the `Conductor` type. Below is an example of how to use it.

### Process Interface
The `Process` interface defines the contract for processes managed by the `Conductor`:

```go
type Process interface {
    Run(ctx context.Context) error
    Stop(ctx context.Context) error
    Name() string
}
```

You must implement this interface for any process you want to manage. Here's an example implementation:

```go
type MyProcess struct {
    name string
}

func (p *MyProcess) Run(ctx context.Context) error {
    log := zerolog.Ctx(ctx).With().Str("process", p.name).Logger()
    log.Info().Msg("running process")
    // Simulate work
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            time.Sleep(time.Second)
            log.Info().Msg("working...")
        }
    }
}

func (p *MyProcess) Stop(ctx context.Context) error {
    log := zerolog.Ctx(ctx).With().Str("process", p.name).Logger()
    log.Info().Msg("stopping process")
    return nil
}

func (p *MyProcess) Name() string {
    return p.name
}
```

### Using the Conductor
The `Conductor` manages a collection of processes, starts them concurrently, monitors for errors, and handles graceful shutdown. Here's an example:

```go
package main

import (
    "context"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "path/to/parallel"
    "time"
)

func main() {
    // Set up zerolog logger
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
    log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

    // Create context with logger
    ctx := log.WithContext(context.Background())

    // Create processes
    processes := []parallel.Process{
        &MyProcess{name: "process1"},
        &MyProcess{name: "process2"},
    }

    // Initialize conductor
    conductor := parallel.NewConductor(ctx, processes...)

    // Start processes and wait for stop signal
    conductor.Run(ctx).ThenStop()

    // Optionally, check for errors
    for err := range conductor.Errors() {
        log.Error().
            Str("process", err.process.Name()).
            Err(err.err).
            Msg("encountered error")
    }
}
```

### How It Works
1. **Initialization**: Create a `Conductor` with `NewConductor`, passing a context and a list of processes. The context should include a `zerolog.Logger` for logging.
2. **Running Processes**: Call `Run` to start all processes concurrently. Each process runs in its own goroutine.
3. **Error Handling**: If a process returns an error from its `Run` method, the `Conductor` captures it and sends a stop signal to trigger a graceful shutdown.
4. **Graceful Shutdown**: When a SIGINT or SIGTERM signal is received (or an error occurs), `ThenStop` stops all processes with a 5-second timeout, ensuring each process's `Stop` method is called.
5. **Error Retrieval**: Use the `Errors` method to retrieve a channel of errors from failed processes.

### Key Methods
- `NewConductor(ctx context.Context, processes ...Process) *Conductor`: Creates a new `Conductor` instance.
- `Run(ctx context.Context) *Conductor`: Starts all processes concurrently and returns the `Conductor` for method chaining.
- `ThenStop()`: Waits for a stop signal or error, then gracefully stops all processes.
- `Errors() <-chan processError`: Returns a channel to receive errors from failed processes.

## Example Output
Running the above example might produce logs like:

```
2025-07-08T23:54:00Z INF conductor initializing conductor engine
2025-07-08T23:54:00Z INF conductor starting process process=process1
2025-07-08T23:54:00Z INF conductor starting process process=process2
2025-07-08T23:54:00Z INF process1 running process
2025-07-08T23:54:00Z INF process2 running process
2025-07-08T23:54:01Z INF process1 working...
2025-07-08T23:54:01Z INF process2 working...
^C
2025-07-08T23:54:02Z INF conductor received stop signal, stopping all processes
2025-07-08T23:54:02Z INF process1 stopping process
2025-07-08T23:54:02Z INF process1 stopped process
2025-07-08T23:54:02Z INF process2 stopping process
2025-07-08T23:54:02Z INF process2 stopped process
```

## Notes
- Ensure the context passed to `NewConductor` and `Run` includes a `zerolog.Logger` to avoid nil logger errors.
- Processes should respect the context's cancellation in their `Run` and `Stop` methods to ensure clean shutdowns.
- The `Conductor` uses a 5-second timeout for stopping processes during shutdown. Adjust this in the `ThenStop` method if needed.
- The `Errors` channel has a buffer size equal to the number of processes to prevent blocking.

## License
This package is provided under the MIT License.