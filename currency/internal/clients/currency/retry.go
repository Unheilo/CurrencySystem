package currency

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

func (p RetryPolicy) Do(ctx context.Context, fn func(attempt int) error) error {
	var lastErr error
	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := fn(attempt)
		if err == nil {
			return nil
		}
		lastErr = err

		var re *retriableError
		if !errors.As(err, &re) {
			return err
		}

		if attempt == p.MaxAttempts-1 {
			break
		}

		delay := p.backoff(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("retries exhausted (%d atttempts): %w", p.MaxAttempts, lastErr)
}

func (p RetryPolicy) backoff(attempt int) time.Duration {
	// base * 2^attempt
	d := p.BaseDelay * (1 << attempt)
	if d > p.MaxDelay {
		d = p.MaxDelay
	}

	// jitter +/- 30%
	jitter := rand.Float64()*0.6 - 0.3
	return time.Duration(float64(d) * (1 + jitter))
}

// retriableError is a wrapper for errors that should trigger a retry.
type retriableError struct{ Err error }

func (e *retriableError) Error() string { return "retiable: " + e.Err.Error() }
func (e *retriableError) Unwrap() error { return e.Err }

func Retriable(err error) error { return &retriableError{Err: err} }
