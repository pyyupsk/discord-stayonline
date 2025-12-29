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

## Deployment Warning

> [!WARNING]
> Service restarts (deployments, server reboots) will cause brief disconnections from Discord voice channels. While session resumption is attempted, Discord may invalidate sessions if the downtime exceeds ~30 seconds, resulting in a visible leave/rejoin. This may reset voice channel time counters. Plan deployments accordingly if maintaining continuous presence is important.

## Features

- Maintain online/idle/dnd presence status
- Join voice channels to appear present
- Web UI for configuration and monitoring
- Real-time status updates via WebSocket
- Automatic reconnection with exponential backoff
- Single binary deployment with embedded assets
- Docker support

## Quick Start

### Docker (Recommended)

```bash
docker run -d \
  --name discord-stayonline \
  -p 8080:8080 \
  -e DISCORD_TOKEN=your_token_here \
  ghcr.io/pyyupsk/discord-stayonline:latest
```

### From Source

```bash
git clone https://github.com/pyyupsk/discord-stayonline.git
cd discord-stayonline
cp .env.example .env
# Edit .env with your Discord token
make build && make start
```

Open <http://localhost:8080> in your browser.

## Configuration

| Variable              | Required | Default | Description                              |
| --------------------- | -------- | ------- | ---------------------------------------- |
| `DISCORD_TOKEN`       | Yes      | -       | Your Discord user token                  |
| `API_KEY`             | Yes      | -       | API key for web UI authentication        |
| `DATABASE_URL`        | No       | -       | PostgreSQL URL (for cloud platforms)     |
| `PORT`                | No       | `8080`  | HTTP server port                         |
| `DISCORD_WEBHOOK_URL` | No       | -       | Discord webhook for status notifications |

## Getting Your Discord Token

1. Open Discord in your web browser
2. Press F12 to open Developer Tools
3. Go to the Network tab
4. Send a message or perform any action
5. Find any request to `discord.com/api`
6. Look for the `authorization` header in the request headers
7. Copy the token value

## Documentation

- [API Reference](docs/api.md)
- [Architecture](docs/architecture.md)
- [Development Guide](docs/development.md)

## License

[PolyForm Noncommercial 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0) - See [LICENSE](LICENSE) for details.
