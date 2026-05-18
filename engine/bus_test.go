package engine

import (
	"sync"
	"testing"
	"time"
)

func TestBusPublish(t *testing.T) {
	bus := NewBus()

	var (
		mu       sync.Mutex
		received []Event
		wg       sync.WaitGroup
	)
	wg.Add(1)
	bus.Subscribe(func(e Event) {
		defer wg.Done()
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	event := Event{
		Type:      EventChoreCompleted,
		Timestamp: time.Now(),
		Data:      map[string]any{"chore_title": "Clean room"},
	}
	bus.Publish(event)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received %d events, want 1", len(received))
	}
	if received[0].Type != EventChoreCompleted {
		t.Errorf("type = %q, want %q", received[0].Type, EventChoreCompleted)
	}
}

func TestBusMultipleSubscribers(t *testing.T) {
	bus := NewBus()

	var (
		mu    sync.Mutex
		count int
		wg    sync.WaitGroup
	)
	wg.Add(3)
	for range 3 {
		bus.Subscribe(func(Event) {
			defer wg.Done()
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	bus.Publish(Event{Type: EventRewardRedeemed, Timestamp: time.Now()})
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestBusSlowSubscriberDoesNotBlockFast(t *testing.T) {
	bus := NewBus()

	var (
		fastDone time.Time
		mu       sync.Mutex
		wg       sync.WaitGroup
	)
	wg.Add(2)

	bus.Subscribe(func(Event) {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
	})
	bus.Subscribe(func(Event) {
		defer wg.Done()
		mu.Lock()
		fastDone = time.Now()
		mu.Unlock()
	})

	start := time.Now()
	bus.Publish(Event{Type: EventChoreCompleted, Timestamp: time.Now()})
	wg.Wait()

	mu.Lock()
	elapsed := fastDone.Sub(start)
	mu.Unlock()
	if elapsed >= 50*time.Millisecond {
		t.Errorf("fast subscriber took %v, want < 50ms; slow subscriber may have blocked it", elapsed)
	}
}

func TestBusNoSubscribers(t *testing.T) {
	bus := NewBus()
	bus.Publish(Event{Type: EventChoreCompleted, Timestamp: time.Now()})
}
