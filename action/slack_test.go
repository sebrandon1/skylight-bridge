package action

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
)

func TestSlackAction(t *testing.T) {
	var receivedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a, err := NewSlackAction(map[string]any{"webhook_url": srv.URL})
	if err != nil {
		t.Fatalf("NewSlackAction: %v", err)
	}

	event := engine.Event{
		Type:      engine.EventChoreCompleted,
		Timestamp: time.Now(),
		Data: map[string]any{
			"assignee_name": "Alice",
			"chore_title":   "Clean room",
		},
	}

	if err := a.Execute(context.Background(), event); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Slack uses "text" key (not "content" like Discord).
	if receivedBody["text"] != "**Alice** completed **Clean room**" {
		t.Errorf("text = %q, want '**Alice** completed **Clean room**'", receivedBody["text"])
	}
}

func TestSlackActionCustomTemplate(t *testing.T) {
	var receivedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a, err := NewSlackAction(map[string]any{
		"webhook_url": srv.URL,
		"message":     "{{.child_name}} got {{.points}} pts",
	})
	if err != nil {
		t.Fatalf("NewSlackAction: %v", err)
	}

	event := engine.Event{
		Type: engine.EventRewardRedeemed,
		Data: map[string]any{"child_name": "Bob", "points": 50},
	}

	if err := a.Execute(context.Background(), event); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if receivedBody["text"] != "Bob got 50 pts" {
		t.Errorf("text = %q, want 'Bob got 50 pts'", receivedBody["text"])
	}
}

func TestSlackActionMissingURL(t *testing.T) {
	_, err := NewSlackAction(map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing webhook_url")
	}
}

func TestSlackActionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	a, _ := NewSlackAction(map[string]any{"webhook_url": srv.URL})
	err := a.Execute(context.Background(), engine.Event{
		Type: engine.EventChoreCompleted,
		Data: map[string]any{"assignee_name": "X", "chore_title": "Y"},
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}
