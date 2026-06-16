# Actions

## `log`

Logs the event to stdout as structured JSON.

```yaml
actions:
  - type: log
    config:
      message: "{{.assignee_name}} completed {{.chore_title}}"  # optional Go template
```

## `webhook`

Sends an HTTP request to a URL.

```yaml
actions:
  - type: webhook
    config:
      url: "https://example.com/hook"
      method: "POST"                    # optional, default: POST
      headers:                           # optional
        Authorization: "Bearer xyz"
      body_template: '{"kid": "{{.child_name}}"}'  # optional, default: full event JSON
      timeout: "30s"                    # optional, default: 10s
```

## `discord`

Posts messages to a Discord channel via webhook.

```yaml
actions:
  - type: discord
    config:
      webhook_url: "https://discord.com/api/webhooks/1234/abcd"
      message: "{{.assignee_name}} completed **{{.chore_title}}"  # optional Go template
      timeout: "30s"                                               # optional, default: 10s
```

If no `message` template is provided, a default human-readable message is generated based on the event type.

## `slack`

Posts messages to a Slack channel via incoming webhook.

```yaml
actions:
  - type: slack
    config:
      webhook_url: "https://hooks.slack.com/services/T.../B.../xxx"
      message: "{{.assignee_name}} completed *{{.chore_title}}*"  # optional Go template
      timeout: "30s"                                               # optional, default: 10s
```

If no `message` template is provided, a default human-readable message is generated based on the event type.

## `homeassistant`

Calls a Home Assistant service or webhook.

```yaml
# Service call (e.g., turn on a light)
actions:
  - type: homeassistant
    config:
      url: "http://homeassistant.local:8123"
      token: "HA_LONG_LIVED_ACCESS_TOKEN"
      service: "light.turn_on"
      entity_id: "light.living_room"
      timeout: "30s"                    # optional, default: 10s

# Webhook trigger
actions:
  - type: homeassistant
    config:
      url: "http://homeassistant.local:8123"
      webhook_id: "my-skylight-hook"
      timeout: "30s"                    # optional, default: 10s
```

## Retry

Any action can be retried on failure with exponential backoff:

```yaml
rules:
  - name: "notify-on-completion"
    event: "chore.completed"
    actions:
      - type: webhook
        retry_attempts: 3   # total attempts (default: 1, no retry)
        retry_delay: "1s"   # initial delay before retry; doubles each attempt, capped at 30s
        config:
          url: "https://example.com/hook"
```

On each failure the bridge waits `retry_delay`, `retry_delay*2`, etc. before the next attempt. If the shutdown signal fires during a retry wait the attempt is abandoned immediately.

## Filters

Rules can filter events by any field in the event data:

```yaml
rules:
  - name: "alice-chores-only"
    event: "chore.completed"
    filters:
      assignee_name: "Alice"
    actions:
      - type: log
```
