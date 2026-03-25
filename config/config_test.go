package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValid(t *testing.T) {
	yaml := `
auth:
  email: "test@example.com"
  password: "secret"
frame_id: "frame-1"
polling:
  interval: "45s"
server:
  addr: ":9090"
  event_buffer_size: 50
log:
  level: "debug"
  format: "text"
rules:
  - name: "test-rule"
    event: "chore.completed"
    filters:
      assignee_name: "Alice"
    actions:
      - type: log
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Auth.Email != "test@example.com" {
		t.Errorf("email = %q, want test@example.com", cfg.Auth.Email)
	}
	if cfg.FrameID != "frame-1" {
		t.Errorf("frame_id = %q, want frame-1", cfg.FrameID)
	}
	if cfg.Server.Addr != ":9090" {
		t.Errorf("addr = %q, want :9090", cfg.Server.Addr)
	}
	if cfg.Server.EventBufferSize != 50 {
		t.Errorf("event_buffer_size = %d, want 50", cfg.Server.EventBufferSize)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("log level = %q, want debug", cfg.Log.Level)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("rules count = %d, want 1", len(cfg.Rules))
	}
	if cfg.Rules[0].Filters["assignee_name"] != "Alice" {
		t.Errorf("filter assignee_name = %q, want Alice", cfg.Rules[0].Filters["assignee_name"])
	}
}

func TestLoadTokenAuth(t *testing.T) {
	yaml := `
auth:
  user_id: "uid1"
  token: "tok1"
frame_id: "frame-1"
rules:
  - name: "r"
    event: "reward.redeemed"
    actions:
      - type: log
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Auth.UserID != "uid1" || cfg.Auth.Token != "tok1" {
		t.Errorf("token auth not parsed correctly")
	}
}

func TestLoadMissingAuth(t *testing.T) {
	yaml := `
frame_id: "frame-1"
rules:
  - name: "r"
    event: "chore.completed"
    actions:
      - type: log
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestLoadMissingFrameID(t *testing.T) {
	yaml := `
auth:
  email: "a@b.com"
  password: "p"
rules:
  - name: "r"
    event: "chore.completed"
    actions:
      - type: log
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing frame_id")
	}
}

func TestLoadMissingRuleName(t *testing.T) {
	yaml := `
auth:
  email: "a@b.com"
  password: "p"
frame_id: "f"
rules:
  - event: "chore.completed"
    actions:
      - type: log
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing rule name")
	}
}

func TestLoadMissingRuleActions(t *testing.T) {
	yaml := `
auth:
  email: "a@b.com"
  password: "p"
frame_id: "f"
rules:
  - name: "r"
    event: "chore.completed"
`
	path := writeTempConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing rule actions")
	}
}

func TestDefaults(t *testing.T) {
	yaml := `
auth:
  email: "a@b.com"
  password: "p"
frame_id: "f"
rules:
  - name: "r"
    event: "chore.completed"
    actions:
      - type: log
`
	path := writeTempConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Addr != ":8080" {
		t.Errorf("default addr = %q, want :8080", cfg.Server.Addr)
	}
	if cfg.Server.EventBufferSize != 100 {
		t.Errorf("default buffer size = %d, want 100", cfg.Server.EventBufferSize)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("default log level = %q, want info", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("default log format = %q, want json", cfg.Log.Format)
	}
}

func TestParsedInterval(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"45s", "45s"},
		{"2m", "2m0s"},
		{"", "30s"},
		{"invalid", "30s"},
		{"-5s", "30s"},
	}
	for _, tt := range tests {
		p := PollingConfig{Interval: tt.input}
		got := p.ParsedInterval().String()
		if got != tt.want {
			t.Errorf("ParsedInterval(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
