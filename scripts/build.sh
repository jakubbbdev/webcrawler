#!/bin/bash

# WebCrawler Build Script
# Automatic version handling and build process

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
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

# Read current version
VERSION=$(cat VERSION)
print_info "WebCrawler v$VERSION Build Script"

# Create build directory
BUILD_DIR="build"
mkdir -p $BUILD_DIR

# Check Go installation
print_info "Checking Go installation..."
if ! command -v go &> /dev/null; then
    print_error "Go is not installed!"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
print_success "Go Version: $GO_VERSION"

# Install dependencies
print_info "Installing dependencies..."
go mod tidy
go mod download
print_success "Dependencies installed"

# Run tests
print_info "Running tests..."
if go test ./...; then
    print_success "All tests passed"
else
    print_error "Tests failed!"
    exit 1
fi

# Format code
print_info "Formatting code..."
go fmt ./...
print_success "Code formatted"

# Build for different platforms
print_info "Building for different platforms..."

# Linux
print_info "Building for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-linux-amd64 .
print_success "Linux build created"

# Windows
print_info "Building for Windows..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-windows-amd64.exe .
print_success "Windows build created"

# macOS
print_info "Building for macOS..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-darwin-amd64 .
print_success "macOS build created"

# macOS ARM64
print_info "Building for macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $BUILD_DIR/webcrawler-darwin-arm64 .
print_success "macOS ARM64 build created"

# Show build information
print_info "Build information:"
echo "Version: $VERSION"
echo "Build Time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo "Go Version: $GO_VERSION"

# Show file sizes
print_info "Build files:"
ls -lh $BUILD_DIR/

# Create checksums
print_info "Creating checksums..."
cd $BUILD_DIR
sha256sum webcrawler-* > checksums.txt
cd ..
print_success "Checksums created"

print_success "Build completed! 🎉"
print_info "Build files are in the '$BUILD_DIR' directory"

# Release hints
if [[ "$1" == "--release" ]]; then
    print_warning "Release mode activated"
    print_info "Next steps for release:"
    echo "1. git add ."
    echo "2. git commit -m 'Release v$VERSION'"
    echo "3. git tag v$VERSION"
    echo "4. git push origin main --tags"
    echo "5. Create release on GitHub with build files"
fi 