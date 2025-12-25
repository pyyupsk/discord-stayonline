# syntax=docker/dockerfile:1

# Web build stage
FROM oven/bun:1-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# Go build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code and web assets
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist

# Build binary with optimizations
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Runtime stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Expose port
EXPOSE 8080

# Set environment defaults
ENV PORT=8080

# Run the binary
ENTRYPOINT ["/app/server"]
