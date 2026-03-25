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

func TestDiscordAction(t *testing.T) {
	var receivedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &receivedBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a, err := NewDiscordAction(map[string]any{
		"webhook_url": srv.URL,
	})
	if err != nil {
		t.Fatalf("NewDiscordAction: %v", err)
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

	if receivedBody["content"] != "**Alice** completed **Clean room**" {
		t.Errorf("content = %q, want '**Alice** completed **Clean room**'", receivedBody["content"])
	}
}

func TestDiscordActionCustomTemplate(t *testing.T) {
	var receivedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &receivedBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a, err := NewDiscordAction(map[string]any{
		"webhook_url": srv.URL,
		"message":     "{{.child_name}} got {{.points}} pts",
	})
	if err != nil {
		t.Fatalf("NewDiscordAction: %v", err)
	}

	event := engine.Event{
		Type: engine.EventRewardRedeemed,
		Data: map[string]any{"child_name": "Bob", "points": 50},
	}

	if err := a.Execute(context.Background(), event); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if receivedBody["content"] != "Bob got 50 pts" {
		t.Errorf("content = %q, want 'Bob got 50 pts'", receivedBody["content"])
	}
}

func TestDiscordActionMissingURL(t *testing.T) {
	_, err := NewDiscordAction(map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing webhook_url")
	}
}

func TestDiscordActionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	a, _ := NewDiscordAction(map[string]any{"webhook_url": srv.URL})
	err := a.Execute(context.Background(), engine.Event{Type: engine.EventChoreCompleted, Data: map[string]any{"assignee_name": "X", "chore_title": "Y"}})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestDiscordDefaultMessages(t *testing.T) {
	var received []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &body)
		received = append(received, body["content"])
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a, _ := NewDiscordAction(map[string]any{"webhook_url": srv.URL})

	events := []engine.Event{
		{Type: engine.EventChoreAllCompleted, Data: map[string]any{"assignee_name": "Alice", "chore_count": 3, "total_points": 10}},
		{Type: engine.EventRewardRedeemed, Data: map[string]any{"child_name": "Bob", "reward_title": "Ice cream", "points": 50}},
	}

	for _, e := range events {
		a.Execute(context.Background(), e)
	}

	if received[0] != "**Alice** finished all chores for today! (3 chores, 10 points)" {
		t.Errorf("all_completed msg = %q", received[0])
	}
	if received[1] != "**Bob** redeemed **Ice cream** (50 points)" {
		t.Errorf("redeemed msg = %q", received[1])
	}
}
