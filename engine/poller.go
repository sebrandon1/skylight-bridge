package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	lib "github.com/sebrandon1/go-skylight/lib"

	"github.com/sebrandon1/skylight-bridge/state"
)

// PollerStats is a snapshot of poller runtime counters.
type PollerStats struct {
	TotalPolls   int64            `json:"total_polls"`
	LastPollAt   time.Time        `json:"last_poll_at"`
	EventsByType map[string]int64 `json:"events_by_type"`
}

// Poller periodically fetches chores and rewards from the Skylight API,
// detects state changes via a Detector, and publishes events to a Bus.
type Poller struct {
	client   *lib.Client
	frameID  string
	interval time.Duration
	detector *Detector
	store    *state.Store
	bus      *Bus
	logger   *slog.Logger

	stop chan struct{}
	done chan struct{}

	statsMu      sync.RWMutex
	totalPolls   int64
	lastPollAt   time.Time
	eventsByType map[EventType]int64
}

// NewPoller creates a Poller. It restores detector state from the store.
func NewPoller(client *lib.Client, frameID string, interval time.Duration, store *state.Store, bus *Bus, logger *slog.Logger) *Poller {
	detector := NewDetector()

	// Restore previous detector state.
	s := store.GetState()
	detector.ImportState(DetectorState{
		ChoreStatuses:     s.Chores,
		RewardRedeemed:    s.Rewards,
		AllCompletedFired: s.AllCompletedFired,
	})

	return &Poller{
		client:       client,
		frameID:      frameID,
		interval:     interval,
		detector:     detector,
		store:        store,
		bus:          bus,
		logger:       logger,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
		eventsByType: make(map[EventType]int64),
	}
}

// Start begins polling in a new goroutine.
func (p *Poller) Start(ctx context.Context) {
	go p.loop(ctx)
}

// Stop signals the poll loop to exit and waits for it to finish.
func (p *Poller) Stop() {
	close(p.stop)
	<-p.done
}

func (p *Poller) loop(ctx context.Context) {
	defer close(p.done)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	today := time.Now().Format("2006-01-02")
	p.detector.CleanupOldEntries(today)

	// Fetch resources in parallel.
	var (
		chores     []lib.Chore
		rewards    []lib.Reward
		categories []lib.Category
		choreErr   error
		rewardErr  error
		catErr     error
	)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		chores, choreErr = p.client.ListChores(p.frameID, lib.ChoreListOptions{
			After:       today,
			Before:      today,
			IncludeLate: true,
		})
	}()
	go func() {
		defer wg.Done()
		rewards, rewardErr = p.client.ListRewards(p.frameID)
	}()
	go func() {
		defer wg.Done()
		categories, catErr = p.client.ListCategories(p.frameID)
	}()
	wg.Wait()

	// Check for context cancellation.
	select {
	case <-ctx.Done():
		return
	default:
	}

	if choreErr != nil {
		p.logger.Warn("poll: ListChores failed", slog.String("error", choreErr.Error()))
	}
	if rewardErr != nil {
		p.logger.Warn("poll: ListRewards failed", slog.String("error", rewardErr.Error()))
	}
	if catErr != nil {
		p.logger.Warn("poll: ListCategories failed", slog.String("error", catErr.Error()))
	}

	// Resolve child names.
	if catErr == nil {
		names := make(map[string]string, len(categories))
		for _, c := range categories {
			names[c.ID] = c.Name
		}
		p.detector.SetChildNames(names)
	}

	// Detect events.
	var allEvents []Event
	if choreErr == nil {
		allEvents = append(allEvents, p.detector.DetectChores(chores, today)...)
	}
	if rewardErr == nil {
		allEvents = append(allEvents, p.detector.DetectRewards(rewards)...)
	}

	// Publish events.
	for _, e := range allEvents {
		p.logger.Info("event detected",
			slog.String("type", string(e.Type)),
			slog.Any("data", e.Data),
		)
		p.bus.Publish(e)
	}

	// Persist state.
	if len(allEvents) > 0 {
		exported := p.detector.ExportState()
		p.store.UpdateState(func(s *state.State) {
			s.Chores = exported.ChoreStatuses
			s.Rewards = exported.RewardRedeemed
			s.AllCompletedFired = exported.AllCompletedFired
			s.LastPollAt = time.Now()
		})
	}

	p.logger.Debug("poll complete",
		slog.Int("chores", len(chores)),
		slog.Int("rewards", len(rewards)),
		slog.Int("events", len(allEvents)),
	)

	p.statsMu.Lock()
	p.totalPolls++
	p.lastPollAt = time.Now()
	for _, e := range allEvents {
		p.eventsByType[e.Type]++
	}
	p.statsMu.Unlock()
}

// Stats returns a snapshot of poller runtime counters. Safe for concurrent use.
func (p *Poller) Stats() PollerStats {
	p.statsMu.RLock()
	defer p.statsMu.RUnlock()

	byType := make(map[string]int64, len(p.eventsByType))
	for k, v := range p.eventsByType {
		byType[string(k)] = v
	}
	return PollerStats{
		TotalPolls:   p.totalPolls,
		LastPollAt:   p.lastPollAt,
		EventsByType: byType,
	}
}
