package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// HomeAssistantAction calls a Home Assistant service or webhook.
type HomeAssistantAction struct {
	baseURL   string
	token     string
	service   string
	entityID  string
	webhookID string
	client    *http.Client
}

// NewHomeAssistantAction creates a HomeAssistantAction from config. Supported keys:
//   - url: HA base URL (required, e.g. "http://homeassistant.local:8123")
//   - token: long-lived access token (required for service calls)
//   - service: HA service to call, e.g. "light.turn_on" (mutually exclusive with webhook_id)
//   - entity_id: target entity, e.g. "light.living_room"
//   - webhook_id: HA webhook ID (mutually exclusive with service)
func NewHomeAssistantAction(config map[string]any) (Action, error) {
	baseURL, ok := config["url"].(string)
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("homeassistant action requires 'url'")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	a := &HomeAssistantAction{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}

	if t, ok := config["token"].(string); ok {
		a.token = t
	}
	if s, ok := config["service"].(string); ok {
		a.service = s
	}
	if e, ok := config["entity_id"].(string); ok {
		a.entityID = e
	}
	if w, ok := config["webhook_id"].(string); ok {
		a.webhookID = w
	}

	if a.service == "" && a.webhookID == "" {
		return nil, fmt.Errorf("homeassistant action requires 'service' or 'webhook_id'")
	}

	return a, nil
}

// Execute calls the Home Assistant API.
func (a *HomeAssistantAction) Execute(ctx context.Context, event engine.Event) error {
	if a.webhookID != "" {
		return a.callWebhook(ctx, event)
	}
	return a.callService(ctx, event)
}

func (a *HomeAssistantAction) callService(ctx context.Context, _ engine.Event) error {
	parts := strings.SplitN(a.service, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid service format %q, expected 'domain.service'", a.service)
	}

	url := fmt.Sprintf("%s/api/services/%s/%s", a.baseURL, parts[0], parts[1])

	payload := map[string]any{}
	if a.entityID != "" {
		payload["entity_id"] = a.entityID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("HA service call failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HA service call returned status %d", resp.StatusCode)
	}
	return nil
}

func (a *HomeAssistantAction) callWebhook(ctx context.Context, event engine.Event) error {
	url := fmt.Sprintf("%s/api/webhook/%s", a.baseURL, a.webhookID)

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("HA webhook call failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HA webhook returned status %d", resp.StatusCode)
	}
	return nil
}
