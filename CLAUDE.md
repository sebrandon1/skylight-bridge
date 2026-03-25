# skylight-bridge

Event-driven bridge for the Skylight Calendar API. Polls for state changes and dispatches configurable actions (webhooks, Home Assistant, logging).

## Go Version

Go 1.26.1 (see `go.mod`)

## Dependencies

- `github.com/sebrandon1/go-skylight` -- Skylight API client library
- `gopkg.in/yaml.v3` -- YAML config parsing

## Build / Test / Lint

```bash
make build          # go build with version injection via ldflags
make test           # go test ./... -v
make lint           # golangci-lint run ./...
make vet            # go vet ./...
make clean          # rm -f skylight-bridge
```

Run `make lint` before committing and fix any issues.

## Project Structure

```
main.go                 # Entrypoint: load config, wire components, signal handling
config/
  config.go             # YAML config types, loader, validation
  config_test.go
engine/
  event.go              # Event type constants and Event struct
  detector.go           # State-diff logic: compare snapshots, emit events
  detector_test.go
  bus.go                # Simple pub/sub fan-out
  bus_test.go
  poller.go             # Multi-resource poll loop (chores + rewards)
action/
  action.go             # Action interface and Factory type
  log.go                # Log action (structured stdout)
  log_test.go
  webhook.go            # Webhook action (POST JSON to URL)
  webhook_test.go
  discord.go            # Discord webhook action
  discord_test.go
  webhook_test.go
  homeassistant.go      # Home Assistant service call / webhook
  homeassistant_test.go
rules/
  rules.go              # Rule matching: event type + filters -> action dispatch
  rules_test.go
state/
  state.go              # JSON file state persistence (atomic writes)
  state_test.go
server/
  server.go             # HTTP server: GET /events, GET /healthz
  server_test.go
```

## Event Types

- `chore.completed` -- Chore status changes from pending to completed
- `chore.all_completed` -- All chores for a kid on today are completed (fires once per kid per day)
- `reward.redeemed` -- Reward's Redeemed flag flips to true

## Architecture

1. **Poller** fetches chores, rewards, and categories in parallel each interval
2. **Detector** diffs new state against previous snapshot and emits events
3. **Bus** fans out events to subscribers (rules engine, HTTP server)
4. **Rules Engine** matches events against configured rules and dispatches actions
5. **State Store** persists detector state to JSON for restart durability

## Notes

- Tests use `httptest.NewServer` for action tests (webhook, HA)
- The detector is not goroutine-safe; the poller serializes access
- State file defaults to `~/.skylight-bridge/state.json`
- Do not add `Co-Authored-By` lines to commit messages
