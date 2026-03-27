package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
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
	mu       sync.Mutex
	filePath string
	state    State
}

// NewStore creates a Store backed by the given file path.
func NewStore(filePath string) *Store {
	return &Store{
		filePath: filePath,
		state: State{
			Chores:            make(map[string]string),
			Rewards:           make(map[string]bool),
			AllCompletedFired: make(map[string]bool),
			SyncedPhotoFiles:  make(map[string]bool),
		},
	}
}

// Load reads state from the file. Returns nil if the file does not exist.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
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
	data, err := json.MarshalIndent(s.state, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
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
	_ = s.Save()
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
