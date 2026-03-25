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

func TestWebhookAction(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a, err := NewWebhookAction(map[string]any{
		"url": srv.URL,
		"headers": map[string]any{
			"X-Custom": "test-value",
		},
	})
	if err != nil {
		t.Fatalf("NewWebhookAction: %v", err)
	}

	event := engine.Event{
		Type:      engine.EventRewardRedeemed,
		Timestamp: time.Now(),
		Data:      map[string]any{"reward_title": "Invest $20"},
	}

	if err := a.Execute(context.Background(), event); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if receivedHeaders.Get("X-Custom") != "test-value" {
		t.Errorf("custom header = %q, want test-value", receivedHeaders.Get("X-Custom"))
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("content-type = %q, want application/json", receivedHeaders.Get("Content-Type"))
	}

	var parsed engine.Event
	if err := json.Unmarshal(receivedBody, &parsed); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if parsed.Type != engine.EventRewardRedeemed {
		t.Errorf("type = %q, want reward.redeemed", parsed.Type)
	}
}

func TestWebhookActionWithTemplate(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a, err := NewWebhookAction(map[string]any{
		"url":           srv.URL,
		"body_template": `{"kid": "{{.child_name}}", "amount": {{.points}}}`,
	})
	if err != nil {
		t.Fatalf("NewWebhookAction: %v", err)
	}

	event := engine.Event{
		Type: engine.EventRewardRedeemed,
		Data: map[string]any{"child_name": "Alice", "points": 100},
	}

	if err := a.Execute(context.Background(), event); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	expected := `{"kid": "Alice", "amount": 100}`
	if receivedBody != expected {
		t.Errorf("body = %q, want %q", receivedBody, expected)
	}
}

func TestWebhookActionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a, _ := NewWebhookAction(map[string]any{"url": srv.URL})
	err := a.Execute(context.Background(), engine.Event{Type: engine.EventChoreCompleted})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestWebhookActionMissingURL(t *testing.T) {
	_, err := NewWebhookAction(map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}
