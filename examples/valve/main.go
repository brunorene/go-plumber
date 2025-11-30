// Package main demonstrates using a Valve to throttle a stream.
package main

import (
	"context"
	"log"
	"time"

	"github.com/brunorene/go-plumber/plumber"
)

const (
	valveRunDuration  = 10 * time.Second
	spigotInterval    = 200 * time.Millisecond
	valveIntervalSlow = 1 * time.Second
)

func logLeaks(stage string, leaks <-chan error) {
	go func() {
		for leak := range leaks {
			log.Printf("💧 [%s] LEAK: %v", stage, leak)
		}
	}()
}

func main() {
	err := runValveExample()
	if err != nil {
		log.Fatalf("valve example error: %v", err)
	}
}

func runValveExample() error {
	log.Println("🔧 Go Plumber - Valve Example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbers, spigotLeaks := newNumberStream(ctx)

	throttled, valveLeaks := buildValveStage(ctx, numbers)

	done := buildValveTap(ctx, throttled)

	logLeaks("spigot", spigotLeaks)
	logLeaks("valve", valveLeaks)

	log.Printf("⏱️  Running valve example for %s...", valveRunDuration)
	time.Sleep(valveRunDuration)

	log.Println("🛑 Stopping valve example...")
	cancel()
	<-done

	log.Println("✅ Valve example finished")

	return nil
}

// newNumberStream creates a spigot that emits incremental integers.
func newNumberStream(ctx context.Context) (<-chan int, <-chan error) {
	counter := 0

	spigot := plumber.NewSpigot[int](
		"numbers",
		func(context.Context) (int, error) {
			counter++

			return counter, nil
		},
		spigotInterval,
	)

	numbers, leaks := spigot.Stream(ctx, nil)

	return numbers, leaks
}

// buildValveStage wires a Valve after the number stream.
func buildValveStage(
	ctx context.Context,
	numbers <-chan int,
) (<-chan int, <-chan error) {
	valve := plumber.NewValve[int](
		"slow-valve",
		valveIntervalSlow,
	)

	throttled, leaks := valve.Stream(ctx, numbers)

	return throttled, leaks
}

// buildValveTap logs the values that make it through the valve.
func buildValveTap(ctx context.Context, input <-chan int) <-chan struct{} {
	tap := plumber.NewTap[int](
		"valve-logger",
		func(_ context.Context, value int) error {
			log.Printf("🚿 throttled value: %d", value)

			return nil
		},
	)

	done, tapLeaks := tap.Stream(ctx, input)

	logLeaks("tap", tapLeaks)

	return done
}
