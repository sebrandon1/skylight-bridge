package engine

import (
	"sync"
	"testing"
	"time"
)

func TestBusPublish(t *testing.T) {
	bus := NewBus()

	var received []Event
	bus.Subscribe(func(e Event) {
		received = append(received, e)
	})

	event := Event{
		Type:      EventChoreCompleted,
		Timestamp: time.Now(),
		Data:      map[string]any{"chore_title": "Clean room"},
	}
	bus.Publish(event)

	if len(received) != 1 {
		t.Fatalf("received %d events, want 1", len(received))
	}
	if received[0].Type != EventChoreCompleted {
		t.Errorf("type = %q, want %q", received[0].Type, EventChoreCompleted)
	}
}

func TestBusMultipleSubscribers(t *testing.T) {
	bus := NewBus()

	var mu sync.Mutex
	count := 0
	for range 3 {
		bus.Subscribe(func(Event) {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	bus.Publish(Event{Type: EventRewardRedeemed, Timestamp: time.Now()})

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestBusNoSubscribers(t *testing.T) {
	bus := NewBus()
	// Should not panic.
	bus.Publish(Event{Type: EventChoreCompleted, Timestamp: time.Now()})
}
