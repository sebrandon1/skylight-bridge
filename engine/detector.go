package engine

import (
	"time"

	lib "github.com/sebrandon1/go-skylight/lib"
)

const statusPending = "pending"

// Detector compares snapshots of Skylight resources and emits events for
// detected state changes. It is not goroutine-safe; the caller (Poller)
// serializes access.
type Detector struct {
	// Previous state keyed by resource ID.
	choreStatuses  map[string]string // chore ID → status
	rewardRedeemed map[string]bool   // reward ID → redeemed

	// Track which assignee+date combos have already fired all_completed.
	allCompletedFired map[string]bool // "date:assigneeID" → true

	// childNames maps category/assignee ID → human-readable name.
	childNames map[string]string
}

// NewDetector creates a Detector with empty previous state.
func NewDetector() *Detector {
	return &Detector{
		choreStatuses:     make(map[string]string),
		rewardRedeemed:    make(map[string]bool),
		allCompletedFired: make(map[string]bool),
		childNames:        make(map[string]string),
	}
}

// DetectorState holds the serialisable state for persistence.
type DetectorState struct {
	ChoreStatuses     map[string]string `json:"chores"`
	RewardRedeemed    map[string]bool   `json:"rewards"`
	AllCompletedFired map[string]bool   `json:"all_completed_fired"`
}

// ExportState returns a copy of the detector's state for persistence.
func (d *Detector) ExportState() DetectorState {
	cs := make(map[string]string, len(d.choreStatuses))
	for k, v := range d.choreStatuses {
		cs[k] = v
	}
	rr := make(map[string]bool, len(d.rewardRedeemed))
	for k, v := range d.rewardRedeemed {
		rr[k] = v
	}
	acf := make(map[string]bool, len(d.allCompletedFired))
	for k, v := range d.allCompletedFired {
		acf[k] = v
	}
	return DetectorState{
		ChoreStatuses:     cs,
		RewardRedeemed:    rr,
		AllCompletedFired: acf,
	}
}

// ImportState restores detector state from a previously persisted snapshot.
func (d *Detector) ImportState(s DetectorState) {
	if s.ChoreStatuses != nil {
		d.choreStatuses = s.ChoreStatuses
	}
	if s.RewardRedeemed != nil {
		d.rewardRedeemed = s.RewardRedeemed
	}
	if s.AllCompletedFired != nil {
		d.allCompletedFired = s.AllCompletedFired
	}
}

// SetChildNames updates the category ID → name mapping used to enrich events.
func (d *Detector) SetChildNames(names map[string]string) {
	d.childNames = names
}

// DetectChores compares new chore state against previous and returns events.
// It also checks for the all_completed condition per assignee per date.
func (d *Detector) DetectChores(chores []lib.Chore, today string) []Event {
	var events []Event
	now := time.Now()

	// Track per-assignee completion for all_completed detection.
	type assigneeStats struct {
		total     int
		completed int
		points    int
	}
	assignees := make(map[string]*assigneeStats)

	for _, c := range chores {
		prevStatus, known := d.choreStatuses[c.ID]

		// Detect individual chore completion.
		if c.Status != statusPending && (!known || prevStatus == statusPending) {
			events = append(events, Event{
				Type:      EventChoreCompleted,
				Timestamp: now,
				Data: map[string]any{
					"chore_id":      c.ID,
					"chore_title":   c.Title,
					"assignee_id":   c.AssigneeID,
					"assignee_name": d.childNames[c.AssigneeID],
					"points":        c.Points,
					"due_date":      c.DueDate,
				},
			})
		}

		// Update stored status.
		d.choreStatuses[c.ID] = c.Status

		// Accumulate stats for all_completed check.
		if c.DueDate == today && c.AssigneeID != "" {
			stats, ok := assignees[c.AssigneeID]
			if !ok {
				stats = &assigneeStats{}
				assignees[c.AssigneeID] = stats
			}
			stats.total++
			if c.Status != statusPending {
				stats.completed++
			}
			stats.points += c.Points
		}
	}

	// Check all_completed per assignee.
	for assigneeID, stats := range assignees {
		if stats.total == 0 || stats.completed < stats.total {
			continue
		}
		key := today + ":" + assigneeID
		if d.allCompletedFired[key] {
			continue
		}
		d.allCompletedFired[key] = true
		events = append(events, Event{
			Type:      EventChoreAllCompleted,
			Timestamp: now,
			Data: map[string]any{
				"assignee_id":   assigneeID,
				"assignee_name": d.childNames[assigneeID],
				"date":          today,
				"chore_count":   stats.total,
				"total_points":  stats.points,
			},
		})
	}

	return events
}

// DetectRewards compares new reward state against previous and returns events.
func (d *Detector) DetectRewards(rewards []lib.Reward) []Event {
	var events []Event
	now := time.Now()

	for _, r := range rewards {
		prevRedeemed, known := d.rewardRedeemed[r.ID]

		if r.Redeemed && (!known || !prevRedeemed) {
			events = append(events, Event{
				Type:      EventRewardRedeemed,
				Timestamp: now,
				Data: map[string]any{
					"reward_id":    r.ID,
					"reward_title": r.Title,
					"category_id":  r.CategoryID,
					"child_name":   d.childNames[r.CategoryID],
					"points":       r.Points,
					"emoji_icon":   r.EmojiIcon,
				},
			})
		}

		d.rewardRedeemed[r.ID] = r.Redeemed
	}

	return events
}

// CleanupOldEntries removes allCompletedFired entries from before today.
func (d *Detector) CleanupOldEntries(today string) {
	for key := range d.allCompletedFired {
		// Keys are "date:assigneeID"; compare the date prefix.
		if len(key) >= len(today) && key[:len(today)] != today {
			delete(d.allCompletedFired, key)
		}
	}
}
