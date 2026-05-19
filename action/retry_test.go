package action

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// countingAction counts calls and fails until successAfter attempts.
type countingAction struct {
	calls        int
	successAfter int
	err          error
}

func (c *countingAction) Execute(_ context.Context, _ engine.Event) error {
	c.calls++
	if c.calls > c.successAfter {
		return nil
	}
	return c.err
}

func TestRetrySucceedsOnFirstAttempt(t *testing.T) {
	inner := &countingAction{successAfter: 0, err: errors.New("fail")}
	a := WrapRetry(inner, 3, time.Millisecond)

	if err := a.Execute(context.Background(), engine.Event{}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if inner.calls != 1 {
		t.Errorf("calls = %d, want 1", inner.calls)
	}
}

func TestRetrySucceedsAfterFailures(t *testing.T) {
	inner := &countingAction{successAfter: 2, err: errors.New("transient")}
	a := WrapRetry(inner, 3, time.Millisecond)

	if err := a.Execute(context.Background(), engine.Event{}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if inner.calls != 3 {
		t.Errorf("calls = %d, want 3", inner.calls)
	}
}

func TestRetryExhausted(t *testing.T) {
	inner := &countingAction{successAfter: 99, err: errors.New("permanent")}
	a := WrapRetry(inner, 3, time.Millisecond)

	err := a.Execute(context.Background(), engine.Event{})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if inner.calls != 3 {
		t.Errorf("calls = %d, want 3", inner.calls)
	}
}

func TestRetryRespectsContextCancellation(t *testing.T) {
	inner := &countingAction{successAfter: 99, err: errors.New("fail")}
	// Use a long initial delay so ctx cancellation fires first.
	a := WrapRetry(inner, 5, 10*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := a.Execute(ctx, engine.Event{})
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
}
