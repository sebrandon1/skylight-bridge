package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
	"github.com/sebrandon1/skylight-bridge/rules"
)

// Server provides HTTP endpoints for event history and health checks.
type Server struct {
	mu      sync.RWMutex
	events  []engine.Event // fixed-size circular buffer
	head    int            // index of the oldest event
	size    int            // number of events currently stored
	maxSize int
	startAt time.Time

	authToken   string
	rulesEngine *rules.Engine
	poller      *engine.Poller
}

// New creates a Server with a ring buffer of the given capacity.
func New(bufferSize int) *Server {
	return &Server{
		events:  make([]engine.Event, bufferSize),
		maxSize: bufferSize,
		startAt: time.Now(),
	}
}

// SetAuthToken configures a bearer token required on all requests. Must be
// called before Handler().
func (s *Server) SetAuthToken(token string) { s.authToken = token }

// SetRulesEngine wires the rules engine for GET /rules introspection.
func (s *Server) SetRulesEngine(e *rules.Engine) { s.rulesEngine = e }

// SetPoller wires the poller for GET /stats and GET /healthz.
func (s *Server) SetPoller(p *engine.Poller) { s.poller = p }

// RecordEvent adds an event to the ring buffer. Safe for concurrent use.
func (s *Server) RecordEvent(e engine.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.size < s.maxSize {
		s.events[(s.head+s.size)%s.maxSize] = e
		s.size++
	} else {
		// Buffer full: overwrite oldest slot and advance head.
		s.events[s.head] = e
		s.head = (s.head + 1) % s.maxSize
	}
}

// snapshot returns buffered events in insertion order. Must be called with mu held (read).
func (s *Server) snapshot() []engine.Event {
	out := make([]engine.Event, s.size)
	for i := range s.size {
		out[i] = s.events[(s.head+i)%s.maxSize]
	}
	return out
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /events", s.handleEvents)
	mux.HandleFunc("GET /rules", s.handleRules)
	mux.HandleFunc("GET /stats", s.handleStats)

	if s.authToken == "" {
		return mux
	}
	token := s.authToken
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{
		"status": "ok",
		"uptime": time.Since(s.startAt).Round(time.Second).String(),
	}
	if s.poller != nil {
		stats := s.poller.Stats()
		if !stats.LastPollAt.IsZero() {
			resp["last_poll_at"] = stats.LastPollAt
		}
		resp["poll_errors"] = stats.PollErrors
	}
	if s.rulesEngine != nil {
		resp["action_failures"] = s.rulesEngine.ActionFailures()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	all := s.snapshot()
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

func (s *Server) handleRules(w http.ResponseWriter, _ *http.Request) {
	if s.rulesEngine == nil {
		http.Error(w, "rules engine not available", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"rules": s.rulesEngine.GetRules(),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	if s.poller == nil {
		http.Error(w, "poller not available", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.poller.Stats())
}
