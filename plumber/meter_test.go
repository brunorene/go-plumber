package plumber_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/brunorene/go-plumber/plumber"
)

func TestBasicMeter_Stream_MeasuresAndPassesThrough(t *testing.T) {
	t.Helper()

	ctx := t.Context()

	inputCh := make(chan int)

	var measured []int

	meter := plumber.NewMeter[int](
		"meter",
		func(_ context.Context, value int) error {
			measured = append(measured, value)

			return nil
		},
	)

	outCh, leakCh := meter.Stream(ctx, inputCh)

	go func() {
		defer close(inputCh)

		for value := 1; value <= 3; value++ {
			inputCh <- value
		}
	}()

	var (
		outputs []int
		leaks   []error
	)

	for outCh != nil || leakCh != nil {
		select {
		case value, ok := <-outCh:
			if !ok {
				outCh = nil

				continue
			}

			outputs = append(outputs, value)
		case err, ok := <-leakCh:
			if !ok {
				leakCh = nil

				continue
			}

			leaks = append(leaks, err)
		}
	}

	assert.ElementsMatch(t, []int{1, 2, 3}, outputs)
	assert.ElementsMatch(t, []int{1, 2, 3}, measured)
	assert.Empty(t, leaks)
}

func TestBasicMeter_Stream_LeakesOnError(t *testing.T) {
	t.Helper()

	ctx := t.Context()

	inputCh := make(chan int)

	errMeasure := errors.New("boom from meter")

	meter := plumber.NewMeter[int](
		"failing-meter",
		func(_ context.Context, value int) error {
			if value == 2 {
				return errMeasure
			}

			return nil
		},
	)

	outCh, leakCh := meter.Stream(ctx, inputCh)

	go func() {
		defer close(inputCh)

		for value := 1; value <= 3; value++ {
			inputCh <- value
		}
	}()

	var (
		outputs []int
		leaks   []error
	)

	for outCh != nil || leakCh != nil {
		select {
		case value, ok := <-outCh:
			if !ok {
				outCh = nil

				continue
			}

			outputs = append(outputs, value)
		case err, ok := <-leakCh:
			if !ok {
				leakCh = nil

				continue
			}

			leaks = append(leaks, err)
		}
	}

	require.Len(t, leaks, 1)
	require.ErrorIs(t, leaks[0], plumber.ErrMeter)
	require.ErrorIs(t, leaks[0], errMeasure)
	assert.Contains(t, leaks[0].Error(), "failing-meter")
	assert.ElementsMatch(t, []int{1, 3}, outputs)
}
