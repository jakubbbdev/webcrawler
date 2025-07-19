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
		t.Fatal("Service sollte nicht nil sein")
	}

	if service.client == nil {
		t.Fatal("HTTP Client sollte nicht nil sein")
	}

	if service.logger == nil {
		t.Fatal("Logger sollte nicht nil sein")
	}
}

func TestScrapeWebsite_InvalidURL(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := service.ScrapeWebsite(ctx, "invalid-url")
	if err == nil {
		t.Fatal("Sollte einen Fehler für ungültige URL zurückgeben")
	}
}

func TestScrapeWebsite_Timeout(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	// Sehr kurzer Timeout für Test
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := service.ScrapeWebsite(ctx, "https://httpbin.org/delay/10")
	if err == nil {
		t.Fatal("Sollte einen Timeout-Fehler zurückgeben")
	}
}

func TestScrapeMultipleWebsites_EmptyList(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := service.ScrapeMultipleWebsites(ctx, []string{})
	if err != nil {
		t.Fatalf("Sollte keinen Fehler für leere Liste zurückgeben: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("Sollte leere Ergebnisse zurückgeben, bekam %d", len(results))
	}
}

func TestGetWebsiteStats_InvalidURL(t *testing.T) {
	logger := logger.New("info")
	service := NewService(logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := service.GetWebsiteStats(ctx, "invalid-url")
	if err == nil {
		t.Fatal("Sollte einen Fehler für ungültige URL zurückgeben")
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
		t.Errorf("URL sollte 'https://example.com' sein, bekam '%s'", data.URL)
	}

	if data.Title != "Test Title" {
		t.Errorf("Title sollte 'Test Title' sein, bekam '%s'", data.Title)
	}

	if len(data.Keywords) != 2 {
		t.Errorf("Sollte 2 Keywords haben, bekam %d", len(data.Keywords))
	}

	if data.StatusCode != 200 {
		t.Errorf("Status Code sollte 200 sein, bekam %d", data.StatusCode)
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
			b.Fatalf("Benchmark Fehler: %v", err)
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
			b.Fatalf("Benchmark Fehler: %v", err)
		}
	}
} 