# Makefile für WebCrawler

# Variablen
BINARY_NAME=webcrawler
BUILD_DIR=build
DOCKER_IMAGE=webcrawler
DOCKER_TAG=latest

# Go Befehle
.PHONY: build
build:
	@echo "🔨 Baue Anwendung..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: run
run:
	@echo "🚀 Starte Anwendung..."
	go run main.go

.PHONY: test
test:
	@echo "🧪 Führe Tests aus..."
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "📊 Tests mit Coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage Report: coverage.html"

.PHONY: clean
clean:
	@echo "🧹 Räume auf..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean

.PHONY: fmt
fmt:
	@echo "🎨 Formatiere Code..."
	go fmt ./...

.PHONY: lint
lint:
	@echo "🔍 Linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint nicht installiert. Installiere mit: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

.PHONY: deps
deps:
	@echo "📦 Installiere Dependencies..."
	go mod tidy
	go mod download

.PHONY: install
install: deps build
	@echo "✅ Installation abgeschlossen"

# Docker Befehle
.PHONY: docker-build
docker-build:
	@echo "🐳 Baue Docker Image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run:
	@echo "🐳 Starte Docker Container..."
	docker run -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-compose-up
docker-compose-up:
	@echo "🐳 Starte mit Docker Compose..."
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down:
	@echo "🐳 Stoppe Docker Compose..."
	docker-compose down

.PHONY: docker-clean
docker-clean:
	@echo "🧹 Räume Docker auf..."
	docker system prune -f
	docker image prune -f

# Development Befehle
.PHONY: dev
dev:
	@echo "🛠️  Development Mode..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air nicht installiert. Installiere mit: go install github.com/cosmtrek/air@latest"; \
		echo "Oder verwende: make run"; \
		make run; \
	fi

.PHONY: debug
debug:
	@echo "🐛 Debug Mode..."
	LOG_LEVEL=debug go run main.go

# Version Befehle
.PHONY: version
version:
	@echo "📋 Versionsinformationen:"
	@go run main.go --version 2>/dev/null || echo "Version: $(shell cat VERSION)"

.PHONY: release
release:
	@echo "🏷️  Erstelle Release..."
	@echo "Aktuelle Version: $(shell cat VERSION)"
	@echo "Führe folgende Schritte aus:"
	@echo "1. git tag v$(shell cat VERSION)"
	@echo "2. git push origin v$(shell cat VERSION)"
	@echo "3. Erstelle Release auf GitHub"

# Utility Befehle
.PHONY: help
help:
	@echo "📚 Verfügbare Befehle:"
	@echo ""
	@echo "🔨 Build:"
	@echo "  build          - Baue Anwendung"
	@echo "  install        - Installiere Dependencies und baue"
	@echo ""
	@echo "🚀 Run:"
	@echo "  run            - Starte Anwendung"
	@echo "  dev            - Development Mode mit Hot Reload"
	@echo "  debug          - Debug Mode"
	@echo ""
	@echo "🧪 Test:"
	@echo "  test           - Führe Tests aus"
	@echo "  test-coverage  - Tests mit Coverage Report"
	@echo ""
	@echo "🎨 Code Quality:"
	@echo "  fmt            - Formatiere Code"
	@echo "  lint           - Linting"
	@echo "  deps           - Installiere Dependencies"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  docker-build   - Baue Docker Image"
	@echo "  docker-run     - Starte Docker Container"
	@echo "  docker-compose-up   - Starte mit Docker Compose"
	@echo "  docker-compose-down - Stoppe Docker Compose"
	@echo "  docker-clean   - Räume Docker auf"
	@echo ""
	@echo "📋 Version:"
	@echo "  version        - Zeige Versionsinformationen"
	@echo "  release        - Erstelle Release"
	@echo ""
	@echo "🧹 Clean:"
	@echo "  clean          - Räume Build-Dateien auf"
	@echo ""
	@echo "📚 Help:"
	@echo "  help           - Zeige diese Hilfe"

# Default Target
.DEFAULT_GOAL := help 