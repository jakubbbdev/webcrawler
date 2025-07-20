package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"web-scraper-api/internal/config"
	"web-scraper-api/internal/logger"
	"web-scraper-api/internal/scraper"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router         *gin.Engine
	server         *http.Server
	config         *config.Config
	scraperService *scraper.Service
	logger         *logger.Logger
	wsManager      *WebSocketManager
}

func NewServer(cfg *config.Config, scraperService *scraper.Service, logger *logger.Logger) *Server {
	// Set Gin mode
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Load HTML templates
	router.LoadHTMLGlob("templates/*")

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Initialize WebSocket manager
	wsManager := NewWebSocketManager(logger)

	server := &Server{
		router:         router,
		config:         cfg,
		scraperService: scraperService,
		logger:         logger,
		wsManager:      wsManager,
	}

	server.setupRoutes()

	// Start WebSocket manager
	go wsManager.Start()

	return server
}

func (s *Server) setupRoutes() {
	// Health Check
	s.router.GET("/health", s.healthCheck)

	// API Routes
	api := s.router.Group("/api/v1")
	{
		// Scraping Routes
		api.POST("/scrape", s.scrapeWebsite)
		api.POST("/scrape/batch", s.scrapeMultipleWebsites)
		api.GET("/scrape/stats", s.getWebsiteStats)

		// Export Routes
		api.GET("/export/csv", s.exportToCSV)
		api.GET("/export/json", s.exportToJSON)

		// WebSocket for live updates
		api.GET("/ws", s.handleWebSocket)
	}

	// Frontend (simple HTML page)
	s.router.GET("/", s.serveFrontend)
	s.router.Static("/static", "./static")
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "webcrawler-api",
		"version":   "1.0.0",
	})
}

func (s *Server) scrapeWebsite(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	// Broadcast scraping start
	s.wsManager.BroadcastScrapingUpdate(request.URL, "started", nil)

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	data, err := s.scraperService.ScrapeWebsite(ctx, request.URL)
	if err != nil {
		s.logger.Errorf("Scraping error: %v", err)
		s.wsManager.BroadcastError(request.URL, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Broadcast scraping completion
	s.wsManager.BroadcastScrapingUpdate(request.URL, "completed", data)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func (s *Server) scrapeMultipleWebsites(c *gin.Context) {
	var request struct {
		URLs []string `json:"urls" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URLs array is required",
		})
		return
	}

	if len(request.URLs) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 10 URLs allowed",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	results := make([]*scraper.ScrapedData, 0, len(request.URLs))
	completed := 0

	// Semaphore for concurrency control
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests
	resultsChan := make(chan *scraper.ScrapedData, len(request.URLs))
	errorsChan := make(chan error, len(request.URLs))

	// Broadcast batch start
	s.wsManager.BroadcastBatchProgress(len(request.URLs), 0, "Starting batch scraping...")

	for _, url := range request.URLs {
		semaphore <- struct{}{} // Acquire semaphore

		go func(u string) {
			defer func() { <-semaphore }() // Release semaphore

			// Broadcast individual scraping start
			s.wsManager.BroadcastScrapingUpdate(u, "started", nil)

			data, err := s.scraperService.ScrapeWebsite(ctx, u)
			if err != nil {
				s.logger.Errorf("Error scraping %s: %v", u, err)
				s.wsManager.BroadcastError(u, err.Error())
				errorsChan <- err
				return
			}

			// Broadcast individual scraping completion
			s.wsManager.BroadcastScrapingUpdate(u, "completed", data)
			resultsChan <- data
		}(url)
	}

	// Collect results
	for i := 0; i < len(request.URLs); i++ {
		select {
		case data := <-resultsChan:
			results = append(results, data)
			completed++
			s.wsManager.BroadcastBatchProgress(len(request.URLs), completed, data.URL)
		case err := <-errorsChan:
			s.logger.Errorf("Batch scraping error: %v", err)
		}
	}

	s.logger.Infof("Scraping completed: %d successful, %d errors", len(results), len(request.URLs)-len(results))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"count":   len(results),
	})
}

func (s *Server) getWebsiteStats(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	stats, err := s.scraperService.GetWebsiteStats(ctx, url)
	if err != nil {
		s.logger.Errorf("Stats error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

func (s *Server) exportToCSV(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	data, err := s.scraperService.ScrapeWebsite(ctx, url)
	if err != nil {
		s.logger.Errorf("Export error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	csvData := s.convertToCSV(data)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=scraped_data_%s.csv", time.Now().Format("20060102_150405")))
	c.Data(http.StatusOK, "text/csv", []byte(csvData))
}

func (s *Server) exportToJSON(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL parameter is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	data, err := s.scraperService.ScrapeWebsite(ctx, url)
	if err != nil {
		s.logger.Errorf("Export error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=scraped_data_%s.json", time.Now().Format("20060102_150405")))
	c.JSON(http.StatusOK, data)
}

func (s *Server) convertToCSV(data *scraper.ScrapedData) string {
	// Simple CSV conversion
	csv := "URL,Title,Description,Keywords,Images,Links,Text Length,Status Code,Scraped At\n"
	csv += fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",%d,%d,\"%s\"\n",
		data.URL,
		data.Title,
		data.Description,
		fmt.Sprintf("%v", data.Keywords),
		fmt.Sprintf("%v", data.Images),
		fmt.Sprintf("%v", data.Links),
		len(data.Text),
		data.StatusCode,
		data.ScrapedAt.Format(time.RFC3339),
	)
	return csv
}

func (s *Server) handleWebSocket(c *gin.Context) {
	s.wsManager.HandleWebSocket(c.Writer, c.Request)
}

func (s *Server) serveFrontend(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "WebCrawler API",
	})
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: s.router,
	}

	s.logger.Infof("Server starting on port %d", s.config.Port)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
