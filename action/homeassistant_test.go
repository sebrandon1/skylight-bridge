package action

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sebrandon1/skylight-bridge/engine"
)

func TestHomeAssistantService(t *testing.T) {
	var receivedPath string
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuth = r.Header.Get("Authorization")
		io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a, err := NewHomeAssistantAction(map[string]any{
		"url":       srv.URL,
		"token":     "my-ha-token",
		"service":   "light.turn_on",
		"entity_id": "light.living_room",
	})
	if err != nil {
		t.Fatalf("NewHomeAssistantAction: %v", err)
	}

	if err := a.Execute(context.Background(), engine.Event{Type: engine.EventChoreAllCompleted}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if receivedPath != "/api/services/light/turn_on" {
		t.Errorf("path = %q, want /api/services/light/turn_on", receivedPath)
	}
	if receivedAuth != "Bearer my-ha-token" {
		t.Errorf("auth = %q, want Bearer my-ha-token", receivedAuth)
	}
}

func TestHomeAssistantWebhook(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a, err := NewHomeAssistantAction(map[string]any{
		"url":        srv.URL,
		"webhook_id": "test-hook",
	})
	if err != nil {
		t.Fatalf("NewHomeAssistantAction: %v", err)
	}

	if err := a.Execute(context.Background(), engine.Event{Type: engine.EventChoreCompleted}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if receivedPath != "/api/webhook/test-hook" {
		t.Errorf("path = %q, want /api/webhook/test-hook", receivedPath)
	}
}

func TestHomeAssistantMissingURL(t *testing.T) {
	_, err := NewHomeAssistantAction(map[string]any{"service": "light.turn_on"})
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestHomeAssistantMissingServiceAndWebhook(t *testing.T) {
	_, err := NewHomeAssistantAction(map[string]any{"url": "http://localhost"})
	if err == nil {
		t.Fatal("expected error when neither service nor webhook_id is set")
	}
}

func TestHomeAssistantServiceError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	a, _ := NewHomeAssistantAction(map[string]any{
		"url":     srv.URL,
		"service": "light.turn_on",
	})
	err := a.Execute(context.Background(), engine.Event{})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
