package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"web-scraper-api/internal/api"
	"web-scraper-api/internal/config"
	"web-scraper-api/internal/logger"
	"web-scraper-api/internal/scraper"
	"web-scraper-api/internal/version"
)

func main() {
	// Display version information
	version.PrintVersion()

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := logger.New(cfg.LogLevel)
	logger.Info("🚀 WebCrawler API starting...")

	// Initialize scraper service
	scraperService := scraper.NewService(logger)

	// Initialize API server
	server := api.NewServer(cfg, scraperService, logger)

	// Start server in goroutine
	go func() {
		logger.Infof("🌐 Server running on http://localhost:%d", cfg.Port)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server could not be shut down gracefully: %v", err)
	}

	logger.Info("✅ Server successfully shut down")
}
