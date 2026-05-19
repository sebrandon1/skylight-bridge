package rules

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/sebrandon1/skylight-bridge/action"
	"github.com/sebrandon1/skylight-bridge/config"
	"github.com/sebrandon1/skylight-bridge/engine"
)

func TestEngineMatchesEvent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// We need to use a factory that returns a trackable action.
	var executed bool
	factory := func(_ map[string]any) (action.Action, error) {
		return &trackAction{executed: &executed}, nil
	}

	eng, err := NewEngine([]config.RuleConfig{
		{
			Name:  "test",
			Event: "chore.completed",
			Actions: []config.ActionConfig{
				{Type: "mock"},
			},
		},
	}, map[string]action.Factory{"mock": factory}, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	event := engine.Event{
		Type:      engine.EventChoreCompleted,
		Timestamp: time.Now(),
		Data:      map[string]any{"chore_title": "Clean room"},
	}

	eng.HandleEvent(context.Background(), event)
	if !executed {
		t.Error("expected action to be executed")
	}
}

type trackAction struct {
	executed *bool
}

func (a *trackAction) Execute(_ context.Context, _ engine.Event) error {
	*a.executed = true
	return nil
}

func TestEngineFilters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	var executed bool
	factory := func(_ map[string]any) (action.Action, error) {
		return &trackAction{executed: &executed}, nil
	}

	eng, err := NewEngine([]config.RuleConfig{
		{
			Name:    "filtered",
			Event:   "reward.redeemed",
			Filters: map[string]string{"reward_title": "Invest $20 in VOO"},
			Actions: []config.ActionConfig{{Type: "mock"}},
		},
	}, map[string]action.Factory{"mock": factory}, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Non-matching event.
	eng.HandleEvent(context.Background(), engine.Event{
		Type: engine.EventRewardRedeemed,
		Data: map[string]any{"reward_title": "Ice cream"},
	})
	if executed {
		t.Error("action should not execute for non-matching filter")
	}

	// Matching event.
	eng.HandleEvent(context.Background(), engine.Event{
		Type: engine.EventRewardRedeemed,
		Data: map[string]any{"reward_title": "Invest $20 in VOO"},
	})
	if !executed {
		t.Error("action should execute for matching filter")
	}
}

func TestEngineWrongEventType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	var executed bool
	factory := func(_ map[string]any) (action.Action, error) {
		return &trackAction{executed: &executed}, nil
	}

	eng, err := NewEngine([]config.RuleConfig{
		{
			Name:    "chore-only",
			Event:   "chore.completed",
			Actions: []config.ActionConfig{{Type: "mock"}},
		},
	}, map[string]action.Factory{"mock": factory}, logger)
	if err != nil {
		t.Fatal(err)
	}

	eng.HandleEvent(context.Background(), engine.Event{
		Type: engine.EventRewardRedeemed,
		Data: map[string]any{},
	})
	if executed {
		t.Error("action should not execute for wrong event type")
	}
}

func TestParseRetryDelay(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"2s", 2 * time.Second},
		{"500ms", 500 * time.Millisecond},
		{"", time.Second},
		{"invalid", time.Second},
		{"-1s", time.Second},
	}
	for _, tt := range tests {
		got := parseRetryDelay(tt.input)
		if got != tt.want {
			t.Errorf("parseRetryDelay(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEngineRetryOnFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	calls := 0
	factory := func(_ map[string]any) (action.Action, error) {
		return &countFailAction{successAfter: 2, calls: &calls}, nil
	}

	eng, err := NewEngine([]config.RuleConfig{
		{
			Name:  "retried",
			Event: "chore.completed",
			Actions: []config.ActionConfig{
				{Type: "mock", RetryAttempts: 3, RetryDelay: "1ms"},
			},
		},
	}, map[string]action.Factory{"mock": factory}, logger)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	eng.HandleEvent(context.Background(), engine.Event{
		Type: engine.EventChoreCompleted,
		Data: map[string]any{},
	})

	if calls != 3 {
		t.Errorf("calls = %d, want 3 (2 failures + 1 success)", calls)
	}
}

type countFailAction struct {
	calls        *int
	successAfter int
}

func (a *countFailAction) Execute(_ context.Context, _ engine.Event) error {
	*a.calls++
	if *a.calls > a.successAfter {
		return nil
	}
	return fmt.Errorf("transient error on attempt %d", *a.calls)
}

func TestEngineUnknownActionType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	_, err := NewEngine([]config.RuleConfig{
		{
			Name:    "bad",
			Event:   "chore.completed",
			Actions: []config.ActionConfig{{Type: "nonexistent"}},
		},
	}, map[string]action.Factory{}, logger)
	if err == nil {
		t.Fatal("expected error for unknown action type")
	}
}
