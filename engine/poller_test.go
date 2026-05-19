package engine

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	lib "github.com/sebrandon1/go-skylight/lib"

	"github.com/sebrandon1/skylight-bridge/state"
)

// fakeClient is a controllable skylightClient for tests.
type fakeClient struct {
	mu         sync.Mutex
	chores     []lib.Chore
	rewards    []lib.Reward
	categories []lib.Category
	choreErr   error
	rewardErr  error
	catErr     error
}

func (f *fakeClient) ListChores(_ string, _ lib.ChoreListOptions) ([]lib.Chore, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.chores, f.choreErr
}

func (f *fakeClient) ListRewards(_ string) ([]lib.Reward, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rewards, f.rewardErr
}

func (f *fakeClient) ListCategories(_ string) ([]lib.Category, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.categories, f.catErr
}

func newTestPoller(client skylightClient, store *state.Store, bus *Bus) *Poller {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewPoller(client, "frame-1", time.Hour, store, bus, logger)
}

func newTestStore(t *testing.T) *state.Store {
	t.Helper()
	store := state.NewStore(filepath.Join(t.TempDir(), "state.json"))
	if err := store.Load(); err != nil {
		t.Fatalf("store.Load: %v", err)
	}
	return store
}

// TestPollerFirstPollOnStart verifies poll runs immediately on Start.
func TestPollerFirstPollOnStart(t *testing.T) {
	client := &fakeClient{}
	store := newTestStore(t)
	bus := NewBus()

	p := newTestPoller(client, store, bus)
	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx)

	// Give the first poll time to complete.
	time.Sleep(20 * time.Millisecond)
	cancel()
	p.Stop()

	stats := p.Stats()
	if stats.TotalPolls < 1 {
		t.Errorf("TotalPolls = %d, want >= 1", stats.TotalPolls)
	}
	if stats.LastPollAt.IsZero() {
		t.Error("LastPollAt should be set after first poll")
	}
}

// TestPollerStopViaContext verifies the loop exits when ctx is canceled.
func TestPollerStopViaContext(t *testing.T) {
	client := &fakeClient{}
	p := newTestPoller(client, newTestStore(t), NewBus())

	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx)
	time.Sleep(10 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() {
		p.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop() did not return after context cancellation")
	}
}

// TestPollerStopViaStop verifies Stop() blocks until the loop exits.
func TestPollerStopViaStop(t *testing.T) {
	client := &fakeClient{}
	p := newTestPoller(client, newTestStore(t), NewBus())

	ctx := context.Background()
	p.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		p.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop() did not return")
	}
}

// TestPollerAPIErrorsCountedInStats verifies failed API calls increment PollErrors.
func TestPollerAPIErrorsCountedInStats(t *testing.T) {
	client := &fakeClient{
		choreErr:  errors.New("chore API down"),
		rewardErr: errors.New("reward API down"),
	}
	p := newTestPoller(client, newTestStore(t), NewBus())
	p.poll(context.Background())

	stats := p.Stats()
	if stats.PollErrors != 2 {
		t.Errorf("PollErrors = %d, want 2", stats.PollErrors)
	}
}

// TestPollerPublishesEvents verifies detected events reach bus subscribers.
func TestPollerPublishesEvents(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	client := &fakeClient{
		categories: []lib.Category{{ID: "cat1", Name: "Alice"}},
		chores: []lib.Chore{
			{ID: "c1", Title: "Clean room", Status: "completed", AssigneeID: "cat1", DueDate: today, Points: 5},
		},
	}

	bus := NewBus()
	received := make(chan Event, 10)
	bus.Subscribe(func(e Event) { received <- e })

	p := newTestPoller(client, newTestStore(t), bus)
	p.poll(context.Background())

	// Allow async bus dispatch.
	time.Sleep(20 * time.Millisecond)

	if len(received) == 0 {
		t.Fatal("expected at least one event to be published")
	}

	// Drain all events; order is non-deterministic since bus dispatches each in its own goroutine.
	var choreCompleted *Event
	for len(received) > 0 {
		e := <-received
		if e.Type == EventChoreCompleted {
			ev := e
			choreCompleted = &ev
		}
	}
	if choreCompleted == nil {
		t.Fatal("expected chore.completed event among published events")
	}
	if choreCompleted.Data["assignee_name"] != "Alice" {
		t.Errorf("assignee_name = %v, want Alice", choreCompleted.Data["assignee_name"])
	}
}

// TestPollerPersistsStateOnEvents verifies the state store is updated when events fire.
func TestPollerPersistsStateOnEvents(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	client := &fakeClient{
		categories: []lib.Category{{ID: "cat1", Name: "Alice"}},
		chores: []lib.Chore{
			{ID: "c1", Title: "Clean room", Status: "completed", AssigneeID: "cat1", DueDate: today, Points: 5},
		},
	}

	store := newTestStore(t)
	p := newTestPoller(client, store, NewBus())
	p.poll(context.Background())

	s := store.GetState()
	if len(s.Chores) == 0 {
		t.Error("expected chore state to be persisted after event")
	}
	if s.LastPollAt.IsZero() {
		t.Error("expected LastPollAt to be set after event")
	}
}

// TestPollerSkipsChoreDetectionOnError verifies no chore events emitted when ListChores fails.
func TestPollerSkipsChoreDetectionOnError(t *testing.T) {
	client := &fakeClient{
		choreErr: errors.New("api error"),
		rewards:  []lib.Reward{},
	}

	bus := NewBus()
	received := make(chan Event, 10)
	bus.Subscribe(func(e Event) { received <- e })

	p := newTestPoller(client, newTestStore(t), bus)
	p.poll(context.Background())
	time.Sleep(20 * time.Millisecond)

	if len(received) > 0 {
		t.Errorf("expected no events when chore API fails, got %d", len(received))
	}
}

// TestPollerRestoresStateFromStore verifies detector state is imported from an existing store.
func TestPollerRestoresStateFromStore(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	store := newTestStore(t)

	// Pre-seed store with a completed chore AND the all_completed key so no events fire.
	store.UpdateState(func(s *state.State) {
		s.Chores = map[string]string{"c1": "completed"}
		s.AllCompletedFired = map[string]bool{today + ":cat1": true}
	})

	client := &fakeClient{
		categories: []lib.Category{{ID: "cat1", Name: "Alice"}},
		chores: []lib.Chore{
			// Same chore, still completed — should NOT fire an event again.
			{ID: "c1", Title: "Clean room", Status: "completed", AssigneeID: "cat1", DueDate: today, Points: 5},
		},
	}

	bus := NewBus()
	received := make(chan Event, 10)
	bus.Subscribe(func(e Event) { received <- e })

	p := newTestPoller(client, store, bus)
	p.poll(context.Background())
	time.Sleep(20 * time.Millisecond)

	if len(received) > 0 {
		t.Errorf("expected no events for already-completed chore, got %d", len(received))
	}
}

// TestPollerContextCancelledDuringFetch verifies poll aborts cleanly if ctx is canceled mid-fetch.
func TestPollerContextCancelledDuringFetch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before poll runs

	client := &fakeClient{}
	p := newTestPoller(client, newTestStore(t), NewBus())
	p.poll(ctx) // should return without panicking

	// Stats are updated even on a canceled poll.
	stats := p.Stats()
	if stats.TotalPolls != 1 {
		t.Errorf("TotalPolls = %d, want 1", stats.TotalPolls)
	}
}

// TestPollerStatsEventsByType verifies per-event-type counters are tracked.
func TestPollerStatsEventsByType(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	client := &fakeClient{
		categories: []lib.Category{{ID: "cat1", Name: "Alice"}},
		chores: []lib.Chore{
			{ID: "c1", Title: "Task A", Status: "completed", AssigneeID: "cat1", DueDate: today, Points: 5},
		},
	}

	p := newTestPoller(client, newTestStore(t), NewBus())
	p.poll(context.Background())

	stats := p.Stats()
	// chore.completed + chore.all_completed (single chore = all completed)
	if stats.EventsByType[string(EventChoreCompleted)] < 1 {
		t.Errorf("EventsByType[chore.completed] = %d, want >= 1", stats.EventsByType[string(EventChoreCompleted)])
	}
}
