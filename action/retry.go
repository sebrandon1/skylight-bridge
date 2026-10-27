package action

import (
	"context"
	"time"

	"github.com/sebrandon1/go-toolbox/retry"

	"github.com/sebrandon1/skylight-bridge/engine"
)

const maxRetryDelay = 30 * time.Second

type retryAction struct {
	inner        Action
	maxAttempts  int
	initialDelay time.Duration
}

// WrapRetry wraps an Action with exponential backoff retry.
// maxAttempts is the total number of attempts (1 = no retry).
// initialDelay is the wait before the second attempt; each subsequent wait doubles,
// capped at 30s.
func WrapRetry(inner Action, maxAttempts int, initialDelay time.Duration) Action {
	return &retryAction{inner: inner, maxAttempts: maxAttempts, initialDelay: initialDelay}
}

func (r *retryAction) Execute(ctx context.Context, event engine.Event) error {
	// Preserve the historical no-op behavior for a non-positive attempt budget.
	if r.maxAttempts <= 0 {
		return nil
	}

	delay := r.initialDelay
	if delay <= 0 {
		// retry substitutes its 1s default for non-positive delays; use a
		// negligible delay instead so a configured zero stays immediate.
		delay = time.Nanosecond
	}

	return retry.Retry(ctx, func() error {
		return r.inner.Execute(ctx, event)
	},
		retry.WithMaxRetries(r.maxAttempts-1),
		retry.WithInitialDelay(delay),
		retry.WithMaxDelay(maxRetryDelay),
	)
}
