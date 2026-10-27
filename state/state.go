package state

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/sebrandon1/go-toolbox/jsonstore"
)

// State holds the persisted snapshot of detected resource states.
type State struct {
	Chores            map[string]string `json:"chores"`
	Rewards           map[string]bool   `json:"rewards"`
	AllCompletedFired map[string]bool   `json:"all_completed_fired"`
	LastPollAt        time.Time         `json:"last_poll_at"`
	SyncedPhotoFiles  map[string]bool   `json:"synced_photo_files,omitempty"`
}

// Store manages loading and saving state to a JSON file with atomic writes.
type Store struct {
	mu    sync.Mutex
	store *jsonstore.Store[State]
	state State
}

func newState() State {
	return State{
		Chores:            make(map[string]string),
		Rewards:           make(map[string]bool),
		AllCompletedFired: make(map[string]bool),
		SyncedPhotoFiles:  make(map[string]bool),
	}
}

// NewStore creates a Store backed by the given file path.
func NewStore(filePath string) *Store {
	return &Store{
		store: jsonstore.New[State](filePath, jsonstore.WithPerm(0o600), jsonstore.WithIndent(true)),
		state: newState(),
	}
}

// Load reads state from the file. Returns nil if the file does not exist.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.store.Load()
	if err != nil {
		if errors.Is(err, jsonstore.ErrNotFound) {
			return nil
		}
		return err
	}

	if st.Chores == nil {
		st.Chores = make(map[string]string)
	}
	if st.Rewards == nil {
		st.Rewards = make(map[string]bool)
	}
	if st.AllCompletedFired == nil {
		st.AllCompletedFired = make(map[string]bool)
	}
	if st.SyncedPhotoFiles == nil {
		st.SyncedPhotoFiles = make(map[string]bool)
	}

	s.state = st
	return nil
}

// Save writes the current state to disk atomically (write to temp, then rename).
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.store.Save(s.state)
}

// GetState returns a copy of the current state.
func (s *Store) GetState() State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return State{
		Chores:            copyMapSS(s.state.Chores),
		Rewards:           copyMapSB(s.state.Rewards),
		AllCompletedFired: copyMapSB(s.state.AllCompletedFired),
		LastPollAt:        s.state.LastPollAt,
		SyncedPhotoFiles:  copyMapSB(s.state.SyncedPhotoFiles),
	}
}

// UpdateState applies a mutation function to the state and saves it to disk.
func (s *Store) UpdateState(fn func(*State)) {
	s.mu.Lock()
	fn(&s.state)
	s.mu.Unlock()
	if err := s.Save(); err != nil {
		slog.Error("failed to persist state", slog.String("error", err.Error()))
	}
}

func copyMapSS(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copyMapSB(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
