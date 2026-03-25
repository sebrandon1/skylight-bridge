package engine

import (
	"testing"

	lib "github.com/sebrandon1/go-skylight/lib"
)

func TestDetectChoreCompleted(t *testing.T) {
	d := NewDetector()
	d.SetChildNames(map[string]string{"cat1": "Alice"})

	// First poll: chore is pending.
	chores := []lib.Chore{
		{ID: "c1", Title: "Clean room", Status: "pending", AssigneeID: "cat1", DueDate: "2026-03-25", Points: 5},
	}
	events := d.DetectChores(chores, "2026-03-25")
	if len(events) != 0 {
		t.Fatalf("expected 0 events on first poll with pending chore, got %d", len(events))
	}

	// Second poll: chore completed.
	chores[0].Status = "completed"
	events = d.DetectChores(chores, "2026-03-25")
	if len(events) != 2 {
		t.Fatalf("expected 2 events (completed + all_completed), got %d", len(events))
	}
	if events[0].Type != EventChoreCompleted {
		t.Errorf("event[0] type = %q, want chore.completed", events[0].Type)
	}
	if events[0].Data["chore_title"] != "Clean room" {
		t.Errorf("chore_title = %v, want Clean room", events[0].Data["chore_title"])
	}
	if events[0].Data["assignee_name"] != "Alice" {
		t.Errorf("assignee_name = %v, want Alice", events[0].Data["assignee_name"])
	}
}

func TestDetectChoreCompletedNoDuplicate(t *testing.T) {
	d := NewDetector()

	chores := []lib.Chore{
		{ID: "c1", Title: "Clean room", Status: "completed", DueDate: "2026-03-25"},
	}
	// First detection.
	d.DetectChores(chores, "2026-03-25")
	// Second poll with same state should not re-emit.
	events := d.DetectChores(chores, "2026-03-25")
	if len(events) != 0 {
		t.Fatalf("expected 0 events on duplicate poll, got %d", len(events))
	}
}

func TestDetectChoreUncompleted(t *testing.T) {
	d := NewDetector()
	d.SetChildNames(map[string]string{"cat1": "Alice"})

	// First poll: chore completed.
	chores := []lib.Chore{
		{ID: "c1", Title: "Clean room", Status: "completed", AssigneeID: "cat1", DueDate: "2026-03-25"},
	}
	d.DetectChores(chores, "2026-03-25")

	// Second poll: chore unchecked back to pending.
	chores[0].Status = "pending"
	events := d.DetectChores(chores, "2026-03-25")

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventChoreUncompleted {
		t.Errorf("type = %q, want chore.uncompleted", events[0].Type)
	}
	if events[0].Data["chore_title"] != "Clean room" {
		t.Errorf("chore_title = %v, want Clean room", events[0].Data["chore_title"])
	}
	if events[0].Data["assignee_name"] != "Alice" {
		t.Errorf("assignee_name = %v, want Alice", events[0].Data["assignee_name"])
	}
}

func TestDetectChoreAllCompleted(t *testing.T) {
	d := NewDetector()
	d.SetChildNames(map[string]string{"cat1": "Bob"})

	// Two chores, both pending.
	chores := []lib.Chore{
		{ID: "c1", Title: "Chore 1", Status: "pending", AssigneeID: "cat1", DueDate: "2026-03-25", Points: 5},
		{ID: "c2", Title: "Chore 2", Status: "pending", AssigneeID: "cat1", DueDate: "2026-03-25", Points: 10},
	}
	events := d.DetectChores(chores, "2026-03-25")
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}

	// Complete first chore only.
	chores[0].Status = "completed"
	events = d.DetectChores(chores, "2026-03-25")
	// Should get chore.completed but NOT all_completed.
	if len(events) != 1 {
		t.Fatalf("expected 1 event (only chore.completed), got %d", len(events))
	}
	if events[0].Type != EventChoreCompleted {
		t.Errorf("type = %q, want chore.completed", events[0].Type)
	}

	// Complete second chore.
	chores[1].Status = "completed"
	events = d.DetectChores(chores, "2026-03-25")
	if len(events) != 2 {
		t.Fatalf("expected 2 events (chore.completed + all_completed), got %d", len(events))
	}

	var found bool
	var allCompleted Event
	for _, e := range events {
		if e.Type == EventChoreAllCompleted {
			found = true
			allCompleted = e
		}
	}
	if !found {
		t.Fatal("expected chore.all_completed event")
	}
	if allCompleted.Data["assignee_name"] != "Bob" {
		t.Errorf("assignee_name = %v, want Bob", allCompleted.Data["assignee_name"])
	}
	if allCompleted.Data["chore_count"] != 2 {
		t.Errorf("chore_count = %v, want 2", allCompleted.Data["chore_count"])
	}
	if allCompleted.Data["total_points"] != 15 {
		t.Errorf("total_points = %v, want 15", allCompleted.Data["total_points"])
	}

	// Third poll: should not re-fire all_completed.
	events = d.DetectChores(chores, "2026-03-25")
	if len(events) != 0 {
		t.Fatalf("expected 0 events on re-poll, got %d", len(events))
	}
}

func TestDetectChoreAllCompletedDifferentDays(t *testing.T) {
	d := NewDetector()

	// Chore from different day should not affect today's all_completed.
	chores := []lib.Chore{
		{ID: "c1", Title: "Today", Status: "completed", AssigneeID: "cat1", DueDate: "2026-03-25"},
		{ID: "c2", Title: "Yesterday", Status: "pending", AssigneeID: "cat1", DueDate: "2026-03-24"},
	}
	events := d.DetectChores(chores, "2026-03-25")
	// Should fire chore.completed for c1 AND all_completed (only c1 is for today).
	hasAllCompleted := false
	for _, e := range events {
		if e.Type == EventChoreAllCompleted {
			hasAllCompleted = true
		}
	}
	if !hasAllCompleted {
		t.Error("expected chore.all_completed since only today's chore (c1) is completed")
	}
}

func TestDetectRewardRedeemed(t *testing.T) {
	d := NewDetector()
	d.SetChildNames(map[string]string{"cat1": "Alice"})

	// First poll: reward not redeemed.
	rewards := []lib.Reward{
		{ID: "r1", Title: "Invest $20 in VOO", Points: 100, CategoryID: "cat1", Redeemed: false},
	}
	events := d.DetectRewards(rewards)
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}

	// Second poll: reward redeemed.
	rewards[0].Redeemed = true
	events = d.DetectRewards(rewards)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventRewardRedeemed {
		t.Errorf("type = %q, want reward.redeemed", events[0].Type)
	}
	if events[0].Data["reward_title"] != "Invest $20 in VOO" {
		t.Errorf("reward_title = %v, want Invest $20 in VOO", events[0].Data["reward_title"])
	}
	if events[0].Data["child_name"] != "Alice" {
		t.Errorf("child_name = %v, want Alice", events[0].Data["child_name"])
	}
}

func TestDetectRewardRedeemedNoDuplicate(t *testing.T) {
	d := NewDetector()

	rewards := []lib.Reward{
		{ID: "r1", Title: "Reward", Points: 50, Redeemed: true},
	}
	d.DetectRewards(rewards)
	events := d.DetectRewards(rewards)
	if len(events) != 0 {
		t.Fatalf("expected 0 events on re-poll, got %d", len(events))
	}
}

func TestDetectRewardNewAndRedeemed(t *testing.T) {
	d := NewDetector()

	// A brand-new reward that appears already redeemed should fire.
	rewards := []lib.Reward{
		{ID: "r1", Title: "Reward", Points: 50, Redeemed: true},
	}
	events := d.DetectRewards(rewards)
	if len(events) != 1 {
		t.Fatalf("expected 1 event for new redeemed reward, got %d", len(events))
	}
}

func TestCleanupOldEntries(t *testing.T) {
	d := NewDetector()
	d.allCompletedFired["2026-03-24:cat1"] = true
	d.allCompletedFired["2026-03-25:cat1"] = true

	d.CleanupOldEntries("2026-03-25")

	if _, ok := d.allCompletedFired["2026-03-24:cat1"]; ok {
		t.Error("old entry should have been cleaned up")
	}
	if _, ok := d.allCompletedFired["2026-03-25:cat1"]; !ok {
		t.Error("today's entry should still exist")
	}
}

func TestExportImportState(t *testing.T) {
	d := NewDetector()
	d.choreStatuses["c1"] = "completed"
	d.rewardRedeemed["r1"] = true
	d.allCompletedFired["2026-03-25:cat1"] = true

	exported := d.ExportState()

	d2 := NewDetector()
	d2.ImportState(exported)

	if d2.choreStatuses["c1"] != "completed" {
		t.Error("choreStatuses not restored")
	}
	if !d2.rewardRedeemed["r1"] {
		t.Error("rewardRedeemed not restored")
	}
	if !d2.allCompletedFired["2026-03-25:cat1"] {
		t.Error("allCompletedFired not restored")
	}
}
