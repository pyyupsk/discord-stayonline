.PHONY: dev start build test lint lint-fix format clean docker help

# Binary output
BINARY_NAME=discord-stayonline
BUILD_DIR=bin

# Go settings
GOFLAGS=-ldflags="-s -w"

# ============================================================================
# Main Commands
# ============================================================================

# Development mode - run with hot reload feel (rebuilds on each run)
dev:
	@echo "Starting development server..."
	go run ./cmd/server

# Start the built binary
start: build
	@echo "Starting production server..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Build for production
build:
	@echo "Building..."
	@cd web && bun install && bun run build
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	@echo "Done! Binary: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	go test -v ./...

# Run linter (Go + Web)
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run
	cd web && bun run lint

# Format code (Go + Web)
format:
	go fmt ./...
	cd web && bun run format

# Fix lint errors
lint-fix:
	cd web && bun run lint:fix

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR) web/dist web/node_modules coverage.out coverage.html
	@go clean
	@echo "Cleaned!"

# ============================================================================
# Web UI Commands
# ============================================================================

# Run web dev server with hot reload
web-dev:
	cd web && bun run dev

# Build web UI only
web-build:
	cd web && bun install && bun run build

# Install web dependencies
web-install:
	cd web && bun install

# ============================================================================
# Docker Commands
# ============================================================================

# Build Docker image
docker:
	docker build -t ghcr.io/pyyupsk/discord-stayonline:latest .

# Run Docker container
docker-run:
	docker run -d \
		--name discord-stayonline \
		-p 8080:8080 \
		-e DISCORD_TOKEN=$${DISCORD_TOKEN} \
		-v $$(pwd)/config.json:/app/config.json \
		ghcr.io/pyyupsk/discord-stayonline:latest

# Build multi-arch Docker image
docker-push:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t ghcr.io/pyyupsk/discord-stayonline:latest \
		--push .

# ============================================================================
# CI/Quality Commands
# ============================================================================

# Run all checks (CI pipeline)
ci: lint test check-size
	@echo "All checks passed!"

# Check code quality
check: format lint test

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Check gateway package coverage (>=80% required)
coverage-gateway:
	go test -coverprofile=coverage.out ./internal/gateway/...
	go tool cover -func=coverage.out | grep total

# Verify bundle size (<500KB gzipped)
check-size:
	@echo "Checking bundle size..."
	@tar -czf /tmp/web-bundle.tar.gz web/dist/
	@size=$$(stat -c%s /tmp/web-bundle.tar.gz 2>/dev/null || stat -f%z /tmp/web-bundle.tar.gz 2>/dev/null); \
	echo "Gzipped: $$size bytes"; \
	if [ "$$size" -gt 512000 ]; then \
		echo "ERROR: Exceeds 500KB limit"; \
		exit 1; \
	fi
	@rm -f /tmp/web-bundle.tar.gz

# ============================================================================
# Setup Commands
# ============================================================================

# Install all dependencies
install:
	go mod download
	cd web && bun install
	@echo "Dependencies installed!"

# Install dev tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Tidy dependencies
tidy:
	go mod tidy
	go mod verify

# ============================================================================
# Help
# ============================================================================

help:
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  dev          Run development server"
	@echo "  web          Run web UI dev server (hot reload)"
	@echo "  build        Build for production"
	@echo "  start        Build and run production server"
	@echo ""
	@echo "Testing:"
	@echo "  test         Run tests"
	@echo "  lint         Run linter (Go + Web)"
	@echo "  lint-fix     Fix lint errors (Web)"
	@echo "  format       Format code (Go + Web)"
	@echo "  check        Run format + lint + test"
	@echo "  coverage     Generate coverage report"
	@echo ""
	@echo "Docker:"
	@echo "  docker       Build Docker image"
	@echo "  docker-run   Run Docker container"
	@echo "  docker-push  Build and push multi-arch image"
	@echo ""
	@echo "Setup:"
	@echo "  install      Install all dependencies"
	@echo "  clean        Remove build artifacts"
	@echo ""
