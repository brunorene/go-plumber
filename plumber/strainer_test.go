package plumber_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/brunorene/go-plumber/plumber"
)

func TestBasicStrainer_Stream_FiltersValues(t *testing.T) {
	t.Helper()

	ctx := t.Context()

	inputCh := make(chan int)

	strainer := plumber.NewStrainer[int](
		"even-only",
		func(_ context.Context, value int) (bool, error) {
			return value%2 == 0, nil
		},
	)

	outCh, leakCh := strainer.Stream(ctx, inputCh)

	go func() {
		defer close(inputCh)

		for value := 1; value <= 5; value++ {
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

	assert.ElementsMatch(t, []int{2, 4}, outputs)
	assert.Empty(t, leaks)
}

func TestBasicStrainer_Stream_LeakesOnPredicateError(t *testing.T) {
	t.Helper()

	ctx := t.Context()

	inputCh := make(chan int)

	predicateErr := errors.New("boom from predicate")

	strainer := plumber.NewStrainer[int](
		"failing-strainer",
		func(_ context.Context, value int) (bool, error) {
			if value < 0 {
				return false, predicateErr
			}

			return true, nil
		},
	)

	outCh, leakCh := strainer.Stream(ctx, inputCh)

	go func() {
		defer close(inputCh)

		inputCh <- -1

		inputCh <- 1
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
	require.ErrorIs(t, leaks[0], plumber.ErrStrainer)
	require.ErrorIs(t, leaks[0], predicateErr)
	assert.Contains(t, leaks[0].Error(), "failing-strainer")
	assert.ElementsMatch(t, []int{1}, outputs)
}

func TestBasicStrainer_Stream_LeakesOnMissingPredicate(t *testing.T) {
	t.Helper()

	ctx := t.Context()

	inputCh := make(chan int)

	strainer := plumber.NewStrainer[int]("no-predicate", nil)

	outCh, leakCh := strainer.Stream(ctx, inputCh)

	go func() {
		defer close(inputCh)

		inputCh <- 1
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
	require.ErrorIs(t, leaks[0], plumber.ErrStrainer)
	assert.Empty(t, outputs)
}
