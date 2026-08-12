package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func alwaysRetriable(error) bool { return true }
func neverRetriable(error) bool  { return false }

func TestDo_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), alwaysRetriable, func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_NonRetriableStopsImmediately(t *testing.T) {
	calls := 0
	sentinel := errors.New("fatal")
	err := Do(context.Background(), neverRetriable, func() error {
		calls++
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, calls)
}

func TestDo_RetriesThenSucceeds(t *testing.T) {
	calls := 0
	err := Do(context.Background(), alwaysRetriable, func() error {
		calls++
		if calls < 2 {
			return errors.New("temporary")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, calls) // одна неудачная + одна успешная (после паузы 1s)
}

func TestDo_AlreadyCancelledContextSkipsCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := Do(ctx, alwaysRetriable, func() error {
		calls++
		return nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, calls)
}

func TestDo_ContextCancelledDuringDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sentinel := errors.New("temporary")

	calls := 0
	start := time.Now()
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, alwaysRetriable, func() error {
		calls++
		return sentinel
	})

	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, calls) // отмена наступила во время первой паузы, повтор не выполнен
	assert.Less(t, time.Since(start), time.Second)
}
