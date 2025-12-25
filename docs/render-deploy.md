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
6. Add environment variable:
   - **Key**: `DISCORD_TOKEN`
   - **Value**: Your Discord token (mark as Secret)
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
5. Add environment variable:
   - **Key**: `DISCORD_TOKEN`
   - **Value**: Your Discord token (mark as Secret)
6. Click "Create Web Service"

## Environment Variables

| Variable | Required | Description |
| -------- | -------- | ----------- |
| `DISCORD_TOKEN` | Yes | Your Discord user token (Secret) |
| `PORT` | No | HTTP port (Render sets this automatically) |
| `ALLOWED_ORIGINS` | No | Comma-separated allowed origins for WebSocket |

## Important Notes

- **Port**: Render automatically sets the `PORT` environment variable. The service uses this.
- **Health Check**: The service exposes `/health` which Render uses for health monitoring.
- **Persistent Storage**: Server configuration is stored in memory. For persistence across restarts, consider mounting a disk for `config.json`.

## Accessing the Service

After deployment:
1. Wait for the service to start (shows "Live" in dashboard)
2. Click the service URL (e.g., `https://discord-stayonline-xxxx.onrender.com`)
3. Accept the TOS warning
4. Configure your server connections

## Monitoring

Set up [UptimeRobot](https://uptimerobot.com) or similar to monitor:
```
GET https://your-service.onrender.com/health
```

Expected response: `200 OK` with body `OK`

## Troubleshooting

### Service Keeps Restarting
- Check logs in Render dashboard
- Verify `DISCORD_TOKEN` is set correctly
- Token may be invalid or expired

### Cannot Connect
- Ensure the service shows "Live" status
- Check browser console for WebSocket errors
- Verify `ALLOWED_ORIGINS` includes your domain
