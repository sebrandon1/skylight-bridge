package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for skylight-bridge.
type Config struct {
	Auth      AuthConfig       `yaml:"auth"`
	FrameID   string           `yaml:"frame_id"`
	Polling   PollingConfig    `yaml:"polling"`
	StateFile string           `yaml:"state_file"`
	Server    ServerConfig     `yaml:"server"`
	Log       LogConfig        `yaml:"log"`
	Rules     []RuleConfig     `yaml:"rules"`
	PhotoSync *PhotoSyncConfig `yaml:"photo_sync,omitempty"`
}

// PhotoSyncConfig holds settings for the local folder photo sync.
type PhotoSyncConfig struct {
	WatchFolder string `yaml:"watch_folder"`
	// FrameID overrides the top-level frame_id for photo uploads.
	// Defaults to the top-level frame_id if omitted.
	FrameID      string `yaml:"frame_id"`
	SyncInterval string `yaml:"sync_interval"`
}

// ParsedSyncInterval returns the sync interval as a time.Duration.
// Defaults to 1m if not set or unparseable.
func (p PhotoSyncConfig) ParsedSyncInterval() time.Duration {
	return parseDurationWithDefault(p.SyncInterval, time.Minute)
}

// AuthConfig holds Skylight authentication credentials.
type AuthConfig struct {
	Email             string `yaml:"email"`
	Password          string `yaml:"password"`
	UserID            string `yaml:"user_id"`
	Token             string `yaml:"token"`
	RefreshToken      string `yaml:"refresh_token"`
	DeviceFingerprint string `yaml:"device_fingerprint"`
}

// PollingConfig controls how often the bridge polls the Skylight API.
type PollingConfig struct {
	Interval string `yaml:"interval"`
}

// ParsedInterval returns the polling interval as a time.Duration.
// Defaults to 30s if not set or unparseable.
func (p PollingConfig) ParsedInterval() time.Duration {
	return parseDurationWithDefault(p.Interval, 30*time.Second)
}

func parseDurationWithDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

// ServerConfig controls the built-in HTTP server.
type ServerConfig struct {
	Addr            string `yaml:"addr"`
	EventBufferSize int    `yaml:"event_buffer_size"`
}

// LogConfig controls logging output.
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// RuleConfig defines a single event-matching rule with actions.
type RuleConfig struct {
	Name    string            `yaml:"name"`
	Event   string            `yaml:"event"`
	Filters map[string]string `yaml:"filters"`
	Actions []ActionConfig    `yaml:"actions"`
}

// ActionConfig defines an action to execute when a rule matches.
type ActionConfig struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config"`
}

// Load reads and validates a config file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) validate() error {
	hasEmailAuth := c.Auth.Email != "" || c.Auth.Password != ""
	hasTokenAuth := c.Auth.UserID != "" && c.Auth.Token != ""
	hasRefreshAuth := c.Auth.RefreshToken != "" && c.Auth.DeviceFingerprint != ""
	if hasEmailAuth && !hasTokenAuth && !hasRefreshAuth {
		return fmt.Errorf("email+password auth is no longer supported by Skylight; use refresh_token+device_fingerprint instead")
	}
	if !hasTokenAuth && !hasRefreshAuth {
		return fmt.Errorf("auth requires either refresh_token+device_fingerprint or user_id+token")
	}
	if c.FrameID == "" {
		return fmt.Errorf("frame_id is required")
	}
	if err := c.validatePhotoSync(); err != nil {
		return err
	}
	return c.validateRules()
}

func (c *Config) validateRules() error {
	for i, r := range c.Rules {
		if r.Name == "" {
			return fmt.Errorf("rule %d: name is required", i)
		}
		if r.Event == "" {
			return fmt.Errorf("rule %q: event is required", r.Name)
		}
		if len(r.Actions) == 0 {
			return fmt.Errorf("rule %q: at least one action is required", r.Name)
		}
		for j, a := range r.Actions {
			if a.Type == "" {
				return fmt.Errorf("rule %q action %d: type is required", r.Name, j)
			}
		}
	}
	return nil
}

func (c *Config) validatePhotoSync() error {
	ps := c.PhotoSync
	if ps == nil {
		return nil
	}
	if ps.WatchFolder == "" {
		return fmt.Errorf("photo_sync requires watch_folder")
	}
	return nil
}

func expandHomeDir(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

func (c *Config) applyDefaults() {
	if c.StateFile == "" {
		c.StateFile = "~/.skylight-bridge/state.json"
	}
	c.StateFile = expandHomeDir(c.StateFile)
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Server.EventBufferSize == 0 {
		c.Server.EventBufferSize = 100
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}
	if ps := c.PhotoSync; ps != nil {
		if ps.FrameID == "" {
			ps.FrameID = c.FrameID
		}
		ps.WatchFolder = expandHomeDir(ps.WatchFolder)
	}
}
