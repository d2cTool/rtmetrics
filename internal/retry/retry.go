// Package retry повторяет функцию с паузами 1s, 3s, 5s, пока ошибка ретрабельна.
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

// Do вызывает fn. При ретрабельной ошибке ждёт и повторяет, пока ctx не отменён.
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
