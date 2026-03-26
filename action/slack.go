package action

// NewSlackAction creates an action that posts events to a Slack channel via incoming webhook.
// Supported config keys:
//   - webhook_url: Slack incoming webhook URL (required)
//   - message: Go text/template for the message content (optional)
func NewSlackAction(config map[string]any) (Action, error) {
	return newChatAction(config, "slack", "text")
}
