package engine

import "time"

// EventType identifies the kind of state change detected.
type EventType string

const (
	EventChoreCompleted    EventType = "chore.completed"
	EventChoreUncompleted  EventType = "chore.uncompleted"
	EventChoreAllCompleted EventType = "chore.all_completed"
	EventRewardRedeemed    EventType = "reward.redeemed"
)

// Event represents a detected state change from the Skylight API.
type Event struct {
	Type      EventType      `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}
