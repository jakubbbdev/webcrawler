package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"web-scraper-api/internal/config"
	"web-scraper-api/internal/logger"
	"web-scraper-api/internal/scheduler"
	"web-scraper-api/internal/scraper"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router         *gin.Engine
	server         *http.Server
	config         *config.Config
	scraperService *scraper.Service
	scheduler      *scheduler.Scheduler
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

	// Initialize Scheduler
	scheduler := scheduler.NewScheduler(logger, scraperService)

	server := &Server{
		router:         router,
		config:         cfg,
		scraperService: scraperService,
		scheduler:      scheduler,
		logger:         logger,
		wsManager:      wsManager,
	}

	// Set up scheduler callbacks
	scheduler.SetCallbacks(
		server.onScheduledJobStart,
		server.onScheduledJobComplete,
		server.onScheduledJobError,
	)

	server.setupRoutes()

	// Start WebSocket manager and Scheduler
	go wsManager.Start()
	go scheduler.Start()

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
		api.POST("/scrape/advanced", s.scrapeWebsiteAdvanced)
		api.POST("/scrape/batch", s.scrapeMultipleWebsites)
		api.POST("/scrape/batch/advanced", s.scrapeMultipleWebsitesAdvanced)
		api.GET("/scrape/stats", s.getWebsiteStats)
		api.POST("/scrape/stats/advanced", s.getWebsiteStatsAdvanced)

		// Export Routes
		api.GET("/export/csv", s.exportToCSV)
		api.GET("/export/json", s.exportToJSON)
		api.POST("/export/csv/advanced", s.exportToCSVAdvanced)
		api.POST("/export/json/advanced", s.exportToJSONAdvanced)

		// Scheduled Jobs Routes
		api.GET("/scheduler/jobs", s.getScheduledJobs)
		api.POST("/scheduler/jobs", s.createScheduledJob)
		api.GET("/scheduler/jobs/:id", s.getScheduledJob)
		api.PUT("/scheduler/jobs/:id", s.updateScheduledJob)
		api.DELETE("/scheduler/jobs/:id", s.deleteScheduledJob)
		api.POST("/scheduler/jobs/:id/pause", s.pauseScheduledJob)
		api.POST("/scheduler/jobs/:id/resume", s.resumeScheduledJob)
		api.POST("/scheduler/jobs/:id/run", s.runScheduledJobNow)
		api.GET("/scheduler/stats", s.getSchedulerStats)
		api.GET("/scheduler/export", s.exportScheduledJobs)
		api.POST("/scheduler/import", s.importScheduledJobs)

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

func (s *Server) scrapeWebsiteAdvanced(c *gin.Context) {
	var request struct {
		URL     string                   `json:"url" binding:"required"`
		Options *scraper.CrawlingOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	// Use default options if none provided
	if request.Options == nil {
		request.Options = &scraper.CrawlingOptions{
			MaxDepth:         1,
			MaxPages:         1,
			Timeout:          30 * time.Second,
			Delay:            0,
			ExtractImages:    true,
			ExtractLinks:     true,
			ExtractForms:     false,
			ExtractTables:    false,
			ExtractScripts:   false,
			ExtractStyles:    false,
			ExtractHeaders:   false,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			FollowRedirects:  true,
			RespectRobotsTxt: false,
		}
	}

	// Broadcast scraping start
	s.wsManager.BroadcastScrapingUpdate(request.URL, "started", nil)

	ctx, cancel := context.WithTimeout(c.Request.Context(), request.Options.Timeout)
	defer cancel()

	data, err := s.scraperService.ScrapeWebsiteWithOptions(ctx, request.URL, request.Options)
	if err != nil {
		s.logger.Errorf("Advanced scraping error: %v", err)
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
		"options": request.Options,
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

func (s *Server) scrapeMultipleWebsitesAdvanced(c *gin.Context) {
	var request struct {
		URLs    []string                 `json:"urls" binding:"required"`
		Options *scraper.CrawlingOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URLs array is required",
		})
		return
	}

	if len(request.URLs) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum 50 URLs allowed for advanced crawling",
		})
		return
	}

	// Use default options if none provided
	if request.Options == nil {
		request.Options = &scraper.CrawlingOptions{
			MaxDepth:         1,
			MaxPages:         len(request.URLs),
			Timeout:          30 * time.Second,
			Delay:            0,
			ExtractImages:    true,
			ExtractLinks:     true,
			ExtractForms:     false,
			ExtractTables:    false,
			ExtractScripts:   false,
			ExtractStyles:    false,
			ExtractHeaders:   false,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			FollowRedirects:  true,
			RespectRobotsTxt: false,
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), request.Options.Timeout)
	defer cancel()

	results := make([]*scraper.ScrapedData, 0, len(request.URLs))
	completed := 0

	// Semaphore for concurrency control
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests
	resultsChan := make(chan *scraper.ScrapedData, len(request.URLs))
	errorsChan := make(chan error, len(request.URLs))

	// Broadcast batch start
	s.wsManager.BroadcastBatchProgress(len(request.URLs), 0, "Starting advanced batch scraping...")

	for _, url := range request.URLs {
		semaphore <- struct{}{} // Acquire semaphore

		go func(u string) {
			defer func() { <-semaphore }() // Release semaphore

			// Add delay if specified
			if request.Options.Delay > 0 {
				time.Sleep(request.Options.Delay)
			}

			// Broadcast individual scraping start
			s.wsManager.BroadcastScrapingUpdate(u, "started", nil)

			data, err := s.scraperService.ScrapeWebsiteWithOptions(ctx, u, request.Options)
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
			s.logger.Errorf("Advanced batch scraping error: %v", err)
		}
	}

	s.logger.Infof("Advanced scraping completed: %d successful, %d errors", len(results), len(request.URLs)-len(results))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"count":   len(results),
		"options": request.Options,
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

func (s *Server) getWebsiteStatsAdvanced(c *gin.Context) {
	var request struct {
		URL     string                   `json:"url" binding:"required"`
		Options *scraper.CrawlingOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	// Use default options if none provided
	if request.Options == nil {
		request.Options = &scraper.CrawlingOptions{
			MaxDepth:         1,
			MaxPages:         1,
			Timeout:          30 * time.Second,
			Delay:            0,
			ExtractImages:    true,
			ExtractLinks:     true,
			ExtractForms:     true,
			ExtractTables:    true,
			ExtractScripts:   true,
			ExtractStyles:    true,
			ExtractHeaders:   true,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			FollowRedirects:  true,
			RespectRobotsTxt: false,
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), request.Options.Timeout)
	defer cancel()

	stats, err := s.scraperService.GetWebsiteStatsWithOptions(ctx, request.URL, request.Options)
	if err != nil {
		s.logger.Errorf("Advanced stats error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
		"options": request.Options,
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

func (s *Server) exportToCSVAdvanced(c *gin.Context) {
	var request struct {
		URL     string                   `json:"url" binding:"required"`
		Options *scraper.CrawlingOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	// Use default options if none provided
	if request.Options == nil {
		request.Options = &scraper.CrawlingOptions{
			MaxDepth:         1,
			MaxPages:         1,
			Timeout:          30 * time.Second,
			Delay:            0,
			ExtractImages:    true,
			ExtractLinks:     true,
			ExtractForms:     true,
			ExtractTables:    true,
			ExtractScripts:   true,
			ExtractStyles:    true,
			ExtractHeaders:   true,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			FollowRedirects:  true,
			RespectRobotsTxt: false,
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), request.Options.Timeout)
	defer cancel()

	data, err := s.scraperService.ScrapeWebsiteWithOptions(ctx, request.URL, request.Options)
	if err != nil {
		s.logger.Errorf("Advanced export error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	csvData := s.convertToCSVAdvanced(data)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=advanced_scraped_data_%s.csv", time.Now().Format("20060102_150405")))
	c.Data(http.StatusOK, "text/csv", []byte(csvData))
}

func (s *Server) exportToJSONAdvanced(c *gin.Context) {
	var request struct {
		URL     string                   `json:"url" binding:"required"`
		Options *scraper.CrawlingOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "URL is required",
		})
		return
	}

	// Use default options if none provided
	if request.Options == nil {
		request.Options = &scraper.CrawlingOptions{
			MaxDepth:         1,
			MaxPages:         1,
			Timeout:          30 * time.Second,
			Delay:            0,
			ExtractImages:    true,
			ExtractLinks:     true,
			ExtractForms:     true,
			ExtractTables:    true,
			ExtractScripts:   true,
			ExtractStyles:    true,
			ExtractHeaders:   true,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			FollowRedirects:  true,
			RespectRobotsTxt: false,
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), request.Options.Timeout)
	defer cancel()

	data, err := s.scraperService.ScrapeWebsiteWithOptions(ctx, request.URL, request.Options)
	if err != nil {
		s.logger.Errorf("Advanced export error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=advanced_scraped_data_%s.json", time.Now().Format("20060102_150405")))
	c.JSON(http.StatusOK, gin.H{
		"data":    data,
		"options": request.Options,
	})
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

func (s *Server) convertToCSVAdvanced(data *scraper.ScrapedData) string {
	// Enhanced CSV conversion with all new fields
	csv := "URL,Title,Description,Keywords,Images,Links,Forms,Tables,Scripts,Styles,H1Tags,H2Tags,H3Tags,Text Length,Status Code,Scraped At\n"

	// Convert arrays to strings
	keywords := fmt.Sprintf("%v", data.Keywords)
	images := fmt.Sprintf("%v", data.Images)
	links := fmt.Sprintf("%v", data.Links)
	forms := fmt.Sprintf("%v", data.Forms)
	tables := fmt.Sprintf("%v", data.Tables)
	scripts := fmt.Sprintf("%v", data.Scripts)
	styles := fmt.Sprintf("%v", data.Styles)
	h1Tags := fmt.Sprintf("%v", data.H1Tags)
	h2Tags := fmt.Sprintf("%v", data.H2Tags)
	h3Tags := fmt.Sprintf("%v", data.H3Tags)

	csv += fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",\"%s\",%d,%d,\"%s\"\n",
		data.URL,
		data.Title,
		data.Description,
		keywords,
		images,
		links,
		forms,
		tables,
		scripts,
		styles,
		h1Tags,
		h2Tags,
		h3Tags,
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

// Scheduler callback methods
func (s *Server) onScheduledJobStart(jobResult *scheduler.JobResult) {
	s.wsManager.BroadcastScheduledJobStart(jobResult)
}

func (s *Server) onScheduledJobComplete(jobResult *scheduler.JobResult) {
	s.wsManager.BroadcastScheduledJobComplete(jobResult)
}

func (s *Server) onScheduledJobError(jobResult *scheduler.JobResult) {
	s.wsManager.BroadcastScheduledJobError(jobResult)
}

// Scheduled Jobs API endpoints
func (s *Server) getScheduledJobs(c *gin.Context) {
	jobs := s.scheduler.GetAllJobs()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    jobs,
		"count":   len(jobs),
	})
}

func (s *Server) createScheduledJob(c *gin.Context) {
	var request struct {
		Name        string                   `json:"name" binding:"required"`
		Description string                   `json:"description"`
		Schedule    string                   `json:"schedule" binding:"required"`
		URL         string                   `json:"url" binding:"required"`
		Options     *scraper.CrawlingOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name, schedule, and URL are required",
		})
		return
	}

	// Use default options if none provided
	if request.Options == nil {
		request.Options = &scraper.CrawlingOptions{
			MaxDepth:         1,
			MaxPages:         1,
			Timeout:          30 * time.Second,
			Delay:            0,
			ExtractImages:    true,
			ExtractLinks:     true,
			ExtractForms:     false,
			ExtractTables:    false,
			ExtractScripts:   false,
			ExtractStyles:    false,
			ExtractHeaders:   false,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			FollowRedirects:  true,
			RespectRobotsTxt: false,
		}
	}

	job := &scheduler.ScheduledJob{
		Name:        request.Name,
		Description: request.Description,
		Schedule:    request.Schedule,
		URL:         request.URL,
		Options:     request.Options,
	}

	if err := s.scheduler.AddJob(job); err != nil {
		s.logger.Errorf("Failed to create scheduled job: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Broadcast job list update
	s.wsManager.BroadcastScheduledJobList(s.scheduler.GetAllJobs())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}

func (s *Server) getScheduledJob(c *gin.Context) {
	jobID := c.Param("id")
	job, err := s.scheduler.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}

func (s *Server) updateScheduledJob(c *gin.Context) {
	jobID := c.Param("id")
	job, err := s.scheduler.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	var request struct {
		Name        string                   `json:"name"`
		Description string                   `json:"description"`
		Schedule    string                   `json:"schedule"`
		URL         string                   `json:"url"`
		Options     *scraper.CrawlingOptions `json:"options"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	// Update fields if provided
	if request.Name != "" {
		job.Name = request.Name
	}
	if request.Description != "" {
		job.Description = request.Description
	}
	if request.Schedule != "" {
		job.Schedule = request.Schedule
	}
	if request.URL != "" {
		job.URL = request.URL
	}
	if request.Options != nil {
		job.Options = request.Options
	}

	job.UpdatedAt = time.Now()

	// Remove and re-add job to update schedule
	if err := s.scheduler.RemoveJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update job",
		})
		return
	}

	if err := s.scheduler.AddJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update job schedule",
		})
		return
	}

	// Broadcast job list update
	s.wsManager.BroadcastScheduledJobList(s.scheduler.GetAllJobs())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    job,
	})
}

func (s *Server) deleteScheduledJob(c *gin.Context) {
	jobID := c.Param("id")
	if err := s.scheduler.RemoveJob(jobID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Job not found",
		})
		return
	}

	// Broadcast job list update
	s.wsManager.BroadcastScheduledJobList(s.scheduler.GetAllJobs())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job deleted successfully",
	})
}

func (s *Server) pauseScheduledJob(c *gin.Context) {
	jobID := c.Param("id")
	if err := s.scheduler.PauseJob(jobID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Broadcast job list update
	s.wsManager.BroadcastScheduledJobList(s.scheduler.GetAllJobs())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job paused successfully",
	})
}

func (s *Server) resumeScheduledJob(c *gin.Context) {
	jobID := c.Param("id")
	if err := s.scheduler.ResumeJob(jobID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Broadcast job list update
	s.wsManager.BroadcastScheduledJobList(s.scheduler.GetAllJobs())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job resumed successfully",
	})
}

func (s *Server) runScheduledJobNow(c *gin.Context) {
	jobID := c.Param("id")
	if err := s.scheduler.RunJobNow(jobID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Job started successfully",
	})
}

func (s *Server) getSchedulerStats(c *gin.Context) {
	stats := s.scheduler.GetJobStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

func (s *Server) exportScheduledJobs(c *gin.Context) {
	data, err := s.scheduler.ExportJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to export jobs",
		})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=scheduled_jobs_%s.json", time.Now().Format("20060102_150405")))
	c.Data(http.StatusOK, "application/json", data)
}

func (s *Server) importScheduledJobs(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No file provided",
		})
		return
	}

	// Read file content
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read file",
		})
		return
	}
	defer f.Close()

	// Read all data
	data := make([]byte, file.Size)
	_, err = f.Read(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read file content",
		})
		return
	}

	// Import jobs
	if err := s.scheduler.ImportJobs(data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to import jobs: " + err.Error(),
		})
		return
	}

	// Broadcast job list update
	s.wsManager.BroadcastScheduledJobList(s.scheduler.GetAllJobs())

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Jobs imported successfully",
	})
}
