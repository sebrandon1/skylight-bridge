package photosync

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	skylightlib "github.com/sebrandon1/go-skylight/lib"
)

var supportedExts = map[string]bool{
	"jpg": true, "jpeg": true, "png": true,
	"gif": true, "webp": true, "heic": true, "heif": true,
}

// Syncer periodically scans a local folder for new image files and uploads
// them to a Skylight frame. Already-uploaded filenames are tracked to avoid
// duplicate uploads across restarts.
type Syncer struct {
	skylightClient *skylightlib.Client
	watchFolder    string
	frameID        string
	interval       time.Duration
	logger         *slog.Logger
	mu             sync.Mutex
	syncedFiles    map[string]bool
	// onBatchDone is called after each scan pass with the updated synced file
	// set, allowing the caller to persist state incrementally.
	onBatchDone func(map[string]bool)
}

// NewSyncer creates a Syncer. syncedFiles is the set of already-uploaded
// filenames (loaded from persistent state by the caller).
// onBatchDone is called after each scan pass; pass nil to skip persistence.
func NewSyncer(
	skylightClient *skylightlib.Client,
	watchFolder string,
	frameID string,
	interval time.Duration,
	syncedFiles map[string]bool,
	logger *slog.Logger,
	onBatchDone func(map[string]bool),
) *Syncer {
	if syncedFiles == nil {
		syncedFiles = make(map[string]bool)
	}
	return &Syncer{
		skylightClient: skylightClient,
		watchFolder:    watchFolder,
		frameID:        frameID,
		interval:       interval,
		logger:         logger,
		syncedFiles:    syncedFiles,
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

// SyncedFiles returns a copy of the synced filename set for state persistence.
func (s *Syncer) SyncedFiles() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.syncedFiles))
	for k, v := range s.syncedFiles {
		out[k] = v
	}
	return out
}

func (s *Syncer) sync() {
	entries, err := os.ReadDir(s.watchFolder)
	if err != nil {
		s.logger.Error("photo sync: failed to read watch folder",
			slog.String("folder", s.watchFolder),
			slog.String("error", err.Error()),
		)
		return
	}

	uploaded := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := extFromFilename(name)
		if !supportedExts[ext] {
			continue
		}

		s.mu.Lock()
		seen := s.syncedFiles[name]
		s.mu.Unlock()
		if seen {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.watchFolder, name))
		if err != nil {
			s.logger.Error("photo sync: failed to read file",
				slog.String("file", name),
				slog.String("error", err.Error()),
			)
			continue
		}

		_, err = s.skylightClient.UploadPhoto(s.frameID, ext, data, "")
		if err != nil {
			s.logger.Error("photo sync: failed to upload to skylight",
				slog.String("file", name),
				slog.String("error", err.Error()),
			)
			continue
		}

		s.mu.Lock()
		s.syncedFiles[name] = true
		s.mu.Unlock()
		uploaded++
		s.logger.Info("photo sync: uploaded photo", slog.String("file", name))
	}

	if uploaded > 0 {
		s.logger.Info("photo sync: scan complete", slog.Int("uploaded", uploaded))
	}

	if s.onBatchDone != nil {
		s.onBatchDone(s.SyncedFiles())
	}
}

func extFromFilename(filename string) string {
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext)
}
