# Development Guide

## Prerequisites

- Go 1.25+
- Node.js 22+ with Bun
- Docker (optional)

## Commands

```bash
# Development
make dev          # Run Go server with hot reload
make web-dev      # Run Vue frontend dev server (Vite HMR)

# Build & Run
make build        # Build production binary (builds web first, embeds assets)
make start        # Build and run production server

# Testing
make test         # Run all Go tests
go test -v ./internal/gateway/...   # Run tests for specific package
make coverage     # Generate coverage report (coverage.html)

# Code Quality
make lint         # Run golangci-lint + ESLint
make format       # Format Go + frontend code
make lint-fix     # Auto-fix ESLint errors
```

## Code Style

- **Go**: Standard conventions, `slog` for structured logging
- **Vue**: Composition API with TypeScript, shadcn-vue components, Tailwind CSS v4
- **Frontend**: Uses `bun` as package manager

## Configuration Options

### PostgreSQL Storage

Set `DATABASE_URL` to use PostgreSQL for config storage instead of a file. Recommended for platforms like Render where the filesystem is ephemeral.

```bash
DATABASE_URL=postgres://user:password@host:5432/dbname
```

The app auto-creates the required table on startup.

### Authentication (Required)

The web UI requires API key authentication. The server will not start without it:

```bash
# Generate a secure key
echo "sk-live_$(openssl rand -base64 48 | tr -d '=+/')"

# Add to .env
API_KEY=your_generated_key_here
```

Users must enter the API key to access the dashboard. The key is stored in an HTTP-only cookie (7-day expiry).

### Health Monitoring

Set up UptimeRobot or similar to ping:

```http
GET http://your-server:8080/health
```

Returns `200 OK` with JSON containing status, uptime, connections, and runtime info.
