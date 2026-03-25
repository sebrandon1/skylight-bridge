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

// SlackAction posts messages to a Slack channel via incoming webhook.
type SlackAction struct {
	webhookURL string
	msgTmpl    *template.Template
	client     *http.Client
}

// NewSlackAction creates a SlackAction from config. Supported keys:
//   - webhook_url: Slack incoming webhook URL (required)
//   - message: Go text/template for the message content (optional)
func NewSlackAction(config map[string]any) (Action, error) {
	url, ok := config["webhook_url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("slack action requires 'webhook_url'")
	}

	a := &SlackAction{
		webhookURL: url,
		client:     &http.Client{Timeout: 10 * time.Second},
	}

	if msg, ok := config["message"].(string); ok && msg != "" {
		tmpl, err := template.New("slack").Parse(msg)
		if err != nil {
			return nil, fmt.Errorf("parsing slack message template: %w", err)
		}
		a.msgTmpl = tmpl
	}

	return a, nil
}

// Execute posts the event as a message to Slack.
func (a *SlackAction) Execute(ctx context.Context, event engine.Event) error {
	var content string

	if a.msgTmpl != nil {
		var buf bytes.Buffer
		if err := a.msgTmpl.Execute(&buf, event.Data); err != nil {
			return fmt.Errorf("executing slack message template: %w", err)
		}
		content = buf.String()
	} else {
		content = formatDefaultMessage(event)
	}

	payload, err := json.Marshal(map[string]string{"text": content})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("slack webhook failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}
