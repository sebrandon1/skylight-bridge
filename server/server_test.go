package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
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
