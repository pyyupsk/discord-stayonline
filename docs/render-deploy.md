# Deploying to Render

This guide explains how to deploy Discord Stay Online to [Render](https://render.com).

## Prerequisites

- A Render account
- Your Discord user token

## Option 1: Deploy from Docker Image

1. Go to [Render Dashboard](https://dashboard.render.com)
2. Click "New +" → "Web Service"
3. Select "Deploy an existing image from a registry"
4. Enter the image URL: `ghcr.io/pyyupsk/discord-stayonline:latest`
5. Configure the service:
   - **Name**: `discord-stayonline`
   - **Region**: Choose closest to you
   - **Instance Type**: Free tier works for personal use
6. Add environment variables (see below)
7. Click "Create Web Service"

## Option 2: Deploy from GitHub Repository

1. Go to [Render Dashboard](https://dashboard.render.com)
2. Click "New +" → "Web Service"
3. Connect your GitHub account and select the repository
4. Configure the service:
   - **Name**: `discord-stayonline`
   - **Environment**: Docker
   - **Region**: Choose closest to you
   - **Instance Type**: Free tier works for personal use
5. Add environment variables (see below)
6. Click "Create Web Service"

## Environment Variables

| Variable          | Required | Description                                        |
| ----------------- | -------- | -------------------------------------------------- |
| `DISCORD_TOKEN`   | Yes      | Your Discord user token (mark as Secret)           |
| `DATABASE_URL`    | Yes\*    | PostgreSQL connection URL (see below)              |
| `API_KEY`         | No       | API key for web UI authentication (mark as Secret) |
| `PORT`            | No       | HTTP port (Render sets this automatically)         |
| `ALLOWED_ORIGINS` | No       | Comma-separated allowed origins for WebSocket      |

\*Required for persistent storage on Render's ephemeral filesystem.

## Setting Up PostgreSQL (Recommended)

Render's filesystem is ephemeral, so configuration is lost on restart. Use PostgreSQL for persistence:

1. In Render Dashboard, click "New +" → "PostgreSQL"
2. Configure the database:
   - **Name**: `discord-stayonline-db`
   - **Region**: Same as your web service
   - **Instance Type**: Free tier works for personal use
3. After creation, copy the **Internal Database URL**
4. Add it to your web service as `DATABASE_URL` environment variable

The app auto-creates the required tables on startup.

## Setting Up Authentication (Recommended)

To protect the web UI:

1. Generate a secure key: `openssl rand -hex 32`
2. Add `API_KEY` environment variable with the generated key (mark as Secret)
3. Users must enter this key to access the dashboard

## Build Configuration

To avoid unnecessary rebuilds, configure **Included Paths** in your Render service settings:

```txt
cmd/**
internal/**
web/**
go.mod
go.sum
Dockerfile
Makefile
```

Only changes to source code, dependencies, and build files will trigger new builds. Documentation and config changes are ignored.

## Important Notes

- **Port**: Render automatically sets the `PORT` environment variable. The service uses this.
- **Health Check**: The service exposes `/health` which Render uses for health monitoring.
- **Persistent Storage**: Use `DATABASE_URL` with PostgreSQL for configuration persistence. Without it, settings are lost on restart.
- **Session Resumption**: PostgreSQL also stores Gateway session data, enabling faster reconnects after restarts.

## Accessing the Service

After deployment:

1. Wait for the service to start (shows "Live" in dashboard)
2. Click the service URL (e.g., `https://discord-stayonline-xxxx.onrender.com`)
3. If `API_KEY` is set, enter your API key to log in
4. Accept the TOS warning
5. Configure your server connections

## Monitoring

Set up [UptimeRobot](https://uptimerobot.com) or similar to monitor:

```http
HEAD https://your-service.onrender.com/health
```

Returns `200 OK`. Use GET for detailed JSON with status, uptime, and connection info.

## Troubleshooting

### Service Keeps Restarting

- Check logs in Render dashboard
- Verify `DISCORD_TOKEN` is set correctly
- Token may be invalid or expired

### Cannot Connect

- Ensure the service shows "Live" status
- Check browser console for WebSocket errors
- Verify `ALLOWED_ORIGINS` includes your domain

### Configuration Lost After Restart

- Ensure `DATABASE_URL` is set with a valid PostgreSQL connection URL
- Check database connection in logs
