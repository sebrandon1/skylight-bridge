package action

import (
	"context"
	"time"

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
	delay := r.initialDelay
	var err error
	for attempt := range r.maxAttempts {
		if attempt > 0 {
			t := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
			delay = min(delay*2, maxRetryDelay)
		}
		if err = r.inner.Execute(ctx, event); err == nil {
			return nil
		}
	}
	return err
}
