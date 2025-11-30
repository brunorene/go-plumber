package plumber_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/brunorene/go-plumber/plumber"
)

// TestBasicSplitTee_Stream_FansOutValues verifies that BasicSplitTee broadcasts
// each incoming value to all output branches.
func TestBasicSplitTee_Stream_FansOutValues(t *testing.T) {
	ctx := context.Background()

	inputCh := make(chan int)

	splitTee := plumber.NewSplitTee[int](
		"splitter",
		3,
	)

	outs, leakCh := splitTee.Stream(ctx, inputCh)

	assert.Len(t, outs, 3)

	var (
		results     []int
		collectDone = make(chan struct{})
	)

	go func() {
		defer close(collectDone)

		for _, ch := range outs {
			v := <-ch
			results = append(results, v)
		}
	}()

	go func() {
		// Drain leaks to avoid goroutine leaks; BasicSplitTee does not currently
		// emit any leaks.
		for leak := range leakCh {
			_ = leak
		}
	}()

	inputCh <- 10

	close(inputCh)

	<-collectDone

	assert.ElementsMatch(t, []int{10, 10, 10}, results)
}
