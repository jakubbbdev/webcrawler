package scraper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"web-scraper-api/internal/logger"
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
	s.logger.Infof("Scraping Website: %s", url)

	// HTTP Request erstellen
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Request erstellen fehlgeschlagen: %w", err)
	}

	// User-Agent setzen
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Request ausführen
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP Request fehlgeschlagen: %w", err)
	}
	defer resp.Body.Close()

	// HTML parsen
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML Parsing fehlgeschlagen: %w", err)
	}

	// Daten extrahieren
	data := &ScrapedData{
		URL:        url,
		StatusCode: resp.StatusCode,
		ScrapedAt:  time.Now(),
		MetaTags:   make(map[string]string),
	}

	// Title extrahieren
	data.Title = doc.Find("title").Text()

	// Meta Tags extrahieren
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

	// Description aus Meta Tags
	if desc, ok := data.MetaTags["description"]; ok {
		data.Description = desc
	}

	// Keywords extrahieren
	if keywords, ok := data.MetaTags["keywords"]; ok {
		data.Keywords = strings.Split(keywords, ",")
		for i, keyword := range data.Keywords {
			data.Keywords[i] = strings.TrimSpace(keyword)
		}
	}

	// Bilder extrahieren
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists && src != "" {
			data.Images = append(data.Images, src)
		}
	})

	// Links extrahieren
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists && href != "" {
			data.Links = append(data.Links, href)
		}
	})

	// Text extrahieren (ohne HTML Tags)
	data.Text = doc.Text()

	s.logger.Infof("Website erfolgreich gescraped: %s (Status: %d)", url, resp.StatusCode)

	return data, nil
}

func (s *Service) ScrapeMultipleWebsites(ctx context.Context, urls []string) ([]*ScrapedData, error) {
	s.logger.Infof("Scraping %d Websites", len(urls))

	results := make([]*ScrapedData, 0, len(urls))
	errors := make([]error, 0)

	// Semaphore für Concurrency Control
	semaphore := make(chan struct{}, 5) // Max 5 gleichzeitige Requests

	for _, url := range urls {
		semaphore <- struct{}{} // Semaphore erwerben

		go func(u string) {
			defer func() { <-semaphore }() // Semaphore freigeben

			data, err := s.ScrapeWebsite(ctx, u)
			if err != nil {
				s.logger.Errorf("Fehler beim Scrapen von %s: %v", u, err)
				errors = append(errors, err)
				return
			}

			results = append(results, data)
		}(url)
	}

	// Warten bis alle Goroutines fertig sind
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- struct{}{}
	}

	s.logger.Infof("Scraping abgeschlossen: %d erfolgreich, %d Fehler", len(results), len(errors))

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