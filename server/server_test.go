package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sebrandon1/skylight-bridge/action"
	"github.com/sebrandon1/skylight-bridge/config"
	"github.com/sebrandon1/skylight-bridge/engine"
	"github.com/sebrandon1/skylight-bridge/rules"
)

func TestHealthz(t *testing.T) {
	srv := New(10)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
}

func TestEvents(t *testing.T) {
	srv := New(10)
	srv.RecordEvent(engine.Event{
		Type:      engine.EventChoreCompleted,
		Timestamp: time.Now(),
		Data:      map[string]any{"chore_title": "Clean"},
	})
	srv.RecordEvent(engine.Event{
		Type:      engine.EventRewardRedeemed,
		Timestamp: time.Now(),
		Data:      map[string]any{"reward_title": "Invest"},
	})

	handler := srv.Handler()

	// All events.
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp struct {
		Events []engine.Event `json:"events"`
		Count  int            `json:"count"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
}

func TestEventsFilterByType(t *testing.T) {
	srv := New(10)
	srv.RecordEvent(engine.Event{Type: engine.EventChoreCompleted, Timestamp: time.Now()})
	srv.RecordEvent(engine.Event{Type: engine.EventRewardRedeemed, Timestamp: time.Now()})
	srv.RecordEvent(engine.Event{Type: engine.EventChoreCompleted, Timestamp: time.Now()})

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/events?type=chore.completed", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp struct {
		Count int `json:"count"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
}

func TestEventsLimit(t *testing.T) {
	srv := New(10)
	for i := range 5 {
		srv.RecordEvent(engine.Event{
			Type:      engine.EventChoreCompleted,
			Timestamp: time.Now(),
			Data:      map[string]any{"i": i},
		})
	}

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/events?limit=2", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp struct {
		Events []engine.Event `json:"events"`
		Count  int            `json:"count"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 2 {
		t.Errorf("count = %d, want 2", resp.Count)
	}
}

func TestRulesNotWired(t *testing.T) {
	srv := New(10)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/rules", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestRules(t *testing.T) {
	re, err := rules.NewEngine(
		[]config.RuleConfig{
			{
				Name:    "test-rule",
				Event:   "chore.completed",
				Filters: map[string]string{"assignee_name": "Alice"},
				Actions: []config.ActionConfig{{Type: "log"}},
			},
		},
		map[string]action.Factory{"log": action.NewLogAction},
		slog.Default(),
	)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	srv := New(10)
	srv.SetRulesEngine(re)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/rules", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Rules []rules.RuleInfo `json:"rules"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Rules) != 1 {
		t.Fatalf("len(rules) = %d, want 1", len(resp.Rules))
	}
	r := resp.Rules[0]
	if r.Name != "test-rule" {
		t.Errorf("name = %q, want test-rule", r.Name)
	}
	if r.Event != "chore.completed" {
		t.Errorf("event = %q, want chore.completed", r.Event)
	}
	if len(r.ActionTypes) != 1 || r.ActionTypes[0] != "log" {
		t.Errorf("actions = %v, want [log]", r.ActionTypes)
	}
}

func TestStatsNotWired(t *testing.T) {
	srv := New(10)
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestRingBufferOverflow(t *testing.T) {
	srv := New(3)
	for i := range 5 {
		srv.RecordEvent(engine.Event{
			Type: engine.EventChoreCompleted,
			Data: map[string]any{"i": i},
		})
	}

	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp struct {
		Events []engine.Event `json:"events"`
		Count  int            `json:"count"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 3 {
		t.Errorf("count = %d, want 3 (buffer size)", resp.Count)
	}
	// Should have the last 3 events (i=2,3,4).
	first := resp.Events[0].Data["i"]
	if first != float64(2) {
		t.Errorf("first event i = %v, want 2", first)
	}
}
