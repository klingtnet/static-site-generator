package distribute

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOneToN(t *testing.T) {
	t.Run("concurrency below zero", func(t *testing.T) {
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				return nil
			},
			func(ctx context.Context, data interface{}) error {
				return nil
			},
			-1,
		)
		require.ErrorIs(t, err, ErrNotEnoughConcurrency)
	})

	t.Run("concurrency is zero", func(t *testing.T) {
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				return nil
			},
			func(ctx context.Context, data interface{}) error {
				return nil
			},
			0,
		)
		require.ErrorIs(t, err, ErrNotEnoughConcurrency)
	})

	t.Run("sourceFn fails", func(t *testing.T) {
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				return fmt.Errorf("sourceFn failed")
			},
			func(ctx context.Context, data interface{}) error {
				return nil
			},
			1,
		)
		require.EqualError(t, err, "sourceFn failed")
	})

	t.Run("workerFn fails", func(t *testing.T) {
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				dataCh <- "we need some data, otherwise workerFn is never called"
				return nil
			},
			func(ctx context.Context, data interface{}) error {
				return fmt.Errorf("workerFn failed")
			},
			1,
		)
		require.EqualError(t, err, "workerFn failed")
	})

	t.Run("sourceFn and workerFn fails", func(t *testing.T) {
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				dataCh <- "we need some data, otherwise workerFn is never called"
				return fmt.Errorf("sourceFn failed")
			},
			func(ctx context.Context, data interface{}) error {
				return fmt.Errorf("workerFn failed")
			},
			1,
		)
		require.Error(t, err)
		require.Contains(t, []string{"sourceFn failed", "workerFn failed"}, err.Error())
	})

	t.Run("concurrency of one", func(t *testing.T) {
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				dataCh <- "concurrency of one is serial"
				return nil
			},
			func(ctx context.Context, data interface{}) error {
				require.Equal(t, "concurrency of one is serial", data.(string))
				return nil
			},
			1,
		)
		require.NoError(t, err)
	})

	t.Run("more workers than data", func(t *testing.T) {
		results := make([]int, 8)
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				for i := 0; i < 8; i++ {
					dataCh <- i
				}
				return nil
			},
			func(ctx context.Context, data interface{}) error {
				idx := data.(int)
				results[idx] = idx + 1
				return nil
			},
			16,
		)
		require.NoError(t, err)
		require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, results)
	})

	t.Run("equal amount of data and workers", func(t *testing.T) {
		N := 16
		results := make([]int, N)
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				for i := 0; i < N; i++ {
					dataCh <- i
				}
				return nil
			},
			func(ctx context.Context, data interface{}) error {
				idx := data.(int)
				results[idx] = idx + 1
				return nil
			},
			16,
		)
		require.NoError(t, err)
		require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, results)
	})

	t.Run("more data than workers", func(t *testing.T) {
		var sum int64
		N := 1000
		err := OneToN(
			context.Background(),
			func(ctx context.Context, dataCh chan<- interface{}) error {
				for i := 1; i <= N; i++ {
					dataCh <- int64(i)
				}
				return nil
			},
			func(ctx context.Context, data interface{}) error {
				atomic.AddInt64(&sum, data.(int64))
				return nil
			},
			16,
		)
		require.NoError(t, err)
		gaussSum := int((N*N + N) / 2)
		require.Equal(t, int64(gaussSum), sum)
	})
}
