# skylight-bridge

Event-driven bridge for [Skylight Calendar](https://www.ourskylight.com/). Polls the Skylight API for state changes (chore completions, reward redemptions) and dispatches configurable actions like webhooks, Home Assistant calls, and structured logging.

## Use Cases

- **Kid redeems a reward** (e.g., "Invest $20 in VOO") -> fire a webhook to trigger downstream logic
- **Kid completes all daily chores** -> flash the house lights via Home Assistant
- **Any chore/reward activity** -> log for tracking and visibility

## Quick Start

1. Generate a config interactively:
   ```bash
   make generate-config
   # Or: ./skylight-bridge --generate-config
   ```
   This walks you through entering your Skylight credentials, frame ID, polling interval, and optional Home Assistant settings, then writes `config.yaml`.

   Alternatively, copy and edit the example:
   ```bash
   cp config.example.yaml config.yaml
   ```

2. Build and run:
   ```bash
   make build
   ./skylight-bridge --config config.yaml
   ```

3. Check health and recent events:
   ```bash
   curl http://localhost:8080/healthz
   curl http://localhost:8080/events
   ```

## Event Types

| Event | Description |
|---|---|
| `chore.completed` | A chore's status changed from pending to completed |
| `chore.all_completed` | All chores for a given kid on today's date are completed (fires once per kid per day) |
| `reward.redeemed` | A reward was redeemed |

## Action Types

### `log`
Logs the event to stdout as structured JSON.

```yaml
actions:
  - type: log
    config:
      message: "{{.assignee_name}} completed {{.chore_title}}"  # optional Go template
```

### `webhook`
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
```

### `discord`
Posts messages to a Discord channel via webhook.

```yaml
actions:
  - type: discord
    config:
      webhook_url: "https://discord.com/api/webhooks/1234/abcd"
      message: "{{.assignee_name}} completed **{{.chore_title}}"  # optional Go template
```

If no `message` template is provided, a default human-readable message is generated based on the event type.

### `homeassistant`
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

# Webhook trigger
actions:
  - type: homeassistant
    config:
      url: "http://homeassistant.local:8123"
      webhook_id: "my-skylight-hook"
```

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

## HTTP Endpoints

- `GET /healthz` - Health check with uptime
- `GET /events` - Recent events (ring buffer)
  - `?type=chore.completed` - Filter by event type
  - `?limit=10` - Limit results

## Configuration

See [config.example.yaml](config.example.yaml) for a complete example.

### Authentication

Provide either email + password (auto-login) or user_id + token (pre-existing credentials).

### Getting Your Frame ID

Use the [go-skylight](https://github.com/sebrandon1/go-skylight) CLI:
```bash
go-skylight login --email you@example.com --password secret --save
go-skylight get frame info
```

## Docker

Pull from Docker Hub:
```bash
docker pull sebrandon1/skylight-bridge:latest
```

Run with your config file:
```bash
docker run -d \
  --name skylight-bridge \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/config/config.yaml:ro \
  -v skylight-bridge-data:/data \
  sebrandon1/skylight-bridge:latest
```

The container expects:
- `/config/config.yaml` — your configuration file (mount read-only)
- `/data/` — state persistence directory (use a named volume)

Set `state_file: /data/state.json` in your config to persist state across container restarts.

### Docker Compose

```yaml
services:
  skylight-bridge:
    image: sebrandon1/skylight-bridge:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/config/config.yaml:ro
      - bridge-data:/data
    restart: unless-stopped

volumes:
  bridge-data:
```

### Build Locally

```bash
make docker-build    # Build image tagged skylight-bridge:dev
make docker-run      # Run with local config.yaml
```

## Build from Source

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linters
make vet      # Run go vet
make clean    # Remove binary
```

## Requirements

- Go 1.26.1+ (for building from source)
- A Skylight account with at least one frame
