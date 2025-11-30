// Package main demonstrates using a Meter to observe values in a stream.
// In real applications, the meter's measurement function is a convenient
// place to integrate Prometheus, OpenTelemetry, or any other metrics backend.
package main

import (
	"context"
	"log"
	"time"

	"github.com/brunorene/go-plumber/plumber"
)

const (
	meterRunDuration     = 10 * time.Second
	meterSpigotInterval  = 500 * time.Millisecond
	meterLogIntervalSecs = 2
)

func logLeaks(stage string, leaks <-chan error) {
	go func() {
		for leak := range leaks {
			log.Printf("💧 [%s] LEAK: %v", stage, leak)
		}
	}()
}

func main() {
	err := runMeterExample()
	if err != nil {
		log.Fatalf("meter example error: %v", err)
	}
}

func runMeterExample() error {
	log.Println("🔧 Go Plumber - Meter Example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbers, spigotLeaks := newNumberStream(ctx)

	metered, meterLeaks := buildMeterStage(ctx, numbers)

	done := buildMeterTap(ctx, metered)

	logLeaks("spigot", spigotLeaks)
	logLeaks("meter", meterLeaks)

	log.Printf("⏱️  Running meter example for %s...", meterRunDuration)
	time.Sleep(meterRunDuration)

	log.Println("🛑 Stopping meter example...")
	cancel()
	<-done

	log.Println("✅ Meter example finished")

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
		meterSpigotInterval,
	)

	numbers, leaks := spigot.Stream(ctx, nil)

	return numbers, leaks
}

// buildMeterStage wires a Meter that records simple rate and last-seen metrics.
func buildMeterStage(
	ctx context.Context,
	numbers <-chan int,
) (<-chan int, <-chan error) {
	var (
		count          int
		lastValue      int
		firstTimestamp time.Time
	)

	meter := plumber.NewMeter[int](
		"simple-meter",
		func(_ context.Context, value int) error {
			if count == 0 {
				firstTimestamp = time.Now()
			}

			count++
			lastValue = value

			// In a real application you could plug Prometheus or OpenTelemetry here, e.g.:
			//   requestCount.Inc()
			//   valueGauge.Set(float64(value))
			//   latencyHistogram.Observe(time.Since(firstTimestamp).Seconds())

			return nil
		},
	)

	metered, leaks := meter.Stream(ctx, numbers)

	// Periodically log a simple rate estimate from inside this example.
	go func() {
		interval := time.Duration(meterLogIntervalSecs) * time.Second

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if count == 0 {
					continue
				}

				elapsed := time.Since(firstTimestamp).Seconds()
				if elapsed <= 0 {
					continue
				}

				log.Printf("📈 meter: count=%d, last=%d, approx rate=%.2f values/sec", count, lastValue, float64(count)/elapsed)
			}
		}
	}()

	return metered, leaks
}

// buildMeterTap logs the values that flow through the meter.
func buildMeterTap(ctx context.Context, input <-chan int) <-chan struct{} {
	tap := plumber.NewTap[int](
		"meter-logger",
		func(_ context.Context, value int) error {
			log.Printf("📦 metered value: %d", value)

			return nil
		},
	)

	done, tapLeaks := tap.Stream(ctx, input)

	logLeaks("tap", tapLeaks)

	return done
}
