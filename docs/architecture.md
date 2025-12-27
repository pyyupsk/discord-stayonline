# Architecture

## Project Structure

```filetree
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

## Core Data Flow

```flow
main.go → SessionManager → Gateway Clients → Discord WebSocket
              ↓
         WebSocket Hub → Frontend (real-time status updates)
              ↓
         ConfigStore → PostgreSQL / JSON file
```

## Key Components

### Gateway Client (`internal/gateway/client.go`)

Discord Gateway WebSocket client. Handles IDENTIFY, RESUME, heartbeating, and voice state updates. Uses client property rotation (OS/browser combinations) to avoid rate limits across multiple connections.

### Session Manager (`internal/manager/manager.go`)

Manages multiple Gateway sessions. Handles join/rejoin/exit operations, automatic reconnection with exponential backoff, and session persistence for resumption. Broadcasts status changes to WebSocket hub.

### Configuration (`internal/config/`)

Configuration persistence layer with interface abstraction:

- `interface.go` - `ConfigStore` interface
- `config.go` - Configuration types and `SessionState`
- `errors.go` - Custom error types
- `store/file.go` - JSON file implementation
- `store/postgres.go` - PostgreSQL implementation (also handles session state and logs)
- `store/models.go` - GORM database models

### WebSocket Hub (`internal/ws/hub.go`)

WebSocket hub for broadcasting real-time status updates to connected frontend clients.

### API Router (`internal/api/`)

HTTP routing organized by function:

- `router.go` - Route definitions
- `handlers/` - HTTP request handlers
- `middleware/` - Auth middleware (API_KEY is required)
- `responses/` - JSON response helpers

## Session Resumption

Gateway sessions are persisted to enable Discord session resumption:

1. On READY event, session ID and resume URL are saved to `SessionStore`
2. On reconnect, client attempts RESUME before falling back to IDENTIFY
3. On invalid session, stored data is cleared for fresh connection

## Connection Limits

Maximum 35 concurrent connections enforced at manager level. Gateway client uses rotating client properties (5 OS × 7 browsers = 35 combinations) to avoid Discord rate limiting.
