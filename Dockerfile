# Multi-stage build für optimierte Image-Größe
FROM golang:1.21-alpine AS builder

# Installiere notwendige Build-Tools
RUN apk add --no-cache git ca-certificates tzdata

# Setze Arbeitsverzeichnis
WORKDIR /app

# Kopiere go mod Dateien
COPY go.mod go.sum ./

# Lade Dependencies
RUN go mod download

# Kopiere Source Code
COPY . .

# Baue die Anwendung
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o webcrawler .

# Zweite Stage: Runtime Image
FROM alpine:latest

# Installiere CA-Zertifikate und Timezone-Daten
RUN apk --no-cache add ca-certificates tzdata

# Erstelle nicht-root User
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Setze Arbeitsverzeichnis
WORKDIR /app

# Kopiere Binary von Builder Stage
COPY --from=builder /app/webcrawler .

# Kopiere Templates
COPY --from=builder /app/templates ./templates

# Ändere Besitzer zu nicht-root User
RUN chown -R appuser:appgroup /app

# Wechsle zu nicht-root User
USER appuser

# Exponiere Port
EXPOSE 8080

# Health Check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Starte Anwendung
CMD ["./webcrawler"] 