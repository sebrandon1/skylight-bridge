package engine

import "sync"

// Bus provides simple pub/sub fan-out for events.
type Bus struct {
	mu          sync.RWMutex
	subscribers []func(Event)
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a callback that will be called for every published event.
func (b *Bus) Subscribe(fn func(Event)) {
	b.mu.Lock()
	b.subscribers = append(b.subscribers, fn)
	b.mu.Unlock()
}

// Publish sends an event to all subscribers synchronously.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	subs := make([]func(Event), len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, fn := range subs {
		fn(event)
	}
}
