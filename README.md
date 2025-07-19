# 🕷️ WebCrawler

Eine moderne, in Go geschriebene REST API zum Crawlen und Scrapen von Websites mit einer schönen Web-Oberfläche.

## ✨ Features

- **REST API**: Vollständige REST API für Website-Crawling
- **Batch Processing**: Mehrere Websites gleichzeitig crawlen
- **Website Statistiken**: Detaillierte Statistiken über Websites
- **Moderne UI**: Schöne Web-Oberfläche mit Tabs
- **Concurrency**: Parallele Verarbeitung mit Goroutines
- **Graceful Shutdown**: Sauberes Herunterfahren
- **Strukturiertes Logging**: JSON-basiertes Logging
- **Konfiguration**: Umgebungsvariablen und YAML-Konfiguration
- **CORS Support**: Cross-Origin Resource Sharing
- **Version Management**: Professionelles Version-Handling

## 🚀 Installation

### Voraussetzungen

- Go 1.21 oder höher
- Git

### Installation

```bash
# Repository klonen
git clone https://github.com/dein-username/webcrawler.git
cd webcrawler

# Dependencies installieren
go mod tidy

# Anwendung starten
go run main.go
```

### Mit Docker

```bash
# Docker Image bauen
docker build -t webcrawler .

# Container starten
docker run -p 8080:8080 webcrawler
```

## 📖 Verwendung

### Web Interface

Öffne deinen Browser und gehe zu `http://localhost:8080`

### API Endpunkte

#### 1. Einzelne Website crawlen
```bash
curl -X POST http://localhost:8080/api/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

#### 2. Mehrere Websites crawlen
```bash
curl -X POST http://localhost:8080/api/v1/scrape/batch \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example1.com", "https://example2.com"]}'
```

#### 3. Website Statistiken
```bash
curl http://localhost:8080/api/v1/scrape/stats/https%3A//example.com
```

#### 4. Health Check
```bash
curl http://localhost:8080/health
```

## ⚙️ Konfiguration

### Umgebungsvariablen

```bash
export PORT=8080
export LOG_LEVEL=info
export TIMEOUT=30
```

### Konfigurationsdatei (config.yaml)

```yaml
port: 8080
log_level: info
timeout: 30
```

## 📊 API Response Format

### Crawled Data
```json
{
  "success": true,
  "data": {
    "url": "https://example.com",
    "title": "Example Domain",
    "description": "This domain is for use in illustrative examples...",
    "keywords": ["example", "domain"],
    "images": ["https://example.com/image.jpg"],
    "links": ["https://example.com/page1"],
    "text": "Extracted text content...",
    "meta_tags": {
      "description": "Example description",
      "keywords": "example, domain"
    },
    "status_code": 200,
    "scraped_at": "2024-01-01T12:00:00Z"
  }
}
```

### Website Statistics
```json
{
  "success": true,
  "data": {
    "url": "https://example.com",
    "title_length": 15,
    "text_length": 1024,
    "image_count": 5,
    "link_count": 10,
    "keyword_count": 3,
    "meta_count": 8,
    "status_code": 200,
    "scraped_at": "2024-01-01T12:00:00Z"
  }
}
```

## 🏗️ Projektstruktur

```
webcrawler/
├── main.go                 # Hauptanwendung
├── go.mod                  # Go Module
├── go.sum                  # Dependencies Checksum
├── VERSION                 # Versionsdatei
├── README.md              # Dokumentation
├── Dockerfile             # Docker Konfiguration
├── .gitignore             # Git Ignore
├── config/
│   └── config.yaml        # Konfigurationsdatei
├── templates/
│   └── index.html         # Web Interface
├── internal/
│   ├── api/
│   │   └── server.go      # HTTP Server & Routes
│   ├── config/
│   │   └── config.go      # Konfigurationsmanagement
│   ├── logger/
│   │   └── logger.go      # Strukturiertes Logging
│   ├── scraper/
│   │   └── scraper.go     # Web Crawling Logic
│   └── version/
│       └── version.go     # Version Management
└── tests/
    └── scraper_test.go    # Unit Tests
```

## 🧪 Tests

```bash
# Alle Tests ausführen
go test ./...

# Tests mit Coverage
go test -cover ./...

# Spezifische Tests
go test ./internal/scraper
```

## 🐳 Docker

### Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
EXPOSE 8080
CMD ["./main"]
```

### Docker Compose
```yaml
version: '3.8'
services:
  webcrawler:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - LOG_LEVEL=info
    volumes:
      - ./config:/app/config
```

## 🔧 Entwicklung

### Lokale Entwicklung
```bash
# Dependencies installieren
go mod tidy

# Anwendung im Debug-Modus starten
LOG_LEVEL=debug go run main.go

# Tests ausführen
go test ./...
```

### Code Formatierung
```bash
# Code formatieren
go fmt ./...

# Linting
golangci-lint run
```

## 📝 Lizenz

Dieses Projekt ist unter der MIT-Lizenz lizenziert. Siehe [LICENSE](LICENSE) für Details.

## 🤝 Beitragen

1. Fork das Repository
2. Erstelle einen Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Committe deine Änderungen (`git commit -m 'Add some AmazingFeature'`)
4. Push zum Branch (`git push origin feature/AmazingFeature`)
5. Öffne einen Pull Request

## 📞 Support

Bei Fragen oder Problemen erstelle bitte ein Issue auf GitHub.

## 🚀 Roadmap

- [ ] WebSocket Support für Live Updates
- [ ] Rate Limiting
- [ ] Authentication & Authorization
- [ ] Database Integration
- [ ] Caching Layer
- [ ] More Crawling Options
- [ ] Export to CSV/JSON
- [ ] Scheduled Crawling
- [ ] Sitemap Generation
- [ ] SEO Analysis

---

**Entwickelt mit ❤️ in Go** 