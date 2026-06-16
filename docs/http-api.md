# HTTP API

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Health check: uptime, last poll time, poll errors, action failures |
| `GET` | `/events` | Recent events (ring buffer) |
| `GET` | `/rules` | Active rules (name, event type, filters, action types) |
| `GET` | `/stats` | Runtime counters (total polls, last poll time, events by type) |

### `/events` Query Parameters

| Parameter | Example | Description |
|---|---|---|
| `type` | `?type=chore.completed` | Filter by event type |
| `limit` | `?limit=10` | Limit number of results |

If `server.auth_token` is set in config, all endpoints require `Authorization: Bearer <token>`.

## Dry-Run Mode

To validate your config and see which actions would fire without actually executing them:

```bash
./skylight-bridge --config config.yaml --dry-run
```

In dry-run mode the bridge polls and detects events normally, but all actions are replaced with log-only no-ops.
