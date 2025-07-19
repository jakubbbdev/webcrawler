package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"web-scraper-api/internal/logger"

	"github.com/PuerkitoBio/goquery"
)

type ScrapedData struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Keywords    []string          `json:"keywords"`
	Images      []string          `json:"images"`
	Links       []string          `json:"links"`
	Text        string            `json:"text"`
	MetaTags    map[string]string `json:"meta_tags"`
	StatusCode  int               `json:"status_code"`
	ScrapedAt   time.Time         `json:"scraped_at"`
}

type Service struct {
	client *http.Client
	logger *logger.Logger
}

func NewService(logger *logger.Logger) *Service {
	return &Service{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (s *Service) ScrapeWebsite(ctx context.Context, url string) (*ScrapedData, error) {
	s.logger.Infof("Scraping website: %s", url)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML parsing failed: %w", err)
	}

	// Extract data
	data := &ScrapedData{
		URL:        url,
		StatusCode: resp.StatusCode,
		ScrapedAt:  time.Now(),
		MetaTags:   make(map[string]string),
	}

	// Extract title
	data.Title = doc.Find("title").Text()

	// Extract meta tags
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		if name != "" && content != "" {
			data.MetaTags[name] = content
		}
		if property != "" && content != "" {
			data.MetaTags[property] = content
		}
	})

	// Description from meta tags
	if desc, ok := data.MetaTags["description"]; ok {
		data.Description = desc
	}

	// Extract keywords
	if keywords, ok := data.MetaTags["keywords"]; ok {
		data.Keywords = strings.Split(keywords, ",")
		for i, keyword := range data.Keywords {
			data.Keywords[i] = strings.TrimSpace(keyword)
		}
	}

	// Extract images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			data.Images = append(data.Images, src)
		}
	})

	// Extract links
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			data.Links = append(data.Links, href)
		}
	})

	// Extract text (without HTML tags)
	data.Text = doc.Text()

	s.logger.Infof("Website successfully scraped: %s (Status: %d)", url, resp.StatusCode)

	return data, nil
}

func (s *Service) ScrapeMultipleWebsites(ctx context.Context, urls []string) ([]*ScrapedData, error) {
	s.logger.Infof("Scraping %d websites", len(urls))

	results := make([]*ScrapedData, 0, len(urls))
	errors := make([]error, 0)

	// Semaphore for concurrency control
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

	for _, url := range urls {
		semaphore <- struct{}{} // Acquire semaphore

		go func(u string) {
			defer func() { <-semaphore }() // Release semaphore

			data, err := s.ScrapeWebsite(ctx, u)
			if err != nil {
				s.logger.Errorf("Error scraping %s: %v", u, err)
				errors = append(errors, err)
				return
			}

			results = append(results, data)
		}(url)
	}

	// Wait for all goroutines to finish
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- struct{}{}
	}

	s.logger.Infof("Scraping completed: %d successful, %d errors", len(results), len(errors))

	return results, nil
}

func (s *Service) GetWebsiteStats(ctx context.Context, url string) (map[string]interface{}, error) {
	data, err := s.ScrapeWebsite(ctx, url)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"url":           data.URL,
		"title_length":  len(data.Title),
		"text_length":   len(data.Text),
		"image_count":   len(data.Images),
		"link_count":    len(data.Links),
		"keyword_count": len(data.Keywords),
		"meta_count":    len(data.MetaTags),
		"status_code":   data.StatusCode,
		"scraped_at":    data.ScrapedAt,
	}

	return stats, nil
}
