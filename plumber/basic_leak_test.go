package plumber_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/brunorene/go-plumber/plumber"
)

func TestBasicPipe_LeakContext(t *testing.T) {
	pipe := plumber.NewPipe[int, int](
		"doubler",
		func(_ context.Context, _ int) (int, error) {
			return 0, errors.New("boom from pipe")
		},
	)

	_, err := pipe.Process(context.Background(), 21)
	require.ErrorIs(t, err, plumber.ErrPipe)
	assert.Contains(t, err.Error(), "doubler")
	assert.Contains(t, err.Error(), "boom from pipe")
}

func TestBasicPipe_TypeChangingLeakContext(t *testing.T) {
	pipe := plumber.NewPipe[int, string](
		"int-to-string",
		func(_ context.Context, _ int) (string, error) {
			return "", errors.New("boom from pipe")
		},
	)

	_, err := pipe.Process(context.Background(), 42)
	require.ErrorIs(t, err, plumber.ErrPipe)
	assert.Contains(t, err.Error(), "int-to-string")
	assert.Contains(t, err.Error(), "boom from pipe")
}

func TestBasicTap_LeakContext(t *testing.T) {
	tap := plumber.NewTap[int](
		"failing-tap",
		func(_ context.Context, _ int) error {
			return errors.New("boom from tap")
		},
	)

	err := tap.Drain(context.Background(), 7)
	require.ErrorIs(t, err, plumber.ErrTap)
	assert.Contains(t, err.Error(), "failing-tap")
	assert.Contains(t, err.Error(), "boom from tap")
}

func TestBasicSpigot_LeakContext(t *testing.T) {
	genErr := errors.New("boom from spigot")

	spigot := plumber.NewSpigot[int](
		"number-generator",
		func(context.Context) (int, error) {
			return 0, genErr
		},
		1*time.Millisecond,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, leaks := spigot.Stream(ctx, nil)

	select {
	case err := <-leaks:
		require.ErrorIs(t, err, plumber.ErrSpigot)
		require.ErrorIs(t, err, genErr)
		assert.Contains(t, err.Error(), "number-generator")
	case <-ctx.Done():
		t.Fatal("expected spigot leak but context timed out")
	}
}
