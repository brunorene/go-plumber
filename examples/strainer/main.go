// Package main demonstrates using a Strainer to filter values in a stream.
package main

import (
	"context"
	"log"
	"time"

	"github.com/brunorene/go-plumber/plumber"
)

const (
	strainerRunDuration    = 10 * time.Second
	strainerSpigotInterval = 500 * time.Millisecond
	strainerEvenDivisor    = 2
)

func logLeaks(stage string, leaks <-chan error) {
	go func() {
		for leak := range leaks {
			log.Printf("💧 [%s] LEAK: %v", stage, leak)
		}
	}()
}

func main() {
	err := runStrainerExample()
	if err != nil {
		log.Fatalf("strainer example error: %v", err)
	}
}

func runStrainerExample() error {
	log.Println("🔧 Go Plumber - Strainer Example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbers, spigotLeaks := newNumberStream(ctx)

	filtered, strainerLeaks := buildStrainerStage(ctx, numbers)

	done := buildStrainerTap(ctx, filtered)

	logLeaks("spigot", spigotLeaks)
	logLeaks("strainer", strainerLeaks)

	log.Printf("⏱️  Running strainer example for %s...", strainerRunDuration)
	time.Sleep(strainerRunDuration)

	log.Println("🛑 Stopping strainer example...")
	cancel()
	<-done

	log.Println("✅ Strainer example finished")

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
		strainerSpigotInterval,
	)

	numbers, leaks := spigot.Stream(ctx, nil)

	return numbers, leaks
}

// buildStrainerStage wires a Strainer that passes only even values.
func buildStrainerStage(
	ctx context.Context,
	numbers <-chan int,
) (<-chan int, <-chan error) {
	strainer := plumber.NewStrainer[int](
		"even-only",
		func(_ context.Context, value int) (bool, error) {
			return value%strainerEvenDivisor == 0, nil
		},
	)

	filtered, leaks := strainer.Stream(ctx, numbers)

	return filtered, leaks
}

// buildStrainerTap logs the values that make it through the strainer.
func buildStrainerTap(ctx context.Context, input <-chan int) <-chan struct{} {
	tap := plumber.NewTap[int](
		"strainer-logger",
		func(_ context.Context, value int) error {
			log.Printf("✅ even value: %d", value)

			return nil
		},
	)

	done, tapLeaks := tap.Stream(ctx, input)

	logLeaks("tap", tapLeaks)

	return done
}
