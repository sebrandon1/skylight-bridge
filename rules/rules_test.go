package rules

import (
	"context"
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
