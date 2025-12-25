# Discord Stay Online

A self-hosted service that maintains Discord account presence by managing persistent Gateway WebSocket connections.

## Terms of Service Warning

> [!CAUTION]
> This tool uses Discord user tokens to maintain presence status. Using user tokens with automated tools **may violate Discord's Terms of Service** and could result in **account suspension or termination**.
>
> By using this software, you acknowledge:
>
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
- API key authentication (optional)
- Auto-fetch server/channel names from Discord

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

Open <http://localhost:8080> in your browser.

### Docker

```bash
docker run -d \
  --name discord-stayonline \
  -p 8080:8080 \
  -e DISCORD_TOKEN=your_token_here \
  ghcr.io/pyyupsk/discord-stayonline:latest
```

## Environment Variables

| Variable          | Required | Default       | Description                                       |
| ----------------- | -------- | ------------- | ------------------------------------------------- |
| `DISCORD_TOKEN`   | Yes      | -             | Your Discord user token                           |
| `DATABASE_URL`    | No       | -             | PostgreSQL connection URL (for persistent config) |
| `API_KEY`         | No       | -             | API key for web UI authentication                 |
| `PORT`            | No       | `8080`        | HTTP server port                                  |
| `CONFIG_PATH`     | No       | `config.json` | Path to config file (if not using DATABASE_URL)   |
| `ALLOWED_ORIGINS` | No       | `localhost`   | Comma-separated allowed origins for WebSocket     |

### PostgreSQL Storage (Recommended for cloud deployments)

Set `DATABASE_URL` to use PostgreSQL for config storage instead of a file. This is recommended for platforms like Render where the filesystem is ephemeral.

```bash
DATABASE_URL=postgres://user:password@host:5432/dbname
```

The app auto-creates the required table on startup.

### Authentication

To protect the web UI with an API key:

```bash
# Generate a secure key
openssl rand -hex 32

# Add to .env
API_KEY=your_generated_key_here
```

When `API_KEY` is set, users must enter the key to access the dashboard. The key is stored in an HTTP-only cookie (24h expiry).

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

```http
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

## API Reference

All `/api/*` endpoints (except auth) require authentication when `API_KEY` is set.

### Health Check

```http
GET /health
Response: 200 OK, body: "OK"
```

### Authentication

```http
GET /api/auth/check
Response: {"authenticated": bool, "auth_required": bool}

POST /api/auth/login
Body: {"api_key": "..."}
Response: 200 OK (sets HTTP-only cookie)

POST /api/auth/logout
Response: 200 OK (clears cookie)
```

### TOS Acknowledgment

```http
POST /api/acknowledge-tos
Body: {"acknowledged": true}
Response: 200 OK
```

### Configuration

```http
GET /api/config
Response: {"servers": [...], "status": "online|idle|dnd", "tos_acknowledged": bool}

POST /api/config
Body: {"servers": [...], "status": "..."}  // Full replacement (max 35 entries)

PUT /api/config
Body: {"servers": [...], "status": "..."}  // Partial update, merge by ID
```

### Server Actions

```http
POST /api/servers/{id}/action
Body: {"action": "join" | "rejoin" | "exit"}
```

### Discord Info

```http
GET /api/discord/server-info?guild_id=...&channel_id=...
Response: {"guild_id": "...", "guild_name": "...", "channel_id": "...", "channel_name": "..."}

POST /api/discord/bulk-info
Body: [{"guild_id": "...", "channel_id": "..."}, ...]
Response: [{"guild_id": "...", "guild_name": "...", ...}, ...]
```

### WebSocket Status Updates

```http
WS /ws
Messages: {"type": "status", "server_id": "...", "status": "...", "message": "..."}
```

## Architecture

```filestree
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

### Connection States

| Status         | Description                              |
| -------------- | ---------------------------------------- |
| `disconnected` | Not connected                            |
| `connecting`   | Attempting to connect                    |
| `connected`    | Successfully connected and authenticated |
| `error`        | Connection failed with an error          |
| `backoff`      | Waiting before reconnect attempt         |

## License

[PolyForm Noncommercial 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0) - See [LICENSE](LICENSE) for details.

- Personal and noncommercial use allowed
- Modification and distribution allowed
- No commercial use or selling
- Attribution required
