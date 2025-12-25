.PHONY: build run test docker-build clean fmt lint vet

# Binary output
BINARY_NAME=discord-stayonline
BUILD_DIR=bin

# Go settings
GOFLAGS=-ldflags="-s -w"

# Default target
all: build

# Build the binary
build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Check gateway package coverage (constitution requirement: >=80%)
coverage-gateway:
	go test -coverprofile=coverage.out ./internal/gateway/...
	go tool cover -func=coverage.out | grep total

# Build Docker image
docker-build:
	docker build -t ghcr.io/pyyupsk/discord-stayonline:latest .

# Build multi-arch Docker image
docker-build-multiarch:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t ghcr.io/pyyupsk/discord-stayonline:latest \
		--push .

# Run Docker container
docker-run:
	docker run -d \
		--name discord-stayonline \
		-p 8080:8080 \
		-e DISCORD_TOKEN=$${DISCORD_TOKEN} \
		-v $$(pwd)/config.json:/app/config.json \
		ghcr.io/pyyupsk/discord-stayonline:latest

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Verify bundle size (constitution requirement: <500KB gzipped)
check-bundle-size:
	@echo "Checking web bundle size..."
	@tar -czf /tmp/web-bundle.tar.gz web/
	@size=$$(stat -c%s /tmp/web-bundle.tar.gz 2>/dev/null || stat -f%z /tmp/web-bundle.tar.gz 2>/dev/null); \
	echo "Gzipped bundle size: $$size bytes"; \
	if [ "$$size" -gt 512000 ]; then \
		echo "ERROR: Bundle size exceeds 500KB limit"; \
		exit 1; \
	fi
	@rm -f /tmp/web-bundle.tar.gz

# Install development dependencies
dev-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Tidy and verify dependencies
tidy:
	go mod tidy
	go mod verify

# Show help
help:
	@echo "Available targets:"
	@echo "  build             - Build the binary"
	@echo "  run               - Run the server"
	@echo "  test              - Run tests"
	@echo "  coverage          - Run tests with coverage report"
	@echo "  coverage-gateway  - Check gateway package coverage"
	@echo "  docker-build      - Build Docker image"
	@echo "  docker-run        - Run Docker container"
	@echo "  clean             - Clean build artifacts"
	@echo "  fmt               - Format code"
	@echo "  vet               - Run go vet"
	@echo "  lint              - Run linter"
	@echo "  check-bundle-size - Verify web bundle <500KB"
	@echo "  tidy              - Tidy dependencies"
	@echo "  help              - Show this help"
