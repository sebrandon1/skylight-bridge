package action

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/sebrandon1/go-toolbox/httpx"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// HomeAssistantAction calls a Home Assistant service or webhook.
type HomeAssistantAction struct {
	baseURL   string
	token     string
	service   string
	entityID  string
	webhookID string
	client    *httpx.Client
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
	if err := validateHTTPURL(baseURL); err != nil {
		return nil, fmt.Errorf("homeassistant action: %w", err)
	}

	a := &HomeAssistantAction{
		baseURL: baseURL,
		client:  httpx.NewClient(httpx.Options{Timeout: parseTimeout(config)}),
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

	req, err := a.client.NewRequest(ctx, http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("HA service call failed: %w", httpx.AsNetworkError(err))
	}
	if err := httpx.HandleResponse(resp, nil, 0); err != nil {
		return fmt.Errorf("HA service call failed: %w", err)
	}
	return nil
}

func (a *HomeAssistantAction) callWebhook(ctx context.Context, event engine.Event) error {
	url := fmt.Sprintf("%s/api/webhook/%s", a.baseURL, a.webhookID)

	if err := a.client.DoJSON(ctx, http.MethodPost, url, event, nil); err != nil {
		return fmt.Errorf("HA webhook call failed: %w", err)
	}
	return nil
}
