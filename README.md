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
- **Router** - Directs water to different paths
- **Leak** - Pipeline errors

## 🏗️ Architecture

### Core Interfaces

```go
type Water interface{} // Data flowing through pipes

type Pipe interface {
    Process(ctx context.Context, water Water) (Water, *Leak)
    Connect(downstream Pipe) error
    Name() string
}

type Spigot interface {
    Flow(ctx context.Context) (<-chan Water, <-chan *Leak)
    Name() string
}

type Tap interface {
    Drain(ctx context.Context, water Water) *Leak
    Name() string
}
```

### Pipeline Execution

- **Serial Processing**: Connected pipes process data sequentially
- **Concurrent Processing**: Pipes after a tee run concurrently
- **Error Handling**: Leaks (errors) are propagated through a dedicated channel
- **Graceful Shutdown**: Pipelines can be stopped cleanly

## 🚀 Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/brunorene/go-plumber/pkg/plumber"
)

func main() {
    // Create a pipeline: numbers → double → add prefix → console
    pipeline, err := plumber.NewBuilder("example").
        AddSpigot(createNumberSpigot()).
        AddPipe(createDoublePipe()).
        AddPipe(createPrefixPipe()).
        AddTap(createConsoleTap()).
        Build()

    if err != nil {
        panic(err)
    }

    // Start processing
    pipeline.Start()
    defer pipeline.Stop()

    // Monitor for leaks
    go func() {
        for leak := range pipeline.Leaks() {
            fmt.Printf("LEAK: %v\n", leak.Err)
        }
    }()

    time.Sleep(5 * time.Second)
}
```

## 📁 Project Structure

```
github.com/brunorene/go-plumber/
├── pkg/plumber/           # Core framework
│   ├── types.go          # Interface definitions
│   ├── builder.go        # Pipeline builder
│   └── basic.go          # Basic implementations
├── examples/
│   └── basic/            # Basic usage example
└── main.go               # Simple CLI entry point
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

- [ ] Advanced tee types (reducer, router, etc.)
- [ ] Valve implementations (rate limiting, backpressure)
- [ ] Meter implementations (metrics collection)
- [ ] Strainer implementations (filtering)
- [ ] Configuration-based pipeline assembly
- [ ] Performance optimizations
- [ ] Comprehensive test suite
- [ ] Documentation and examples
Framework to compose workers into a pipeline
