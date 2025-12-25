# API Reference

All `/api/*` endpoints (except auth) require authentication when `API_KEY` is set.

## Health Check

```http
GET /health
Response: 200 OK, JSON with status, uptime, connections, runtime, memory info

HEAD /health
Response: 200 OK (for simple uptime checks)
```

## Authentication

```http
GET /api/auth/check
Response: {"authenticated": bool, "auth_required": bool}

POST /api/auth/login
Body: {"api_key": "..."}
Response: 200 OK (sets HTTP-only cookie)

POST /api/auth/logout
Response: 200 OK (clears cookie)
```

## TOS Acknowledgment

```http
POST /api/acknowledge-tos
Body: {"acknowledged": true}
Response: 200 OK
```

## Configuration

```http
GET /api/config
Response: {"servers": [...], "status": "online|idle|dnd", "tos_acknowledged": bool}

POST /api/config
Body: {"servers": [...], "status": "..."}  // Full replacement (max 35 entries)

PUT /api/config
Body: {"servers": [...], "status": "..."}  // Partial update, merge by ID
```

## Server Actions

```http
GET /api/statuses
Response: {"server_id": "status", ...}

POST /api/servers/{id}/action
Body: {"action": "join" | "rejoin" | "exit"}
```

## Discord Info

```http
GET /api/discord/server-info?guild_id=...&channel_id=...
Response: {"guild_id": "...", "guild_name": "...", "channel_id": "...", "channel_name": "..."}

POST /api/discord/bulk-info
Body: [{"guild_id": "...", "channel_id": "..."}, ...]
Response: [{"guild_id": "...", "guild_name": "...", ...}, ...]

GET /api/discord/guilds
Response: [{guild objects}]

GET /api/discord/guilds/{id}
Response: [{channel objects}]
```

## Activity Logs

```http
GET /api/logs
Response: [{log entries}]
```

## WebSocket Status Updates

```http
WS /ws
Messages: {"type": "status", "server_id": "...", "status": "...", "message": "..."}
```

## Connection States

| Status         | Description                              |
| -------------- | ---------------------------------------- |
| `disconnected` | Not connected                            |
| `connecting`   | Attempting to connect                    |
| `connected`    | Successfully connected and authenticated |
| `error`        | Connection failed with an error          |
| `backoff`      | Waiting before reconnect attempt         |
