# Discord Stay Online

A self-hosted service that maintains Discord account presence by managing persistent Gateway WebSocket connections.

## Terms of Service Warning

> **IMPORTANT: READ BEFORE USE**
>
> This tool uses Discord user tokens to maintain presence status. Using user tokens with automated tools **may violate Discord's Terms of Service** and could result in **account suspension or termination**.
>
> By using this software, you acknowledge:
> - You understand the risks involved with using user tokens
> - You accept full responsibility for any consequences to your Discord account
> - The authors are not responsible for any actions taken against your account
>
> **USE AT YOUR OWN RISK**

## Features

- Maintain online/idle/dnd presence status
- Join voice channels to appear present
- Web UI for configuration and monitoring
- Real-time status updates via WebSocket
- Automatic reconnection with exponential backoff
- Health endpoint for uptime monitoring
- Single binary deployment with embedded assets
- Docker support

## Quick Start

### From Source

```bash
# Clone and build
git clone https://github.com/pyyupsk/discord-stayonline.git
cd discord-stayonline
make build

# Configure
cp .env.example .env
# Edit .env with your Discord token

# Run
make run
```

Open http://localhost:8080 in your browser.

### Docker

```bash
docker run -d \
  --name discord-stayonline \
  -p 8080:8080 \
  -e DISCORD_TOKEN=your_token_here \
  ghcr.io/pyyupsk/discord-stayonline:latest
```

## Environment Variables

| Variable | Required | Default | Description |
| -------- | -------- | ------- | ----------- |
| `DISCORD_TOKEN` | Yes | - | Your Discord user token |
| `PORT` | No | `8080` | HTTP server port |
| `ALLOWED_ORIGINS` | No | `localhost` | Comma-separated allowed origins for WebSocket |

## Getting Your Discord Token

1. Open Discord in your web browser
2. Press F12 to open Developer Tools
3. Go to the Network tab
4. Send a message or perform any action
5. Find any request to `discord.com/api`
6. Look for the `authorization` header in the request headers
7. Copy the token value

## Health Monitoring

Set up UptimeRobot or similar to ping:

```
GET http://your-server:8080/health
```

Expected response: `200 OK` with body `OK`

## Development

```bash
# Run tests
make test

# Run with coverage
make coverage

# Format code
make fmt

# Run linter
make lint
```

## Architecture

```
cmd/server/         - Entry point
internal/
  config/           - Configuration types and persistence
  gateway/          - Discord Gateway WebSocket client
  manager/          - Session management for multiple connections
  api/              - HTTP API handlers
  ws/               - WebSocket hub for UI updates
  ui/               - Static asset embedding
web/                - Frontend assets (HTML, JS, CSS)
tests/              - Integration tests
```

## License

MIT
