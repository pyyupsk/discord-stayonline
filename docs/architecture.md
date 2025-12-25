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
- `config.go` - Configuration types
- `store.go` - JSON file implementation
- `db_store.go` - PostgreSQL implementation (also handles session state and logs)
- `errors.go` - Custom error types

### WebSocket Hub (`internal/ws/hub.go`)

WebSocket hub for broadcasting real-time status updates to connected frontend clients.

### API Router (`internal/api/router.go`)

HTTP routing with auth middleware. Protected endpoints require API key when `API_KEY` env var is set.

## Session Resumption

Gateway sessions are persisted to enable Discord session resumption:

1. On READY event, session ID and resume URL are saved to `SessionStore`
2. On reconnect, client attempts RESUME before falling back to IDENTIFY
3. On invalid session, stored data is cleared for fresh connection

## Connection Limits

Maximum 35 concurrent connections enforced at manager level. Gateway client uses rotating client properties (5 OS × 7 browsers = 35 combinations) to avoid Discord rate limiting.
