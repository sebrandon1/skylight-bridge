# skylight-bridge

Event-driven bridge for [Skylight Calendar](https://www.ourskylight.com/). Polls the Skylight API for state changes (chore completions, reward redemptions) and dispatches configurable actions like webhooks, Home Assistant calls, and structured logging.

## Use Cases

- **Kid redeems a reward** (e.g., "Invest $20 in VOO") -> fire a webhook to trigger downstream logic
- **Kid completes all daily chores** -> flash the house lights via Home Assistant
- **Any chore/reward activity** -> log for tracking and visibility

## Quick Start

1. Generate a config:
   ```bash
   make generate-config
   ```

2. Build and run:
   ```bash
   make build
   ./skylight-bridge --config config.yaml
   ```

3. Check health:
   ```bash
   curl http://localhost:8080/healthz
   ```

## Key Features

- **5 action types** -- log, webhook, Discord, Slack, Home Assistant
- **Event filtering** -- route actions by assignee, chore, or any event field
- **Retry with backoff** -- exponential retry on action failures
- **HTTP API** -- health checks, event history, rule inspection, runtime stats
- **Dry-run mode** -- validate config without executing actions
- **Docker support** -- pre-built images on Docker Hub

## Guides

| Guide | Description |
|---|---|
| [Events](docs/events.md) | Event types and data fields available in templates |
| [Actions](docs/actions.md) | Action types, retry configuration, and filters |
| [Configuration](docs/configuration.md) | Config setup, authentication, and frame ID |
| [HTTP API](docs/http-api.md) | Endpoints, query parameters, and dry-run mode |
| [Deployment](docs/deployment.md) | Docker, Docker Compose, and local image builds |

## Development

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
