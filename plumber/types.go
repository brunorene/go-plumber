// Package plumber provides a pipeline framework using plumbing metaphors.
package plumber

import (
	"context"
	"errors"
)

// ErrPipe identifies errors originating from Pipe components.
var ErrPipe = errors.New("pipe")

// ErrSpigot identifies errors originating from Spigot components.
var ErrSpigot = errors.New("spigot")

// ErrTap identifies errors originating from Tap components.
var ErrTap = errors.New("tap")

// ErrSplitTee identifies errors originating from SplitTee components.
var ErrSplitTee = errors.New("splittee")

// ErrJoinTee identifies errors originating from JoinTee components.
var ErrJoinTee = errors.New("jointee")

// ErrMeter identifies errors originating from Meter components.
var ErrMeter = errors.New("meter")

// ErrStrainer identifies errors originating from Strainer components.
var ErrStrainer = errors.New("strainer")

// Named represents any plumbing component that has a stable, human-readable
// name.
type Named interface {
	Name() string
}

// Streamer is the core building block: a node that turns a stream of In values
// into a stream of Out values plus leaks.
type Streamer[In, Out any] interface {
	Named
	Stream(
		ctx context.Context,
		in <-chan In,
	) (out <-chan Out, leaks <-chan error)
}

// Pipe is a generic processing stage from In to Out. It is the main
// abstraction for both same-type and type-changing stages.
type Pipe[In, Out any] interface {
	Streamer[In, Out]
}

// Spigot is a data source that produces values of type Out from an empty
// input stream (struct{}).
type Spigot[Out any] interface {
	Pipe[struct{}, Out]
}

// Tap is a data sink that consumes values of type In and emits only a
// completion signal (struct{}).
type Tap[In any] interface {
	Pipe[In, struct{}]
}

// Valve is a pass-through stage that regulates flow of values of type T.
type Valve[T any] interface {
	Pipe[T, T]
}

// Meter is a pass-through stage that observes values of type T while letting
// them continue through the pipeline.
type Meter[T any] interface {
	Pipe[T, T]
}

// Strainer is a filtering stage that selectively passes values of type T.
type Strainer[T any] interface {
	Pipe[T, T]
}

// SplitTee is a multi-output stage: it fans out values from a single input
// stream into multiple output streams.
type SplitTee[T any] interface {
	Named
	Stream(
		ctx context.Context,
		in <-chan T,
	) (outs []<-chan T, leaks <-chan error)
}

// JoinTee is a multi-input stage: it joins values from multiple input streams
// into a single output stream.
type JoinTee[In, Out any] interface {
	Named
	Stream(
		ctx context.Context,
		ins []<-chan In,
	) (out <-chan Out, leaks <-chan error)
}
