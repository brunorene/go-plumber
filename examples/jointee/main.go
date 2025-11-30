// Package main demonstrates using BasicSplitTee and BasicJoinTee together:
// values are split into slices and then joined back into aggregated results.
package main

import (
	"context"
	"log"
	"time"

	"github.com/brunorene/go-plumber/plumber"
)

const joinRunDuration = 10 * time.Second

func logLeaks(stage string, leaks <-chan error) {
	go func() {
		for leak := range leaks {
			log.Printf("💧 [%s] LEAK: %v", stage, leak)
		}
	}()
}

func main() {
	err := runJoinTeeExample()
	if err != nil {
		log.Fatalf("join tee example error: %v", err)
	}
}

func runJoinTeeExample() error {
	log.Println("🔧 Go Plumber - JoinTee Example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Source streams: two independent number streams.
	left, leftLeaks := newNumberStream(ctx, "left")
	right, rightLeaks := newNumberStream(ctx, "right")

	sumsDone := buildJoinStage(ctx, left, right)

	logLeaks("spigot-left", leftLeaks)
	logLeaks("spigot-right", rightLeaks)

	log.Printf("⏱️  Running join tee example for %s...", joinRunDuration)
	time.Sleep(joinRunDuration)

	log.Println("🛑 Stopping join tee example...")
	cancel()

	<-sumsDone

	log.Println("✅ Join tee example finished")

	return nil
}

// newNumberStream creates a spigot that emits incremental integers and returns
// its output and leak channels.
func newNumberStream(ctx context.Context, name string) (<-chan int, <-chan error) {
	counter := 0
	spigot := plumber.NewSpigot[int](
		name,
		func(context.Context) (int, error) {
			counter++

			return counter, nil
		},
		1*time.Second,
	)

	numbers, leaks := spigot.Stream(ctx, nil)

	return numbers, leaks
}

// buildJoinStage uses BasicJoinTee to sum pairs of values (one from each input
// stream) and log the result via a Tap.
func buildJoinStage(ctx context.Context, left, right <-chan int) <-chan struct{} {
	joinTee := plumber.NewJoinTee[int, int](
		"sum-join",
		func(_ context.Context, batch []int) (int, error) {
			const expectedInputs = 2
			if len(batch) != expectedInputs {
				return 0, nil
			}

			return batch[0] + batch[1], nil
		},
	)

	sums, joinLeaks := joinTee.Stream(ctx, []<-chan int{left, right})

	tap := plumber.NewTap[int](
		"sum-logger",
		func(_ context.Context, v int) error {
			log.Printf("📦 joined sum: %d", v)

			return nil
		},
	)

	done, tapLeaks := tap.Stream(ctx, sums)

	logLeaks("join", joinLeaks)
	logLeaks("tap", tapLeaks)

	return done
}
