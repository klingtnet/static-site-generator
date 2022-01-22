// Package distribute provides concurrency primitives, like limited distribution of work.
package distribute

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

var ErrNotEnoughConcurrency = fmt.Errorf("concurrency must be greater than zero")

// OneToN distributes work to a limited number of worker functions.
// Data for the workers is provided by sourceFn and distributed to
// concurrency amount of workerFn functions.
// Note that concurrency must be greater than zero.
func OneToN(
	ctx context.Context,
	sourceFn func(ctx context.Context, dataCh chan<- interface{}) error,
	workerFn func(ctx context.Context, data interface{}) error,
	concurrency int,
) error {
	if concurrency < 1 {
		return ErrNotEnoughConcurrency
	}

	ch := make(chan interface{}, concurrency)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer close(ch)
		return sourceFn(ctx, ch)
	})
	for i := 0; i < concurrency; i++ {
		eg.Go(func() error {
			for data := range ch {
				err := workerFn(ctx, data)
				if err != nil {
					return err
				}
			}

			return ctx.Err()
		})
	}

	return eg.Wait()
}
