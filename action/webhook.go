package action

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/template"
	"time"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// WebhookAction sends an HTTP request when an event fires.
type WebhookAction struct {
	url      string
	method   string
	headers  map[string]string
	bodyTmpl *template.Template
	client   *http.Client
}

// NewWebhookAction creates a WebhookAction from config. Supported keys:
//   - url: target URL (required)
//   - method: HTTP method (default: POST)
//   - headers: map of header name → value
//   - body_template: Go text/template for request body (optional; default: JSON event)
func NewWebhookAction(config map[string]any) (Action, error) {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("webhook action requires 'url'")
	}

	method := "POST"
	if m, ok := config["method"].(string); ok && m != "" {
		method = m
	}

	headers := make(map[string]string)
	if h, ok := config["headers"].(map[string]any); ok {
		for k, v := range h {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}

	a := &WebhookAction{
		url:     url,
		method:  method,
		headers: headers,
		client:  &http.Client{Timeout: 10 * time.Second},
	}

	if tmplStr, ok := config["body_template"].(string); ok && tmplStr != "" {
		tmpl, err := template.New("webhook").Parse(tmplStr)
		if err != nil {
			return nil, fmt.Errorf("parsing webhook body template: %w", err)
		}
		a.bodyTmpl = tmpl
	}

	return a, nil
}

// Execute sends the webhook request.
func (a *WebhookAction) Execute(ctx context.Context, event engine.Event) error {
	var body []byte
	var err error

	if a.bodyTmpl != nil {
		var buf bytes.Buffer
		if err := a.bodyTmpl.Execute(&buf, event.Data); err != nil {
			return fmt.Errorf("executing webhook body template: %w", err)
		}
		body = buf.Bytes()
	} else {
		body, err = json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshaling event: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, a.method, a.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range a.headers {
		req.Header.Set(k, v)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
