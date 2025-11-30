package plumber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/brunorene/go-plumber/plumber"
)

func TestBasicValve_Stream_PassesThroughValues(t *testing.T) {
	t.Helper()

	ctx := t.Context()

	inputCh := make(chan int)

	valve := plumber.NewValve[int](
		"valve",
		10*time.Millisecond,
	)

	outCh, leakCh := valve.Stream(ctx, inputCh)

	go func() {
		defer close(inputCh)

		for index := range 3 {
			inputCh <- index
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

	assert.ElementsMatch(t, []int{0, 1, 2}, outputs)
	assert.Empty(t, leaks)
}
