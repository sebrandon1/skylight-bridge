package googlephotos

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	skylightlib "github.com/sebrandon1/go-skylight/lib"
)

// Syncer periodically fetches recent Google Photos and uploads new ones to a
// Skylight frame. Already-synced items are tracked by media item ID to avoid
// duplicates across restarts.
type Syncer struct {
	gpClient       *Client
	skylightClient *skylightlib.Client
	frameID        string
	syncCount      int
	interval       time.Duration
	logger         *slog.Logger
	mu             sync.Mutex
	syncedIDs      map[string]bool
	// onBatchDone is called after each sync pass with the updated synced IDs,
	// allowing the caller to persist state incrementally rather than only on shutdown.
	onBatchDone func(map[string]bool)
}

// NewSyncer creates a Syncer. syncedIDs is the set of already-uploaded Google
// Photos media item IDs (loaded from persistent state by the caller).
// onBatchDone is called after each sync pass; pass nil to skip incremental persistence.
func NewSyncer(
	gpClient *Client,
	skylightClient *skylightlib.Client,
	frameID string,
	syncCount int,
	interval time.Duration,
	syncedIDs map[string]bool,
	logger *slog.Logger,
	onBatchDone func(map[string]bool),
) *Syncer {
	if syncedIDs == nil {
		syncedIDs = make(map[string]bool)
	}
	return &Syncer{
		gpClient:       gpClient,
		skylightClient: skylightClient,
		frameID:        frameID,
		syncCount:      syncCount,
		interval:       interval,
		logger:         logger,
		syncedIDs:      syncedIDs,
		onBatchDone:    onBatchDone,
	}
}

// Start runs the sync loop until ctx is canceled. It syncs immediately on
// startup, then repeats on the configured interval.
func (s *Syncer) Start(ctx context.Context) {
	go func() {
		s.sync()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.sync()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// SyncedIDs returns a copy of the synced ID set for state persistence.
func (s *Syncer) SyncedIDs() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.syncedIDs))
	for k, v := range s.syncedIDs {
		out[k] = v
	}
	return out
}

func (s *Syncer) sync() {
	items, err := s.gpClient.ListRecentItems(s.syncCount)
	if err != nil {
		s.logger.Error("google photos: failed to list items", slog.String("error", err.Error()))
		return
	}

	uploaded := 0
	for _, item := range items {
		s.mu.Lock()
		seen := s.syncedIDs[item.ID]
		s.mu.Unlock()
		if seen {
			continue
		}

		if !isPhoto(item) {
			s.mu.Lock()
			s.syncedIDs[item.ID] = true
			s.mu.Unlock()
			continue
		}

		data, err := s.gpClient.DownloadImage(item)
		if err != nil {
			s.logger.Error("google photos: failed to download item",
				slog.String("id", item.ID),
				slog.String("filename", item.Filename),
				slog.String("error", err.Error()),
			)
			continue
		}

		ext := extFromFilename(item.Filename)
		_, err = s.skylightClient.UploadPhoto(s.frameID, ext, data, "")
		if err != nil {
			s.logger.Error("google photos: failed to upload to skylight",
				slog.String("id", item.ID),
				slog.String("filename", item.Filename),
				slog.String("error", err.Error()),
			)
			continue
		}

		s.mu.Lock()
		s.syncedIDs[item.ID] = true
		s.mu.Unlock()
		uploaded++
		s.logger.Info("google photos: uploaded photo",
			slog.String("id", item.ID),
			slog.String("filename", item.Filename),
		)
	}

	if uploaded > 0 {
		s.logger.Info("google photos: sync complete", slog.Int("uploaded", uploaded))
	}

	if s.onBatchDone != nil {
		s.onBatchDone(s.SyncedIDs())
	}
}

func isPhoto(item MediaItem) bool {
	return strings.HasPrefix(item.MimeType, "image/")
}

func extFromFilename(filename string) string {
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		return "jpg"
	}
	return strings.ToLower(ext)
}
