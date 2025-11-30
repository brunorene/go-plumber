package plumber

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	errPipeNoProcessor     = errors.New("pipe has no processor")
	errSpigotNoGenerator   = errors.New("spigot has no generator")
	errTapNoHandler        = errors.New("tap has no handler")
	errJoinTeeNoJoiner     = errors.New("jointee has no joiner")
	errMeterNoMeasure      = errors.New("meter has no measure function")
	errStrainerNoPredicate = errors.New("strainer has no predicate")
)

// BasicPipe is a simple implementation of Pipe[In, Out].
type BasicPipe[In, Out any] struct {
	name      string
	processor func(context.Context, In) (Out, error)
}

// NewPipe creates a new pipe with a processing function.
func NewPipe[In, Out any](
	name string,
	processor func(context.Context, In) (Out, error),
) *BasicPipe[In, Out] {
	return &BasicPipe[In, Out]{
		name:      name,
		processor: processor,
	}
}

// Process runs the pipe's processor for a single value and returns a stage-level
// error if the processor fails.
//
//nolint:ireturn // generic pipe returns its output type parameter
func (p *BasicPipe[In, Out]) Process(ctx context.Context, input In) (Out, error) {
	var zeroOut Out

	if p.processor == nil {
		return zeroOut, fmt.Errorf("%w: %s: %w", ErrPipe, p.name, errPipeNoProcessor)
	}

	out, err := p.processor(ctx, input)
	if err != nil {
		return zeroOut, fmt.Errorf("%w: %s: %w", ErrPipe, p.name, err)
	}

	return out, nil
}

// Stream implements the Streamer interface for BasicPipe.
// It reads values from the input channel, processes them, and emits either
// transformed values or stage-level errors.
func (p *BasicPipe[In, Out]) Stream(
	ctx context.Context,
	inputCh <-chan In,
) (<-chan Out, <-chan error) {
	out := make(chan Out)
	leaks := make(chan error)

	go func() {
		defer close(out)
		defer close(leaks)

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-inputCh:
				if !ok {
					return
				}

				result, err := p.Process(ctx, value)
				if err != nil {
					leaks <- err

					continue
				}

				out <- result
			}
		}
	}()

	return out, leaks
}

// Name implements the Named interface.
func (p *BasicPipe[In, Out]) Name() string {
	return p.name
}

// BasicSpigot is a simple data source implementation for water of type Out.
type BasicSpigot[Out any] struct {
	name      string
	generator func(context.Context) (Out, error)
	interval  time.Duration
}

// NewSpigot creates a new spigot that generates values on a ticker.
func NewSpigot[Out any](
	name string,
	generator func(context.Context) (Out, error),
	interval time.Duration,
) *BasicSpigot[Out] {
	return &BasicSpigot[Out]{
		name:      name,
		generator: generator,
		interval:  interval,
	}
}

// Stream implements the Streamer interface for BasicSpigot.
// It generates values using the configured generator on a ticker and emits
// them on the output channel, along with any stage-level errors.
func (s *BasicSpigot[Out]) Stream(
	ctx context.Context,
	_ <-chan struct{},
) (<-chan Out, <-chan error) {
	out := make(chan Out)
	leaks := make(chan error)

	go func() {
		defer close(out)
		defer close(leaks)

		if s.generator == nil {
			leaks <- fmt.Errorf("%w: %s: %w", ErrSpigot, s.name, errSpigotNoGenerator)

			return
		}

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				value, err := s.generator(ctx)
				if err != nil {
					leaks <- fmt.Errorf("%w: %s: %w", ErrSpigot, s.name, err)

					continue
				}

				out <- value
			}
		}
	}()

	return out, leaks
}

// Name implements the Named interface.
func (s *BasicSpigot[Out]) Name() string {
	return s.name
}

// BasicTap is a simple data sink implementation for water of type In.
type BasicTap[In any] struct {
	name    string
	handler func(context.Context, In) error
}

// NewTap creates a new tap.
func NewTap[In any](name string, handler func(context.Context, In) error) *BasicTap[In] {
	return &BasicTap[In]{
		name:    name,
		handler: handler,
	}
}

// Drain applies the tap handler to a single value and returns a stage-level
// error if it fails.
func (t *BasicTap[In]) Drain(ctx context.Context, water In) error {
	if t.handler == nil {
		return fmt.Errorf("%w: %s: %w", ErrTap, t.name, errTapNoHandler)
	}

	err := t.handler(ctx, water)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrTap, t.name, err)
	}

	return nil
}

// Stream implements the Streamer interface for BasicTap.
// It drains values from the input channel and emits only errors; the output
// channel carries a struct{} value stream that can be used as a completion
// signal if desired.
func (t *BasicTap[In]) Stream(
	ctx context.Context,
	inputCh <-chan In,
) (<-chan struct{}, <-chan error) {
	out := make(chan struct{})
	leaks := make(chan error)

	go func() {
		defer close(out)
		defer close(leaks)

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-inputCh:
				if !ok {
					return
				}

				err := t.Drain(ctx, value)
				if err != nil {
					leaks <- err
				}
			}
		}
	}()

	return out, leaks
}

// Name implements the Named interface.
func (t *BasicTap[In]) Name() string {
	return t.name
}

// BasicValve is a pass-through stage that throttles the rate at which values
// flow through the pipeline.
type BasicValve[T any] struct {
	name     string
	interval time.Duration
}

// NewValve creates a new valve that limits throughput to at most one value per
// interval. If interval is zero or negative, values are passed through without
// additional throttling.
func NewValve[T any](name string, interval time.Duration) *BasicValve[T] {
	return &BasicValve[T]{
		name:     name,
		interval: interval,
	}
}

// Stream implements the Streamer interface for BasicValve. It forwards values
// unchanged, optionally delaying each value according to the configured
// interval.
//
//nolint:cyclop // small state machine for throttling reads and writes
func (v *BasicValve[T]) Stream(
	ctx context.Context,
	inputCh <-chan T,
) (<-chan T, <-chan error) {
	out := make(chan T)
	leaks := make(chan error)

	go func() {
		defer close(out)
		defer close(leaks)

		if v.interval <= 0 {
			for {
				select {
				case <-ctx.Done():
					return
				case value, ok := <-inputCh:
					if !ok {
						return
					}

					out <- value
				}
			}
		}

		ticker := time.NewTicker(v.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-inputCh:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}

				out <- value
			}
		}
	}()

	return out, leaks
}

// Name implements the Named interface.
func (v *BasicValve[T]) Name() string {
	return v.name
}

// BasicMeter is a pass-through stage that observes values and can emit leaks
// if metric collection fails.
type BasicMeter[T any] struct {
	name    string
	measure func(context.Context, T) error
}

// NewMeter creates a new meter with an optional measurement function.
func NewMeter[T any](
	name string,
	measure func(context.Context, T) error,
) *BasicMeter[T] {
	return &BasicMeter[T]{
		name:    name,
		measure: measure,
	}
}

// Measure runs the configured measurement function for a single value.
func (m *BasicMeter[T]) Measure(ctx context.Context, value T) error {
	if m.measure == nil {
		return fmt.Errorf("%w: %s: %w", ErrMeter, m.name, errMeterNoMeasure)
	}

	measureErr := m.measure(ctx, value)
	if measureErr != nil {
		return fmt.Errorf("%w: %s: %w", ErrMeter, m.name, measureErr)
	}

	return nil
}

// Stream implements the Streamer interface for BasicMeter. It forwards values
// unchanged while invoking the measurement function.
func (m *BasicMeter[T]) Stream(
	ctx context.Context,
	inputCh <-chan T,
) (<-chan T, <-chan error) {
	out := make(chan T)
	leaks := make(chan error)

	go func() {
		defer close(out)
		defer close(leaks)

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-inputCh:
				if !ok {
					return
				}

				measureErr := m.Measure(ctx, value)
				if measureErr != nil {
					leaks <- measureErr

					continue
				}

				out <- value
			}
		}
	}()

	return out, leaks
}

// Name implements the Named interface.
func (m *BasicMeter[T]) Name() string {
	return m.name
}

// BasicStrainer is a filtering stage that only allows values that satisfy a
// predicate to pass through.
type BasicStrainer[T any] struct {
	name      string
	predicate func(context.Context, T) (bool, error)
}

// NewStrainer creates a new strainer with a predicate function.
func NewStrainer[T any](
	name string,
	predicate func(context.Context, T) (bool, error),
) *BasicStrainer[T] {
	return &BasicStrainer[T]{
		name:      name,
		predicate: predicate,
	}
}

// Filter applies the predicate to a single value.
func (s *BasicStrainer[T]) Filter(ctx context.Context, value T) (bool, error) {
	if s.predicate == nil {
		return false, fmt.Errorf("%w: %s: %w", ErrStrainer, s.name, errStrainerNoPredicate)
	}

	keep, err := s.predicate(ctx, value)
	if err != nil {
		return false, fmt.Errorf("%w: %s: %w", ErrStrainer, s.name, err)
	}

	return keep, nil
}

// Stream implements the Streamer interface for BasicStrainer. It emits only
// the values that satisfy the configured predicate.
func (s *BasicStrainer[T]) Stream(
	ctx context.Context,
	inputCh <-chan T,
) (<-chan T, <-chan error) {
	out := make(chan T)
	leaks := make(chan error)

	go func() {
		defer close(out)
		defer close(leaks)

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-inputCh:
				if !ok {
					return
				}

				keep, err := s.Filter(ctx, value)
				if err != nil {
					leaks <- err

					continue
				}

				if keep {
					out <- value
				}
			}
		}
	}()

	return out, leaks
}

// Name implements the Named interface.
func (s *BasicStrainer[T]) Name() string {
	return s.name
}

// BasicSplitTee fans out water of type T from a single input stream to
// multiple output streams, broadcasting each value to every branch.
type BasicSplitTee[T any] struct {
	name     string
	branches int
}

// NewSplitTee creates a new split tee that will emit each input value to the
// specified number of branches.
func NewSplitTee[T any](name string, branches int) *BasicSplitTee[T] {
	return &BasicSplitTee[T]{
		name:     name,
		branches: branches,
	}
}

// Stream implements the SplitTee interface for BasicSplitTee.
// It broadcasts each incoming value to all output branches.
func (t *BasicSplitTee[T]) Stream(
	ctx context.Context,
	inputCh <-chan T,
) ([]<-chan T, <-chan error) {
	outs := make([]chan T, t.branches)
	outChans := make([]<-chan T, t.branches)

	for i := range outs {
		outs[i] = make(chan T)
		outChans[i] = outs[i]
	}

	leaks := make(chan error)

	go func() {
		defer close(leaks)

		for _, outCh := range outs {
			defer close(outCh)
		}

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-inputCh:
				if !ok {
					return
				}

				for _, outCh := range outs {
					select {
					case <-ctx.Done():
						return
					case outCh <- value:
					}
				}
			}
		}
	}()

	return outChans, leaks
}

// Name implements the Named interface.
func (t *BasicSplitTee[T]) Name() string {
	return t.name
}

// BasicJoinTee joins batches of values of type In into single values of type Out.
// It complements BasicSplitTee by collapsing slices back into scalar outputs.
type BasicJoinTee[In, Out any] struct {
	name   string
	joiner func(context.Context, []In) (Out, error)
}

// NewJoinTee creates a new join tee.
func NewJoinTee[In, Out any](
	name string,
	joiner func(context.Context, []In) (Out, error),
) *BasicJoinTee[In, Out] {
	return &BasicJoinTee[In, Out]{
		name:   name,
		joiner: joiner,
	}
}

// Join applies the configured joiner to a batch of input values.
//
//nolint:ireturn // generic join tee returns its output type parameter
func (j *BasicJoinTee[In, Out]) Join(ctx context.Context, batch []In) (Out, error) {
	var zeroOut Out

	if j.joiner == nil {
		return zeroOut, fmt.Errorf("%w: %s: %w", ErrJoinTee, j.name, errJoinTeeNoJoiner)
	}

	out, err := j.joiner(ctx, batch)
	if err != nil {
		return zeroOut, fmt.Errorf("%w: %s: %w", ErrJoinTee, j.name, err)
	}

	return out, nil
}

// Stream implements the JoinTee interface for BasicJoinTee.
// It synchronizes values from all input streams, joining one value from each
// input into a single output value per round.
func (j *BasicJoinTee[In, Out]) Stream(
	ctx context.Context,
	ins []<-chan In,
) (<-chan Out, <-chan error) {
	out := make(chan Out)
	leaks := make(chan error)

	go func() {
		defer close(out)
		defer close(leaks)

		if len(ins) == 0 {
			return
		}

		for {
			batch := make([]In, len(ins))

			// Collect one value from each input stream.
			for inputIndex, inputCh := range ins {
				select {
				case <-ctx.Done():
					return
				case value, ok := <-inputCh:
					if !ok {
						// If any input closes, terminate the join.
						return
					}

					batch[inputIndex] = value
				}
			}

			result, err := j.Join(ctx, batch)
			if err != nil {
				leaks <- err

				continue
			}

			select {
			case <-ctx.Done():
				return
			case out <- result:
			}
		}
	}()

	return out, leaks
}

// Name implements the Named interface.
func (j *BasicJoinTee[In, Out]) Name() string {
	return j.name
}
