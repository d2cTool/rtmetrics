package retry

import (
	"context"
	"time"
)

var defaultDelays = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

func Do(ctx context.Context, isRetriable func(error) bool, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	err := fn()
	if err == nil || !isRetriable(err) {
		return err
	}

	for _, delay := range defaultDelays {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return err
		case <-timer.C:
		}

		err = fn()
		if err == nil || !isRetriable(err) {
			return err
		}
	}

	return err
}
