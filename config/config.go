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
	Auth      AuthConfig    `yaml:"auth"`
	FrameID   string        `yaml:"frame_id"`
	Polling   PollingConfig `yaml:"polling"`
	StateFile string        `yaml:"state_file"`
	Server    ServerConfig  `yaml:"server"`
	Log       LogConfig     `yaml:"log"`
	Rules     []RuleConfig  `yaml:"rules"`
}

// AuthConfig holds Skylight authentication credentials.
type AuthConfig struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
	UserID   string `yaml:"user_id"`
	Token    string `yaml:"token"`
}

// PollingConfig controls how often the bridge polls the Skylight API.
type PollingConfig struct {
	Interval string `yaml:"interval"`
}

// ParsedInterval returns the polling interval as a time.Duration.
// Defaults to 30s if not set or unparseable.
func (p PollingConfig) ParsedInterval() time.Duration {
	if p.Interval == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(p.Interval)
	if err != nil || d <= 0 {
		return 30 * time.Second
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
	hasEmailAuth := c.Auth.Email != "" && c.Auth.Password != ""
	hasTokenAuth := c.Auth.UserID != "" && c.Auth.Token != ""
	if !hasEmailAuth && !hasTokenAuth {
		return fmt.Errorf("auth requires either email+password or user_id+token")
	}
	if c.FrameID == "" {
		return fmt.Errorf("frame_id is required")
	}
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

func (c *Config) applyDefaults() {
	if c.StateFile == "" {
		c.StateFile = "~/.skylight-bridge/state.json"
	}
	if strings.HasPrefix(c.StateFile, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			c.StateFile = filepath.Join(home, c.StateFile[2:])
		}
	}
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
}
