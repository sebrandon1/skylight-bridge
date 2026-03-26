package action

// NewDiscordAction creates an action that posts events to a Discord channel via webhook.
// Supported config keys:
//   - webhook_url: Discord webhook URL (required)
//   - message: Go text/template for the message content (optional)
func NewDiscordAction(config map[string]any) (Action, error) {
	return newChatAction(config, "discord", "content")
}
