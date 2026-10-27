package action

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"text/template"

	"github.com/sebrandon1/go-toolbox/httpx"

	"github.com/sebrandon1/skylight-bridge/engine"
)

// chatAction is the shared implementation for chat-webhook actions (Discord, Slack, etc.).
// The only difference between services is the JSON payload key and service name used in errors.
type chatAction struct {
	webhookURL string
	msgTmpl    *template.Template
	client     *httpx.Client
	payloadKey string // e.g. "content" for Discord, "text" for Slack
	name       string // used in error messages
}

func newChatAction(config map[string]any, name, payloadKey string) (Action, error) {
	url, ok := config["webhook_url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("%s action requires 'webhook_url'", name)
	}

	a := &chatAction{
		webhookURL: url,
		client:     httpx.NewClient(httpx.Options{Timeout: parseTimeout(config)}),
		payloadKey: payloadKey,
		name:       name,
	}

	if msg, ok := config["message"].(string); ok && msg != "" {
		tmpl, err := template.New(name).Parse(msg)
		if err != nil {
			return nil, fmt.Errorf("parsing %s message template: %w", name, err)
		}
		a.msgTmpl = tmpl
	}

	return a, nil
}

func (a *chatAction) Execute(ctx context.Context, event engine.Event) error {
	var content string

	if a.msgTmpl != nil {
		var buf bytes.Buffer
		if err := a.msgTmpl.Execute(&buf, event.Data); err != nil {
			return fmt.Errorf("executing %s message template: %w", a.name, err)
		}
		content = buf.String()
	} else {
		content = formatDefaultMessage(event)
	}

	payload := map[string]string{a.payloadKey: content}
	if err := a.client.DoJSON(ctx, http.MethodPost, a.webhookURL, payload, nil); err != nil {
		return fmt.Errorf("%s webhook failed: %w", a.name, err)
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
