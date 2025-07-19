# Makefile for WebCrawler

# Variables
BINARY_NAME=webcrawler
BUILD_DIR=build
DOCKER_IMAGE=webcrawler
DOCKER_TAG=latest

# Go Commands
.PHONY: build
build:
	@echo "🔨 Building application..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: run
run:
	@echo "🚀 Starting application..."
	go run main.go

.PHONY: test
test:
	@echo "🧪 Running tests..."
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "📊 Tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage Report: coverage.html"

.PHONY: clean
clean:
	@echo "🧹 Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean

.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...

.PHONY: lint
lint:
	@echo "🔍 Linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.PHONY: deps
deps:
	@echo "📦 Installing dependencies..."
	go mod tidy
	go mod download

.PHONY: install
install: deps build
	@echo "✅ Installation completed"

# Docker Commands
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run:
	@echo "🐳 Starting Docker container..."
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-compose-up
docker-compose-up:
	@echo "🐳 Starting with Docker Compose..."
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down:
	@echo "🐳 Stopping Docker Compose..."
	docker-compose down

.PHONY: docker-clean
docker-clean:
	@echo "🧹 Cleaning Docker..."
	docker system prune -f
	docker image prune -f

# Development Commands
.PHONY: dev
dev:
	@echo "🛠️  Development Mode..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Or use: make run"; \
		make run; \
	fi

.PHONY: debug
debug:
	@echo "🐛 Debug Mode..."
	LOG_LEVEL=debug go run main.go

# Version Commands
.PHONY: version
version:
	@echo "📋 Version information:"
	@go run main.go --version 2>/dev/null || echo "Version: $(shell cat VERSION)"

.PHONY: release
release:
	@echo "🏷️  Creating release..."
	@echo "Current version: $(shell cat VERSION)"
	@echo "Follow these steps:"
	@echo "1. git tag v$(shell cat VERSION)"
	@echo "2. git push origin v$(shell cat VERSION)"
	@echo "3. Create release on GitHub"

# Utility Commands
.PHONY: help
help:
	@echo "📚 Available commands:"
	@echo ""
	@echo "🔨 Build:"
	@echo "  build          - Build application"
	@echo "  install        - Install dependencies and build"
	@echo ""
	@echo "🚀 Run:"
	@echo "  run            - Start application"
	@echo "  dev            - Development mode with hot reload"
	@echo "  debug          - Debug mode"
	@echo ""
	@echo "🧪 Test:"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Tests with coverage report"
	@echo ""
	@echo "🎨 Code Quality:"
	@echo "  fmt            - Format code"
	@echo "  lint           - Linting"
	@echo "  deps           - Install dependencies"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Start Docker container"
	@echo "  docker-compose-up   - Start with Docker Compose"
	@echo "  docker-compose-down - Stop Docker Compose"
	@echo "  docker-clean   - Clean Docker"
	@echo ""
	@echo "📋 Version:"
	@echo "  version        - Show version information"
	@echo "  release        - Create release"
	@echo ""
	@echo "🧹 Clean:"
	@echo "  clean          - Clean build files"
	@echo ""
	@echo "📚 Help:"
	@echo "  help           - Show this help"

# Default Target
.DEFAULT_GOAL := help 