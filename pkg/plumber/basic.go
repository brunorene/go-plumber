package plumber

import (
	"context"
	"time"
)

// BasicPipe is a simple pipe implementation.
type BasicPipe struct {
	name      string
	processor func(Water) (Water, error)
	next      Pipe
}

// NewBasicPipe creates a new basic pipe with a processing function.
func NewBasicPipe(name string, processor func(Water) (Water, error)) *BasicPipe {
	return &BasicPipe{
		name:      name,
		processor: processor,
	}
}

// Process implements the Pipe interface.
func (p *BasicPipe) Process(ctx context.Context, water Water) (Water, *Leak) {
	if p.processor == nil {
		return water, nil
	}

	result, err := p.processor(water)
	if err != nil {
		return water, &Leak{
			Err:     err,
			Source:  p.name,
			Water:   water,
			Time:    time.Now(),
			Context: map[string]interface{}{"pipe": p.name},
		}
	}

	return result, nil
}

// Connect implements the Pipe interface.
func (p *BasicPipe) Connect(downstream Pipe) error {
	p.next = downstream
	return nil
}

// Name implements the Pipe interface.
func (p *BasicPipe) Name() string {
	return p.name
}

// BasicSpigot is a simple data source implementation.
type BasicSpigot struct {
	name      string
	generator func(context.Context) (Water, error)
	interval  time.Duration
}

// NewBasicSpigot creates a new basic spigot.
func NewBasicSpigot(name string, generator func(context.Context) (Water, error), interval time.Duration) *BasicSpigot {
	return &BasicSpigot{
		name:      name,
		generator: generator,
		interval:  interval,
	}
}

// Flow implements the Spigot interface.
func (s *BasicSpigot) Flow(ctx context.Context) (<-chan Water, <-chan *Leak) {
	waterChan := make(chan Water)
	leakChan := make(chan *Leak)

	go func() {
		defer close(waterChan)
		defer close(leakChan)

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				water, err := s.generator(ctx)
				if err != nil {
					leakChan <- &Leak{
						Err:     err,
						Source:  s.name,
						Water:   nil,
						Time:    time.Now(),
						Context: map[string]interface{}{"spigot": s.name},
					}
					continue
				}

				select {
				case waterChan <- water:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return waterChan, leakChan
}

// Name implements the Spigot interface.
func (s *BasicSpigot) Name() string {
	return s.name
}

// BasicTap is a simple data sink implementation.
type BasicTap struct {
	name    string
	handler func(Water) error
}

// NewBasicTap creates a new basic tap.
func NewBasicTap(name string, handler func(Water) error) *BasicTap {
	return &BasicTap{
		name:    name,
		handler: handler,
	}
}

// Drain implements the Tap interface.
func (t *BasicTap) Drain(ctx context.Context, water Water) *Leak {
	if t.handler == nil {
		return nil
	}

	if err := t.handler(water); err != nil {
		return &Leak{
			Err:     err,
			Source:  t.name,
			Water:   water,
			Time:    time.Now(),
			Context: map[string]interface{}{"tap": t.name},
		}
	}

	return nil
}

// Name implements the Tap interface.
func (t *BasicTap) Name() string {
	return t.name
}

// BasicSplitTee splits water to multiple outputs.
type BasicSplitTee struct {
	name     string
	outputs  []Pipe
	splitter func(Water) []Water
}

// NewBasicSplitTee creates a new split tee.
func NewBasicSplitTee(name string, splitter func(Water) []Water) *BasicSplitTee {
	return &BasicSplitTee{
		name:     name,
		outputs:  make([]Pipe, 0),
		splitter: splitter,
	}
}

// Split implements the SplitTee interface.
func (t *BasicSplitTee) Split(ctx context.Context, water Water) []Water {
	if t.splitter == nil {
		// Default: duplicate water to all outputs
		result := make([]Water, len(t.outputs))
		for i := range result {
			result[i] = water
		}
		return result
	}
	return t.splitter(water)
}

// ConnectOutputs implements the SplitTee interface.
func (t *BasicSplitTee) ConnectOutputs(pipes ...Pipe) error {
	t.outputs = append(t.outputs, pipes...)
	return nil
}

// Name implements the Tee interface.
func (t *BasicSplitTee) Name() string {
	return t.name
}
