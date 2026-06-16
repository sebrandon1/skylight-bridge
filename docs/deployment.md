# Deployment

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
- `/config/config.yaml` -- your configuration file (mount read-only)
- `/data/` -- state persistence directory (use a named volume)

Set `state_file: /data/state.json` in your config to persist state across container restarts.

## Docker Compose

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

## Build Docker Image Locally

```bash
make docker-build    # Build image tagged skylight-bridge:dev
make docker-run      # Run with local config.yaml
```
