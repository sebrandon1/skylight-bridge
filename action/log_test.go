package action

import (
	"context"
	"testing"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
)

func TestLogAction(t *testing.T) {
	a, err := NewLogAction(map[string]any{})
	if err != nil {
		t.Fatalf("NewLogAction: %v", err)
	}

	event := engine.Event{
		Type:      engine.EventChoreCompleted,
		Timestamp: time.Now(),
		Data:      map[string]any{"chore_title": "Clean room"},
	}

	if err := a.Execute(context.Background(), event); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestLogActionWithTemplate(t *testing.T) {
	a, err := NewLogAction(map[string]any{
		"message": "{{.assignee_name}} completed {{.chore_title}}",
	})
	if err != nil {
		t.Fatalf("NewLogAction: %v", err)
	}

	event := engine.Event{
		Type: engine.EventChoreCompleted,
		Data: map[string]any{"assignee_name": "Alice", "chore_title": "Clean room"},
	}

	if err := a.Execute(context.Background(), event); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestLogActionInvalidTemplate(t *testing.T) {
	_, err := NewLogAction(map[string]any{
		"message": "{{.invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}
