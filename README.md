# 🕷️ WebCrawler

A modern REST API written in Go for crawling and scraping websites with a beautiful web interface.

## ✨ Features

- **REST API**: Complete REST API for website crawling
- **Batch Processing**: Crawl multiple websites simultaneously
- **Website Statistics**: Detailed statistics about websites
- **Modern UI**: Beautiful web interface with tabs
- **Concurrency**: Parallel processing with Goroutines
- **Graceful Shutdown**: Clean shutdown process
- **Structured Logging**: JSON-based logging
- **Configuration**: Environment variables and YAML configuration
- **CORS Support**: Cross-Origin Resource Sharing
- **Version Management**: Professional version handling
- **WebSocket Support**: Real-time live updates during crawling
- **Export Functionality**: Export data to CSV and JSON formats

## 🚀 Installation

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

```bash
# Clone repository
git clone https://github.com/your-username/webcrawler.git
cd webcrawler

# Install dependencies
go mod tidy

# Start application
go run main.go
```

### With Docker

```bash
# Build Docker image
docker build -t webcrawler .

# Start container
docker run -p 8080:8080 webcrawler
```

## 📖 Usage

### Web Interface

Open your browser and go to `http://localhost:8080`

The web interface includes:
- **Single URL Crawling**: Crawl individual websites
- **Batch Crawling**: Crawl multiple websites simultaneously
- **Live Updates**: Real-time progress via WebSocket
- **Export Options**: Download data as CSV or JSON
- **Website Statistics**: View detailed analytics

### API Endpoints

#### 1. Crawl single website
```bash
curl -X POST http://localhost:8080/api/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

#### 2. Crawl multiple websites
```bash
curl -X POST http://localhost:8080/api/v1/scrape/batch \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example1.com", "https://example2.com"]}'
```

#### 3. Website Statistics
```bash
curl http://localhost:8080/api/v1/scrape/stats?url=https://example.com
```

#### 4. Export to CSV
```bash
curl http://localhost:8080/api/v1/export/csv?url=https://example.com
```

#### 5. Export to JSON
```bash
curl http://localhost:8080/api/v1/export/json?url=https://example.com
```

#### 6. WebSocket Connection
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');
ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Live update:', message);
};
```

#### 7. Health Check
```bash
curl http://localhost:8080/health
```

## ⚙️ Configuration

### Environment Variables

```bash
export PORT=8080
export LOG_LEVEL=info
export TIMEOUT=30
```

### Configuration File (config.yaml)

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

### WebSocket Messages
```json
{
  "type": "scraping_update",
  "data": {
    "url": "https://example.com",
    "status": "completed",
    "data": { /* scraped data */ }
  },
  "time": "2024-01-01T12:00:00Z"
}
```

## 🏗️ Project Structure

```
webcrawler/
├── main.go                 # Main application
├── go.mod                  # Go Module
├── go.sum                  # Dependencies Checksum
├── VERSION                 # Version file
├── README.md              # Documentation
├── Dockerfile             # Docker configuration
├── .gitignore             # Git ignore
├── config/
│   └── config.yaml        # Configuration file
├── templates/
│   └── index.html         # Web interface
├── internal/
│   ├── api/
│   │   ├── server.go      # HTTP Server & Routes
│   │   └── websocket.go   # WebSocket management
│   ├── config/
│   │   └── config.go      # Configuration management
│   ├── logger/
│   │   └── logger.go      # Structured logging
│   ├── scraper/
│   │   └── scraper.go     # Web crawling logic
│   └── version/
│       └── version.go     # Version management
└── tests/
    └── scraper_test.go    # Unit tests
```

## 🧪 Tests

```bash
# Run all tests
go test ./...

# Tests with coverage
go test -cover ./...

# Specific tests
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

## 🔧 Development

### Local Development
```bash
# Install dependencies
go mod tidy

# Start application in debug mode
LOG_LEVEL=debug go run main.go

# Run tests
go test ./...
```

### Code Formatting
```bash
# Format code
go fmt ./...

# Linting
golangci-lint run
```

## 📝 License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📞 Support

For questions or issues, please create an issue on GitHub.

## 🚀 Roadmap

- [x] WebSocket Support for Live Updates
- [x] Export to CSV/JSON
- [ ] Rate Limiting
- [ ] Authentication & Authorization
- [ ] Database Integration
- [ ] Caching Layer
- [X] More Crawling Options
- [ ] Scheduled Crawling
- [ ] Sitemap Generation
- [ ] SEO Analysis

---

**Developed with ❤️ in Go** 