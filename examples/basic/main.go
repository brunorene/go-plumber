// Package main demonstrates basic usage of the go-plumber framework.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brunorene/go-plumber/pkg/plumber"
)

func main() {
	fmt.Println("🔧 Go Plumber - Basic Pipeline Example")

	// Create a pipeline that:
	// 1. Generates numbers from a spigot
	// 2. Processes them through pipes (double, then add prefix)
	// 3. Outputs to a tap (console)

	pipeline, err := plumber.NewBuilder("basic-example").
		AddSpigot(createNumberSpigot()).
		AddPipe(createDoublePipe()).
		AddPipe(createPrefixPipe()).
		AddTap(createConsoleTap()).
		Build()
	if err != nil {
		log.Fatalf("Failed to build pipeline: %v", err)
	}

	// Start the pipeline
	fmt.Println("🚰 Starting pipeline...")
	if err := pipeline.Start(); err != nil {
		log.Fatalf("Failed to start pipeline: %v", err)
	}

	// Monitor for leaks
	go func() {
		for leak := range pipeline.Leaks() {
			fmt.Printf("💧 LEAK detected: %v\n", leak.Err)
		}
	}()

	// Run for 10 seconds
	fmt.Println("⏱️  Running for 10 seconds...")
	time.Sleep(10 * time.Second)

	// Stop the pipeline
	fmt.Println("🛑 Stopping pipeline...")
	if err := pipeline.Stop(); err != nil {
		log.Fatalf("Failed to stop pipeline: %v", err)
	}

	fmt.Println("✅ Pipeline stopped successfully")
}

// createNumberSpigot creates a spigot that generates sequential numbers.
func createNumberSpigot() plumber.Spigot {
	counter := 0
	return plumber.NewBasicSpigot(
		"number-generator",
		func(ctx context.Context) (plumber.Water, error) {
			counter++
			return counter, nil
		},
		1*time.Second, // Generate a number every second
	)
}

// createDoublePipe creates a pipe that doubles numbers.
func createDoublePipe() plumber.Pipe {
	return plumber.NewBasicPipe(
		"doubler",
		func(water plumber.Water) (plumber.Water, error) {
			if num, ok := water.(int); ok {
				return num * 2, nil
			}
			return water, fmt.Errorf("expected int, got %T", water)
		},
	)
}

// createPrefixPipe creates a pipe that adds a prefix to numbers.
func createPrefixPipe() plumber.Pipe {
	return plumber.NewBasicPipe(
		"prefixer",
		func(water plumber.Water) (plumber.Water, error) {
			if num, ok := water.(int); ok {
				return fmt.Sprintf("Result: %d", num), nil
			}
			return water, fmt.Errorf("expected int, got %T", water)
		},
	)
}

// createConsoleTap creates a tap that prints to console.
func createConsoleTap() plumber.Tap {
	return plumber.NewBasicTap(
		"console-output",
		func(water plumber.Water) error {
			fmt.Printf("🚿 Output: %v\n", water)
			return nil
		},
	)
}

// Example of a more complex pipeline with a split tee
func createSplitExample() *plumber.Pipeline {
	// This would create a pipeline that splits data to multiple paths
	splitTee := plumber.NewBasicSplitTee(
		"data-splitter",
		func(water plumber.Water) []plumber.Water {
			if str, ok := water.(string); ok {
				// Split string into words
				words := strings.Fields(str)
				result := make([]plumber.Water, len(words))
				for i, word := range words {
					result[i] = word
				}
				return result
			}
			return []plumber.Water{water}
		},
	)

	pipeline, _ := plumber.NewBuilder("split-example").
		AddSpigot(plumber.NewBasicSpigot(
			"sentence-generator",
			func(ctx context.Context) (plumber.Water, error) {
				sentences := []string{
					"Hello world from plumber",
					"Go pipelines are awesome",
					"Plumbing metaphors work great",
				}
				return sentences[time.Now().Second()%len(sentences)], nil
			},
			2*time.Second,
		)).
		AddTee(splitTee).
		AddTap(plumber.NewBasicTap(
			"word-counter",
			func(water plumber.Water) error {
				if word, ok := water.(string); ok {
					fmt.Printf("📝 Word: '%s' (length: %d)\n", word, len(word))
				}
				return nil
			},
		)).
		Build()

	return pipeline
}
