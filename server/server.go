package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// Server provides HTTP endpoints for event history and health checks.
type Server struct {
	mu      sync.RWMutex
	events  []engine.Event
	maxSize int
	startAt time.Time
}

// New creates a Server with a ring buffer of the given capacity.
func New(bufferSize int) *Server {
	return &Server{
		events:  make([]engine.Event, 0, bufferSize),
		maxSize: bufferSize,
		startAt: time.Now(),
	}
}

// RecordEvent adds an event to the ring buffer. Safe for concurrent use.
func (s *Server) RecordEvent(e engine.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.events) >= s.maxSize {
		// Shift left by one to make room.
		copy(s.events, s.events[1:])
		s.events = s.events[:len(s.events)-1]
	}
	s.events = append(s.events, e)
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /events", s.handleEvents)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"uptime": time.Since(s.startAt).Round(time.Second).String(),
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	all := make([]engine.Event, len(s.events))
	copy(all, s.events)
	s.mu.RUnlock()

	// Filter by type.
	if t := r.URL.Query().Get("type"); t != "" {
		filtered := make([]engine.Event, 0, len(all))
		for _, e := range all {
			if string(e.Type) == t {
				filtered = append(filtered, e)
			}
		}
		all = filtered
	}

	// Apply limit.
	if l := r.URL.Query().Get("limit"); l != "" {
		if limit, err := strconv.Atoi(l); err == nil && limit > 0 && limit < len(all) {
			all = all[len(all)-limit:]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"events": all,
		"count":  len(all),
	})
}
