package plumber

import (
	"context"
	"fmt"
	"sync"
)

// Pipeline represents a complete plumbing system.
type Pipeline struct {
	name     string
	spigots  []Spigot
	pipes    []Pipe
	tees     []Tee
	valves   []Valve
	meters   []Meter
	taps     []Tap
	leakChan chan *Leak
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Builder constructs pipelines using the builder pattern.
type Builder struct {
	pipeline *Pipeline
}

// NewBuilder creates a new pipeline builder.
func NewBuilder(name string) *Builder {
	ctx, cancel := context.WithCancel(context.Background())
	return &Builder{
		pipeline: &Pipeline{
			name:     name,
			spigots:  make([]Spigot, 0),
			pipes:    make([]Pipe, 0),
			tees:     make([]Tee, 0),
			valves:   make([]Valve, 0),
			meters:   make([]Meter, 0),
			taps:     make([]Tap, 0),
			leakChan: make(chan *Leak, 100), // Buffered leak channel
			ctx:      ctx,
			cancel:   cancel,
		},
	}
}

// AddSpigot adds a water source to the pipeline.
func (b *Builder) AddSpigot(spigot Spigot) *Builder {
	b.pipeline.spigots = append(b.pipeline.spigots, spigot)
	return b
}

// AddPipe adds a processing pipe to the pipeline.
func (b *Builder) AddPipe(pipe Pipe) *Builder {
	b.pipeline.pipes = append(b.pipeline.pipes, pipe)
	return b
}

// AddTee adds a tee (splitter/joiner) to the pipeline.
func (b *Builder) AddTee(tee Tee) *Builder {
	b.pipeline.tees = append(b.pipeline.tees, tee)
	return b
}

// AddValve adds flow control to the pipeline.
func (b *Builder) AddValve(valve Valve) *Builder {
	b.pipeline.valves = append(b.pipeline.valves, valve)
	return b
}

// AddMeter adds monitoring to the pipeline.
func (b *Builder) AddMeter(meter Meter) *Builder {
	b.pipeline.meters = append(b.pipeline.meters, meter)
	return b
}

// AddTap adds a data sink to the pipeline.
func (b *Builder) AddTap(tap Tap) *Builder {
	b.pipeline.taps = append(b.pipeline.taps, tap)
	return b
}

// Build constructs and returns the pipeline.
func (b *Builder) Build() (*Pipeline, error) {
	if len(b.pipeline.spigots) == 0 {
		return nil, fmt.Errorf("pipeline '%s' must have at least one spigot", b.pipeline.name)
	}
	if len(b.pipeline.taps) == 0 {
		return nil, fmt.Errorf("pipeline '%s' must have at least one tap", b.pipeline.name)
	}
	return b.pipeline, nil
}

// Start begins pipeline execution.
func (p *Pipeline) Start() error {
	// Start leak monitoring
	go p.monitorLeaks()

	// Start spigots
	for _, spigot := range p.spigots {
		p.wg.Add(1)
		go p.runSpigot(spigot)
	}

	return nil
}

// Stop gracefully shuts down the pipeline.
func (p *Pipeline) Stop() error {
	p.cancel()
	p.wg.Wait()
	close(p.leakChan)
	return nil
}

// Leaks returns a channel for monitoring pipeline errors.
func (p *Pipeline) Leaks() <-chan *Leak {
	return p.leakChan
}

// Name returns the pipeline name.
func (p *Pipeline) Name() string {
	return p.name
}

// runSpigot starts a spigot and handles its water flow.
func (p *Pipeline) runSpigot(spigot Spigot) {
	defer p.wg.Done()

	waterChan, leakChan := spigot.Flow(p.ctx)

	for {
		select {
		case <-p.ctx.Done():
			return
		case water, ok := <-waterChan:
			if !ok {
				return
			}
			// Process water through the pipeline
			go p.processWater(water, spigot.Name())
		case leak, ok := <-leakChan:
			if !ok {
				return
			}
			p.leakChan <- leak
		}
	}
}

// processWater handles water flow through the pipeline.
func (p *Pipeline) processWater(water Water, source string) {
	currentWater := water

	// Process through all pipes sequentially
	for _, pipe := range p.pipes {
		processedWater, leak := pipe.Process(p.ctx, currentWater)
		if leak != nil {
			p.leakChan <- leak
			return // Stop processing on leak
		}
		currentWater = processedWater
	}

	// Send processed water to all taps
	for _, tap := range p.taps {
		if leak := tap.Drain(p.ctx, currentWater); leak != nil {
			p.leakChan <- leak
		}
	}
}

// monitorLeaks handles leak reporting and logging.
func (p *Pipeline) monitorLeaks() {
	for leak := range p.leakChan {
		// For now, just log leaks - could be extended to handle differently
		fmt.Printf("LEAK in pipeline '%s': %v (source: %s)\n", p.name, leak.Err, leak.Source)
	}
}
