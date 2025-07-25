package scraper

import (
	"context"
	"testing"
	"time"

	"web-scraper-api/internal/logger"
)

func TestNewService(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	if service == nil {
		t.Fatal("Service should not be nil")
	}

	if service.client == nil {
		t.Fatal("HTTP Client should not be nil")
	}

	if service.logger == nil {
		t.Fatal("Logger should not be nil")
	}
}

func TestScrapeWebsite_InvalidURL(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := service.ScrapeWebsite(ctx, "invalid-url")
	if err == nil {
		t.Fatal("Should return an error for invalid URL")
	}
}

func TestScrapeWebsite_Timeout(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	// Very short timeout for test
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := service.ScrapeWebsite(ctx, "https://httpbin.org/delay/10")
	if err == nil {
		t.Fatal("Should return a timeout error")
	}
}

func TestScrapeMultipleWebsites_EmptyList(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := service.ScrapeMultipleWebsites(ctx, []string{})
	if err != nil {
		t.Fatalf("Should not return an error for empty list: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("Should return empty results, got %d", len(results))
	}
}

func TestGetWebsiteStats_InvalidURL(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := service.GetWebsiteStats(ctx, "invalid-url")
	if err == nil {
		t.Fatal("Should return an error for invalid URL")
	}
}

func TestScrapedData_Structure(t *testing.T) {
	data := &ScrapedData{
		URL:         "https://example.com",
		Title:       "Test Title",
		Description: "Test Description",
		Keywords:    []string{"test", "example"},
		Images:      []string{"https://example.com/image.jpg"},
		Links:       []string{"https://example.com/link"},
		Text:        "Test text content",
		MetaTags:    map[string]string{"description": "Test"},
		StatusCode:  200,
		ScrapedAt:   time.Now(),
	}

	if data.URL != "https://example.com" {
		t.Errorf("URL should be 'https://example.com', got '%s'", data.URL)
	}

	if data.Title != "Test Title" {
		t.Errorf("Title should be 'Test Title', got '%s'", data.Title)
	}

	if len(data.Keywords) != 2 {
		t.Errorf("Should have 2 keywords, got %d", len(data.Keywords))
	}

	if data.StatusCode != 200 {
		t.Errorf("Status Code should be 200, got %d", data.StatusCode)
	}
}

// Benchmark Tests
func BenchmarkScrapeWebsite(b *testing.B) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ScrapeWebsite(ctx, "https://httpbin.org/html")
		if err != nil {
			b.Fatalf("Benchmark error: %v", err)
		}
	}
}

func BenchmarkGetWebsiteStats(b *testing.B) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetWebsiteStats(ctx, "https://httpbin.org/html")
		if err != nil {
			b.Fatalf("Benchmark error: %v", err)
		}
	}
}
