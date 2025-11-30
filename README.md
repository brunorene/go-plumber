# 🔧 Go Plumber

A pipeline framework for Go using plumbing metaphors to create intuitive data processing workflows.

## 🚰 Concept

Go Plumber uses familiar plumbing terminology to describe pipeline components:

- **Water** - Data flowing through the system
- **Spigot** - Data source (where water comes from)
- **Pipe** - Data processor (transforms water)
- **Tap** - Data sink (where water goes out)
- **Tee** - Splits or joins water flow (fan-out/fan-in)
- **Valve** - Controls flow rate and backpressure
- **Meter** - Monitors pipeline performance
- **Strainer** - Filters data based on conditions
- **Leak** - Pipeline errors

## 🏗️ Architecture

### Core streaming abstractions

```go
type Named interface {
    Name() string
}

type Streamer[In, Out any] interface {
    Named
    Stream(
        ctx context.Context,
        in <-chan In,
    ) (out <-chan Out, leaks <-chan error)
}

// 1→1 stages
type Pipe[In, Out any] interface{ Streamer[In, Out] }
type Spigot[Out any] interface{ Pipe[struct{}, Out] }
type Tap[In any] interface{ Pipe[In, struct{}] }

// 1→N and N→1 stages
type SplitTee[T any] interface {
    Named
    Stream(ctx context.Context, in <-chan T) (outs []<-chan T, leaks <-chan error)
}

type JoinTee[In, Out any] interface {
    Named
    Stream(ctx context.Context, ins []<-chan In) (out <-chan Out, leaks <-chan error)
}

// Specialized 1→1 stages
type Valve[T any] interface{ Pipe[T, T] }
type Meter[T any] interface{ Pipe[T, T] }
type Strainer[T any] interface{ Pipe[T, T] }
```

### Execution model

- **Backpressure**: channels and context cancellation naturally propagate slowdown upstream.
- **Concurrency**: each stage runs in its own goroutine; SplitTee and JoinTee let you build fan-out/fan-in graphs.
- **Error handling**: every stage returns a `leaks <-chan error` stream for stage-level failures.
- **Graceful shutdown**: everything observes `ctx.Done()` and closes its output channels.

## 🚀 Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    plumber "github.com/brunorene/go-plumber/plumber"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Spigot: generate incrementing integers.
    counter := 0
    spigot := plumber.NewSpigot[int](
        "numbers",
        func(context.Context) (int, error) {
            counter++

            return counter, nil
        },
        1*time.Second,
    )

    // Pipe: double each value.
    doubler := plumber.NewPipe[int, int](
        "doubler",
        func(_ context.Context, value int) (int, error) {
            return value * 2, nil
        },
    )

    // Tap: log the final output.
    tap := plumber.NewTap[int](
        "logger",
        func(_ context.Context, value int) error {
            log.Printf("output: %d", value)

            return nil
        },
    )

    // Wire the graph
    spigotOut, spigotLeaks := spigot.Stream(ctx, nil)
    doubledOut, doublerLeaks := doubler.Stream(ctx, spigotOut)
    done, tapLeaks := tap.Stream(ctx, doubledOut)

    // Drain leaks
    go func() {
        for _, leaks := range []<-chan error{spigotLeaks, doublerLeaks, tapLeaks} {
            for err := range leaks {
                log.Printf("LEAK: %v", err)
            }
        }
    }()

    time.Sleep(5 * time.Second)
    cancel()
    <-done
}
```

## 📁 Project Structure

```
github.com/brunorene/go-plumber/
├── plumber/               # Core library package
│   ├── types.go           # Interfaces and generic abstractions
│   ├── basic.go           # Basic implementations (Pipe, Spigot, Tap, Tees, Valve, Meter, Strainer)
│   ├── *_test.go          # Unit tests
├── cmd/go-plumber/        # CLI entry point
├── examples/
│   ├── basic/             # Basic Pipe/Spigot/Tap example
│   ├── splittee/          # SplitTee fan-out example
│   └── jointee/           # JoinTee zip/join example
├── magefile.go            # Build, test, lint, and dev tasks
└── go.mod                 # Module definition
```

## 🛠️ Development

### Build & Test
```bash
# Build everything
mage build

# Run tests
mage test

# Format code
mage format

# Run linters
mage lint

# Pre-commit checks
mage preCommit
```

### Example Usage
```bash
# Run basic example
go run examples/basic/main.go

# Build and run main CLI
mage run
```

## 🔮 Roadmap

- [ ] Advanced tee types (reducers, windowing, etc.)
- [x] Valve implementations (rate limiting / throttling)
- [x] Meter implementations (metrics hooks)
- [x] Strainer implementations (filtering)
- [ ] Configuration-based pipeline assembly
- [ ] Performance optimizations
- [ ] More examples and documentation
