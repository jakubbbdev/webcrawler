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
	// Version anzeigen
	version.PrintVersion()

	// Konfiguration laden
	cfg := config.Load()

	// Logger initialisieren
	logger := logger.New(cfg.LogLevel)
	logger.Info("🚀 WebCrawler API wird gestartet...")

	// Scraper Service initialisieren
	scraperService := scraper.NewService(logger)

	// API Server initialisieren
	server := api.NewServer(cfg, scraperService, logger)

	// Server in Goroutine starten
	go func() {
		logger.Infof("🌐 Server läuft auf http://localhost:%d", cfg.Port)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server Fehler: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Server wird heruntergefahren...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server konnte nicht ordnungsgemäß heruntergefahren werden: %v", err)
	}

	logger.Info("✅ Server erfolgreich heruntergefahren")
}
