// Package main demonstrates basic usage of the go-plumber framework using the
// streaming API directly (no Pipeline abstraction).
package main

import (
	"context"
	"log"
	"time"

	"github.com/brunorene/go-plumber/plumber"
)

const (
	doubleFactor     = 2
	basicRunDuration = 10 * time.Second
)

func logLeaks(leaks <-chan error) {
	go func() {
		for leak := range leaks {
			log.Printf("💧 LEAK: %v", leak)
		}
	}()
}

func main() {
	err := runBasicStream()
	if err != nil {
		log.Fatalf("stream error: %v", err)
	}
}

// runBasicStream builds and runs a simple streaming graph:
//
//	spigot -> doubler -> incrementer -> tap
//
// The spigot generates an increasing integer every second, the pipes transform
// it, and the tap logs the final value.
func runBasicStream() error {
	log.Println("🔧 Go Plumber - Basic Streaming Example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := buildIntStream(ctx)

	log.Println("⏱️  Running for 10 seconds...")
	time.Sleep(basicRunDuration)

	log.Println("🛑 Stopping stream...")
	cancel()

	// Wait for the tap to finish draining.
	<-done

	log.Println("✅ Stream stopped successfully")

	return nil
}

// buildIntStream constructs the integer streaming graph and starts all the
// goroutines. It returns a channel that is closed when the tap finishes
// draining.
func buildIntStream(ctx context.Context) <-chan struct{} {
	// 1. Final tap that prints to the console.
	tap := plumber.NewTap[int](
		"console-output",
		func(_ context.Context, value int) error {
			log.Printf("🚿 Output: %d", value)

			return nil
		},
	)

	// 2. Pipes that transform the integers.
	incrementPipe := plumber.NewPipe[int, int](
		"incrementer",
		func(_ context.Context, value int) (int, error) {
			return value + 1, nil
		},
	)

	doublePipe := plumber.NewPipe[int, int](
		"doubler",
		func(_ context.Context, value int) (int, error) {
			return value * doubleFactor, nil
		},
	)

	// 3. Spigot that generates sequential integers.
	counter := 0
	spigot := plumber.NewSpigot[int](
		"number-generator",
		func(_ context.Context) (int, error) {
			counter++

			return counter, nil
		},
		1*time.Second, // Generate a number every second
	)

	// Wire the graph using Stream.
	spigotOut, spigotLeaks := spigot.Stream(ctx, nil)
	doubledOut, doublerLeaks := doublePipe.Stream(ctx, spigotOut)
	incrementedOut, incrementLeaks := incrementPipe.Stream(ctx, doubledOut)
	done, tapLeaks := tap.Stream(ctx, incrementedOut)

	// Log leaks from each stage.
	logLeaks(spigotLeaks)
	logLeaks(doublerLeaks)
	logLeaks(incrementLeaks)
	logLeaks(tapLeaks)

	return done
}
