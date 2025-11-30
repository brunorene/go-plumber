package plumber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brunorene/go-plumber/plumber"
)

// TestStream_PipeChain_Success verifies that the streaming layer can be used
// directly (without a Pipeline) by wiring BasicPipe and BasicTap via channels.
func TestStream_PipeChain_Success(t *testing.T) {
	ctx := t.Context()

	inputCh := make(chan int)

	var outputs []int

	tap := plumber.NewTap[int]("collector", func(_ context.Context, v int) error {
		outputs = append(outputs, v)

		return nil
	})

	increment := plumber.NewPipe[int, int](
		"incrementer",
		func(_ context.Context, v int) (int, error) {
			return v + 1, nil
		},
	)

	doubler := plumber.NewPipe[int, int](
		"doubler",
		func(_ context.Context, v int) (int, error) {
			return v * 2, nil
		},
	)

	doubledOut, _ := doubler.Stream(ctx, inputCh)
	incrementedOut, _ := increment.Stream(ctx, doubledOut)
	done, _ := tap.Stream(ctx, incrementedOut)

	go func() {
		defer close(inputCh)

		for _, v := range []int{1, 2, 3} {
			inputCh <- v
		}
	}()

	<-done

	require.Equal(t, []int{3, 5, 7}, outputs)
}
