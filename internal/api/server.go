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

	server := &Server{
		router:         router,
		config:         cfg,
		scraperService: scraperService,
		logger:         logger,
	}

	server.setupRoutes()

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
		api.GET("/scrape/stats/:url", s.getWebsiteStats)

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

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	data, err := s.scraperService.ScrapeWebsite(ctx, request.URL)
	if err != nil {
		s.logger.Errorf("Scraping error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

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

	results, err := s.scraperService.ScrapeMultipleWebsites(ctx, request.URLs)
	if err != nil {
		s.logger.Errorf("Batch scraping error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"count":   len(results),
	})
}

func (s *Server) getWebsiteStats(c *gin.Context) {
	url := c.Param("url")
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

func (s *Server) handleWebSocket(c *gin.Context) {
	// WebSocket handler for live updates
	c.JSON(http.StatusOK, gin.H{
		"message": "WebSocket Endpoint - Coming Soon!",
	})
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
