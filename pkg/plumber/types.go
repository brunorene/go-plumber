// Package plumber provides a pipeline framework using plumbing metaphors.
package plumber

import (
	"context"
	"time"
)

// Water represents data flowing through the pipeline.
type Water interface{}

// Leak represents an error in the pipeline.
type Leak struct {
	Err     error
	Source  string
	Water   Water
	Time    time.Time
	Context map[string]interface{}
}

// Pipe processes water (data) and can be connected to other pipes.
type Pipe interface {
	Process(ctx context.Context, water Water) (Water, *Leak)
	Connect(downstream Pipe) error
	Name() string
}

// Spigot is a data source that produces water.
type Spigot interface {
	Flow(ctx context.Context) (<-chan Water, <-chan *Leak)
	Name() string
}

// Tap is a data sink that consumes water.
type Tap interface {
	Drain(ctx context.Context, water Water) *Leak
	Name() string
}

// Tee splits or joins water flow.
type Tee interface {
	Name() string
}

// SplitTee fans out water to multiple downstream pipes.
type SplitTee interface {
	Tee
	Split(ctx context.Context, water Water) []Water
	ConnectOutputs(pipes ...Pipe) error
}

// JoinTee fans in water from multiple upstream pipes.
type JoinTee interface {
	Tee
	Join(ctx context.Context, waters []Water) Water
	ConnectInputs(pipes ...Pipe) error
}

// Valve controls flow rate and backpressure.
type Valve interface {
	Control(ctx context.Context, water Water) (Water, *Leak)
	SetRate(rate time.Duration) // Rate limiting
	SetBufferSize(size int)     // Backpressure control
	Name() string
}

// Meter monitors pipeline flow and performance.
type Meter interface {
	Record(water Water, processingTime time.Duration, leak *Leak)
	GetMetrics() map[string]interface{}
	Name() string
}

// Strainer filters water based on conditions.
type Strainer interface {
	Filter(ctx context.Context, water Water) (Water, bool, *Leak) // water, shouldPass, leak
	Name() string
}

// Router directs water to different paths based on conditions.
type Router interface {
	Route(ctx context.Context, water Water) (string, *Leak) // routeName, leak
	AddRoute(name string, pipe Pipe) error
	Name() string
}
