# Configuration

See [config.example.yaml](../config.example.yaml) for a complete example.

## Interactive Setup

Generate a config interactively:

```bash
make generate-config
# Or: ./skylight-bridge --generate-config
```

This walks you through entering your Skylight credentials, frame ID, polling interval, and optional Home Assistant settings, then writes `config.yaml`.

Alternatively, copy and edit the example:

```bash
cp config.example.yaml config.yaml
```

## Authentication

Provide either `refresh_token` + `device_fingerprint` (recommended) or `user_id` + `token`.

### Getting `refresh_token` + `device_fingerprint`

Use the [go-skylight](https://github.com/sebrandon1/go-skylight) CLI's `login` command, which performs a headless OAuth2 login and prints both values:

```bash
go-skylight login --email you@example.com --password secret
```

This prints an access token, refresh token, and device fingerprint. Copy the `Refresh Token` and `Fingerprint` values into `auth.refresh_token` and `auth.device_fingerprint` in your config -- the device fingerprint is just a random UUID that identifies this "device" to Skylight, so any value you generate yourself works too, as long as you reuse the same one on every login. The refresh token does not expire under normal use, so this is a one-time setup step.

Add `--save` to write the credentials directly to `~/.skylight/config` instead of copying them by hand:

```bash
go-skylight login --email you@example.com --password secret --save
```

### Getting Your Frame ID

Use the [go-skylight](https://github.com/sebrandon1/go-skylight) CLI:

```bash
go-skylight login --email you@example.com --password secret --save
go-skylight get frame info
```

## HTTP Server Authentication

Protect the HTTP endpoints with a bearer token:

```yaml
server:
  auth_token: "your-secret-token"
```

Requests must then include `Authorization: Bearer your-secret-token`.
