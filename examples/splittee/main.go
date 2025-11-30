// Package main demonstrates using a split tee style fan-out with multiple
// branches using the streaming API.
package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/brunorene/go-plumber/plumber"
)

const (
	splitRunDuration  = 10 * time.Second
	splitDoubleFactor = 2
)

func logLeaks(stage string, leaks <-chan error) {
	go func() {
		for leak := range leaks {
			log.Printf("💧 [%s] LEAK: %v", stage, leak)
		}
	}()
}

// fanOut3 acts as a simple split tee at the application level: every value
// received on in is broadcast to all three output channels.
func fanOut3[T any](ctx context.Context, input <-chan T) (<-chan T, <-chan T, <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)
	out3 := make(chan T)

	go func() {
		defer close(out1)
		defer close(out2)
		defer close(out3)

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-input:
				if !ok {
					return
				}

				for _, ch := range []chan T{out1, out2, out3} {
					select {
					case <-ctx.Done():
						return
					case ch <- value:
					}
				}
			}
		}
	}()

	return out1, out2, out3
}

func main() {
	err := runSplitTeeExample()
	if err != nil {
		log.Fatalf("split tee example error: %v", err)
	}
}

func runSplitTeeExample() error {
	log.Println("🔧 Go Plumber - SplitTee Streaming Example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := buildSplitTeeStream(ctx)

	log.Println("⏱️  Running split tee example for 10 seconds...")
	time.Sleep(splitRunDuration)

	log.Println("🛑 Stopping split tee example...")
	cancel()

	<-done

	log.Println("✅ Split tee example finished")

	return nil
}

// buildSplitTeeStream wires the following graph:
//
//	spigot -> fanOut3 -> [inc pipe] -> tap
//	                     [double pipe] -> tap
//	                     [inc pipe] -> [int->string "A"*n pipe] -> tap
func buildSplitTeeStream(ctx context.Context) <-chan struct{} {
	numbers, spigotLeaks := newNumberStream(ctx)

	return buildSplitTeeBranches(ctx, numbers, spigotLeaks)
}

// newNumberStream creates a spigot that emits incremental integers and returns
// its output and leak channels.
func newNumberStream(ctx context.Context) (<-chan int, <-chan error) {
	counter := 0
	spigot := plumber.NewSpigot[int](
		"numbers",
		func(context.Context) (int, error) {
			counter++

			return counter, nil
		},
		1*time.Second,
	)

	numbers, spigotLeaks := spigot.Stream(ctx, nil)

	return numbers, spigotLeaks
}

// buildSplitTeeBranches constructs the split tee branches and taps and returns
// a channel that is closed when all taps are done.
func buildSplitTeeBranches(
	ctx context.Context,
	numbers <-chan int,
	spigotLeaks <-chan error,
) <-chan struct{} {
	// Split tee at the application level so all branches see every value.
	b1In, b2In, b3In := fanOut3(ctx, numbers)

	b1Out, b1Leaks, b2Out, b2Leaks, incAgainLeaks, strOut, strLeaks := buildSplitTeePipes(
		ctx,
		b1In,
		b2In,
		b3In,
	)

	return startSplitTeeSinksAndLogging(
		ctx,
		spigotLeaks,
		b1Out,
		b1Leaks,
		b2Out,
		b2Leaks,
		incAgainLeaks,
		strOut,
		strLeaks,
	)
}

// buildSplitTeePipes constructs the processing pipes for each branch.
func buildSplitTeePipes(
	ctx context.Context,
	b1In <-chan int,
	b2In <-chan int,
	b3In <-chan int,
) (
	<-chan int,
	<-chan error,
	<-chan int,
	<-chan error,
	<-chan error,
	<-chan string,
	<-chan error,
) {
	incFun := func(_ context.Context, value int) (int, error) {
		return value + 1, nil
	}

	inc := plumber.NewPipe[int, int]("increment", incFun)

	dbl := plumber.NewPipe[int, int](
		"double",
		func(_ context.Context, value int) (int, error) {
			return value * splitDoubleFactor, nil
		},
	)

	incAgain := plumber.NewPipe[int, int]("increment-again", incFun)

	toAs := plumber.NewPipe[int, string](
		"int-to-As",
		func(_ context.Context, value int) (string, error) {
			if value <= 0 {
				return "", nil
			}

			return strings.Repeat("A", value), nil
		},
	)

	b1Out, b1Leaks := inc.Stream(ctx, b1In)
	b2Out, b2Leaks := dbl.Stream(ctx, b2In)
	incOut, incAgainLeaks := incAgain.Stream(ctx, b3In)
	strOut, strLeaks := toAs.Stream(ctx, incOut)

	return b1Out, b1Leaks, b2Out, b2Leaks, incAgainLeaks, strOut, strLeaks
}

// startSplitTeeSinksAndLogging creates taps, starts sinks, and wires leak
// logging for the split tee example.
func startSplitTeeSinksAndLogging(
	ctx context.Context,
	spigotLeaks <-chan error,
	b1Out <-chan int,
	b1Leaks <-chan error,
	b2Out <-chan int,
	b2Leaks <-chan error,
	incAgainLeaks <-chan error,
	strOut <-chan string,
	strLeaks <-chan error,
) <-chan struct{} {
	tap1 := plumber.NewTap[int](
		"tap-inc",
		func(_ context.Context, value int) error {
			log.Printf("[branch 1] incremented: %d", value)

			return nil
		},
	)
	done1, t1Leaks := tap1.Stream(ctx, b1Out)

	tap2 := plumber.NewTap[int](
		"tap-double",
		func(_ context.Context, value int) error {
			log.Printf("[branch 2] doubled: %d", value)

			return nil
		},
	)
	done2, t2Leaks := tap2.Stream(ctx, b2Out)

	tap3 := plumber.NewTap[string](
		"tap-int-to-As",
		func(_ context.Context, s string) error {
			log.Printf("[branch 3] As: %q (len=%d)", s, len(s))

			return nil
		},
	)
	done3, t3Leaks := tap3.Stream(ctx, strOut)

	logLeaks("spigot", spigotLeaks)
	logLeaks("inc", b1Leaks)
	logLeaks("double", b2Leaks)
	logLeaks("increment-again", incAgainLeaks)
	logLeaks("int-to-As", strLeaks)
	logLeaks("tap-inc", t1Leaks)
	logLeaks("tap-double", t2Leaks)
	logLeaks("tap-int-to-As", t3Leaks)

	doneAll := make(chan struct{})

	go func() {
		defer close(doneAll)

		<-done1
		<-done2
		<-done3
	}()

	return doneAll
}
