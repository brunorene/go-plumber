package plumber_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/brunorene/go-plumber/plumber"
)

func TestBasicJoinTee_Join_SumsBatch(t *testing.T) {
	joinTee := plumber.NewJoinTee[int, int](
		"sum-join",
		func(_ context.Context, batch []int) (int, error) {
			var sum int
			for _, v := range batch {
				sum += v
			}

			return sum, nil
		},
	)

	ctx := context.Background()

	out, err := joinTee.Join(ctx, []int{1, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, 6, out)
}

func TestBasicJoinTee_Join_ErrorWrapping(t *testing.T) {
	cause := errors.New("boom from joiner")

	joinTee := plumber.NewJoinTee[int, int](
		"failing-join",
		func(_ context.Context, _ []int) (int, error) {
			return 0, cause
		},
	)

	ctx := context.Background()

	_, err := joinTee.Join(ctx, []int{1})
	require.ErrorIs(t, err, plumber.ErrJoinTee)
	require.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), "failing-join")
}

func TestBasicJoinTee_Stream_ProcessesBatches(t *testing.T) {
	ctx := context.Background()

	joinTee := plumber.NewJoinTee[int, int](
		"sum-join-stream",
		func(_ context.Context, batch []int) (int, error) {
			var sum int
			for _, v := range batch {
				sum += v
			}

			return sum, nil
		},
	)

	in1 := make(chan int)
	in2 := make(chan int)

	outCh, leakCh := joinTee.Stream(ctx, []<-chan int{in1, in2})

	go func() {
		defer close(in1)
		defer close(in2)

		in1 <- 1

		in2 <- 2

		in1 <- 3

		in2 <- 4
	}()

	var (
		results []int
		leaks   []error
	)

	for outCh != nil || leakCh != nil {
		select {
		case v, ok := <-outCh:
			if !ok {
				outCh = nil

				continue
			}

			results = append(results, v)
		case err, ok := <-leakCh:
			if !ok {
				leakCh = nil

				continue
			}

			leaks = append(leaks, err)
		}
	}

	require.Len(t, results, 2)
	assert.ElementsMatch(t, []int{3, 7}, results)
	assert.Empty(t, leaks)
}
