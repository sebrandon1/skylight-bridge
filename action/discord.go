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

// DiscordAction posts messages to a Discord channel via webhook.
type DiscordAction struct {
	webhookURL string
	msgTmpl    *template.Template
	client     *http.Client
}

// NewDiscordAction creates a DiscordAction from config. Supported keys:
//   - webhook_url: Discord webhook URL (required)
//   - message: Go text/template for the message content (optional)
func NewDiscordAction(config map[string]any) (Action, error) {
	url, ok := config["webhook_url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("discord action requires 'webhook_url'")
	}

	a := &DiscordAction{
		webhookURL: url,
		client:     &http.Client{Timeout: 10 * time.Second},
	}

	if msg, ok := config["message"].(string); ok && msg != "" {
		tmpl, err := template.New("discord").Parse(msg)
		if err != nil {
			return nil, fmt.Errorf("parsing discord message template: %w", err)
		}
		a.msgTmpl = tmpl
	}

	return a, nil
}

// Execute posts the event as a message to Discord.
func (a *DiscordAction) Execute(ctx context.Context, event engine.Event) error {
	var content string

	if a.msgTmpl != nil {
		var buf bytes.Buffer
		if err := a.msgTmpl.Execute(&buf, event.Data); err != nil {
			return fmt.Errorf("executing discord message template: %w", err)
		}
		content = buf.String()
	} else {
		content = formatDefaultMessage(event)
	}

	payload, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord webhook failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func formatDefaultMessage(event engine.Event) string {
	switch event.Type {
	case engine.EventChoreCompleted:
		return fmt.Sprintf("**%s** completed **%s**",
			event.Data["assignee_name"], event.Data["chore_title"])
	case engine.EventChoreUncompleted:
		return fmt.Sprintf("**%s** unchecked **%s**",
			event.Data["assignee_name"], event.Data["chore_title"])
	case engine.EventChoreAllCompleted:
		return fmt.Sprintf("**%s** finished all chores for today! (%v chores, %v points)",
			event.Data["assignee_name"], event.Data["chore_count"], event.Data["total_points"])
	case engine.EventRewardRedeemed:
		return fmt.Sprintf("**%s** redeemed **%s** (%v points)",
			event.Data["child_name"], event.Data["reward_title"], event.Data["points"])
	default:
		return fmt.Sprintf("Event: %s", event.Type)
	}
}
