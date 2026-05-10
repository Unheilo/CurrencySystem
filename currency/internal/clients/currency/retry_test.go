package currency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryPolicy_RetriesUntilSuccess(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}

	calls := 0
	err := p.Do(context.Background(), func(attempt int) error {
		calls++
		if calls < 3 {
			return Retriable(errors.New("boom"))
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryPolicy_NonRetriableFailsFast(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 5, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}

	calls := 0
	err := p.Do(context.Background(), func(attempt int) error {
		calls++
		return errors.New("boom")
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryPolicy_RetriesExhausted(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 10 * time.Millisecond}

	calls := 0
	err := p.Do(context.Background(), func(attempt int) error {
		calls++
		return Retriable(errors.New("boom"))
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retries exhausted")
	assert.Equal(t, 3, calls)
}

func TestRetryPolicy_ContextCanceled(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 10, BaseDelay: 50 * time.Millisecond, MaxDelay: 1 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := p.Do(ctx, func(attempt int) error {
		calls++
		return Retriable(errors.New("boom"))
	})
	require.ErrorIs(t, err, context.Canceled)
	assert.LessOrEqual(t, calls, 1)
}
