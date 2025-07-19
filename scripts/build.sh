#!/bin/bash

# WebCrawler Build Script
# Automatisches Version-Handling und Build-Prozess

set -e

# Farben für Output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Funktionen
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Aktuelle Version lesen
VERSION=$(cat VERSION)
print_info "WebCrawler v$VERSION Build Script"

# Build-Verzeichnis erstellen
BUILD_DIR="build"
mkdir -p $BUILD_DIR

# Go Version prüfen
print_info "Prüfe Go Installation..."
if ! command -v go &> /dev/null; then
    print_error "Go ist nicht installiert!"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
print_success "Go Version: $GO_VERSION"

# Dependencies installieren
print_info "Installiere Dependencies..."
go mod tidy
go mod download
print_success "Dependencies installiert"

# Tests ausführen
print_info "Führe Tests aus..."
if go test ./...; then
    print_success "Alle Tests bestanden"
else
    print_error "Tests fehlgeschlagen!"
    exit 1
fi

# Code formatieren
print_info "Formatiere Code..."
go fmt ./...
print_success "Code formatiert"

# Build für verschiedene Plattformen
print_info "Baue für verschiedene Plattformen..."

# Linux
print_info "Baue für Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-linux-amd64 .
print_success "Linux Build erstellt"

# Windows
print_info "Baue für Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-windows-amd64.exe .
print_success "Windows Build erstellt"

# macOS
print_info "Baue für macOS..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-darwin-amd64 .
print_success "macOS Build erstellt"

# macOS ARM64
print_info "Baue für macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-darwin-arm64 .
print_success "macOS ARM64 Build erstellt"

# Build-Informationen anzeigen
print_info "Build-Informationen:"
echo "Version: $VERSION"
echo "Build Time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo "Go Version: $GO_VERSION"

# Dateigrößen anzeigen
print_info "Build-Dateien:"
ls -lh $BUILD_DIR/

# Checksums erstellen
print_info "Erstelle Checksums..."
cd $BUILD_DIR
sha256sum webcrawler-* > checksums.txt
cd ..
print_success "Checksums erstellt"

print_success "Build abgeschlossen! 🎉"
print_info "Build-Dateien befinden sich im '$BUILD_DIR' Verzeichnis"

# Release-Hinweise
if [[ "$1" == "--release" ]]; then
    print_warning "Release-Modus aktiviert"
    print_info "Nächste Schritte für Release:"
    echo "1. git add ."
    echo "2. git commit -m 'Release v$VERSION'"
    echo "3. git tag v$VERSION"
    echo "4. git push origin main --tags"
    echo "5. Erstelle Release auf GitHub mit den Build-Dateien"
fi 