package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sebrandon1/skylight-bridge/action"
	"github.com/sebrandon1/skylight-bridge/config"
	"github.com/sebrandon1/skylight-bridge/engine"
	"github.com/sebrandon1/skylight-bridge/rules"
	"github.com/sebrandon1/skylight-bridge/server"
	"github.com/sebrandon1/skylight-bridge/state"

	lib "github.com/sebrandon1/go-skylight/lib"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	var (
		configPath  string
		showVersion bool
	)
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg.Log)

	client, err := buildClient(cfg, logger)
	if err != nil {
		logger.Error("failed to create skylight client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	store := state.NewStore(cfg.StateFile)
	if err := store.Load(); err != nil {
		logger.Warn("could not load state, starting fresh", slog.String("error", err.Error()))
	}

	bus := engine.NewBus()

	// Set up event server.
	bufSize := cfg.Server.EventBufferSize
	if bufSize == 0 {
		bufSize = 100
	}
	srv := server.New(bufSize)
	bus.Subscribe(srv.RecordEvent)

	// Set up rules engine.
	factories := map[string]action.Factory{
		"log":           action.NewLogAction,
		"webhook":       action.NewWebhookAction,
		"homeassistant": action.NewHomeAssistantAction,
	}
	rulesEngine, err := rules.NewEngine(cfg.Rules, factories, logger)
	if err != nil {
		logger.Error("failed to build rules engine", slog.String("error", err.Error()))
		os.Exit(1)
	}
	bus.Subscribe(func(e engine.Event) {
		rulesEngine.HandleEvent(context.Background(), e)
	})

	// Set up poller.
	interval := cfg.Polling.ParsedInterval()
	poller := engine.NewPoller(client, cfg.FrameID, interval, store, bus, logger)

	// Signal handling.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start poller.
	poller.Start(ctx)

	// Start HTTP server.
	httpServer := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("starting HTTP server", slog.String("addr", cfg.Server.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", slog.String("error", err.Error()))
		}
	}()

	logger.Info("skylight-bridge started",
		slog.String("version", Version),
		slog.String("frame_id", cfg.FrameID),
		slog.Duration("interval", interval),
	)

	// Wait for shutdown signal.
	<-ctx.Done()
	logger.Info("shutting down...")

	poller.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
	}
}

func setupLogger(cfg config.LogConfig) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level}
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}

func buildClient(cfg *config.Config, logger *slog.Logger) (*lib.Client, error) {
	opts := []lib.ClientOption{
		lib.WithLogger(logger),
		lib.WithRateLimit(2, 5),
		lib.WithRetry(3, 500*time.Millisecond, 10*time.Second),
	}

	if cfg.Auth.Token != "" && cfg.Auth.UserID != "" {
		return lib.NewClientWithToken(cfg.Auth.UserID, cfg.Auth.Token, opts...)
	}
	return lib.NewClient(cfg.Auth.Email, cfg.Auth.Password, opts...)
}
