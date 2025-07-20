package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
	// New fields for advanced crawling
	Headers    map[string]string `json:"headers,omitempty"`
	Forms      []FormData        `json:"forms,omitempty"`
	Tables     []TableData       `json:"tables,omitempty"`
	Scripts    []string          `json:"scripts,omitempty"`
	Styles     []string          `json:"styles,omitempty"`
	H1Tags     []string          `json:"h1_tags,omitempty"`
	H2Tags     []string          `json:"h2_tags,omitempty"`
	H3Tags     []string          `json:"h3_tags,omitempty"`
	CustomData map[string]string `json:"custom_data,omitempty"`
}

type FormData struct {
	Action string      `json:"action"`
	Method string      `json:"method"`
	Inputs []FormInput `json:"inputs"`
}

type FormInput struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type TableData struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

type CrawlingOptions struct {
	// Basic options
	MaxDepth int           `json:"max_depth"`
	MaxPages int           `json:"max_pages"`
	Timeout  time.Duration `json:"timeout"`
	Delay    time.Duration `json:"delay"`

	// Filtering options
	IncludePatterns []string `json:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	AllowedDomains  []string `json:"allowed_domains"`

	// Extraction options
	ExtractImages  bool `json:"extract_images"`
	ExtractLinks   bool `json:"extract_links"`
	ExtractForms   bool `json:"extract_forms"`
	ExtractTables  bool `json:"extract_tables"`
	ExtractScripts bool `json:"extract_scripts"`
	ExtractStyles  bool `json:"extract_styles"`
	ExtractHeaders bool `json:"extract_headers"`

	// Custom selectors
	CustomSelectors map[string]string `json:"custom_selectors"`

	// User agent and headers
	UserAgent string            `json:"user_agent"`
	Headers   map[string]string `json:"headers"`

	// Follow redirects
	FollowRedirects bool `json:"follow_redirects"`

	// Respect robots.txt
	RespectRobotsTxt bool `json:"respect_robots_txt"`
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
	return s.ScrapeWebsiteWithOptions(ctx, url, &CrawlingOptions{
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
	})
}

func (s *Service) ScrapeWebsiteWithOptions(ctx context.Context, url string, options *CrawlingOptions) (*ScrapedData, error) {
	s.logger.Infof("Scraping website: %s with options", url)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent
	if options.UserAgent != "" {
		req.Header.Set("User-Agent", options.UserAgent)
	} else {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	}

	// Set custom headers
	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}

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
		Headers:    make(map[string]string),
		CustomData: make(map[string]string),
	}

	// Extract response headers
	for key, values := range resp.Header {
		data.Headers[key] = values[0]
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

	// Extract images if enabled
	if options.ExtractImages {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists && src != "" {
				data.Images = append(data.Images, src)
			}
		})
	}

	// Extract links if enabled
	if options.ExtractLinks {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists && href != "" {
				data.Links = append(data.Links, href)
			}
		})
	}

	// Extract forms if enabled
	if options.ExtractForms {
		doc.Find("form").Each(func(i int, s *goquery.Selection) {
			form := FormData{}
			form.Action, _ = s.Attr("action")
			form.Method, _ = s.Attr("method")
			if form.Method == "" {
				form.Method = "GET"
			}

			s.Find("input").Each(func(j int, input *goquery.Selection) {
				inputData := FormInput{}
				inputData.Name, _ = input.Attr("name")
				inputData.Type, _ = input.Attr("type")
				inputData.Value, _ = input.Attr("value")
				form.Inputs = append(form.Inputs, inputData)
			})

			data.Forms = append(data.Forms, form)
		})
	}

	// Extract tables if enabled
	if options.ExtractTables {
		doc.Find("table").Each(func(i int, s *goquery.Selection) {
			table := TableData{}

			// Extract headers
			s.Find("thead tr th, tr th").Each(func(j int, th *goquery.Selection) {
				table.Headers = append(table.Headers, strings.TrimSpace(th.Text()))
			})

			// Extract rows
			s.Find("tbody tr, tr").Each(func(j int, tr *goquery.Selection) {
				var row []string
				tr.Find("td").Each(func(k int, td *goquery.Selection) {
					row = append(row, strings.TrimSpace(td.Text()))
				})
				if len(row) > 0 {
					table.Rows = append(table.Rows, row)
				}
			})

			data.Tables = append(data.Tables, table)
		})
	}

	// Extract scripts if enabled
	if options.ExtractScripts {
		doc.Find("script").Each(func(i int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists && src != "" {
				data.Scripts = append(data.Scripts, src)
			}
		})
	}

	// Extract styles if enabled
	if options.ExtractStyles {
		doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists && href != "" {
				data.Styles = append(data.Styles, href)
			}
		})
	}

	// Extract headers if enabled
	if options.ExtractHeaders {
		doc.Find("h1").Each(func(i int, s *goquery.Selection) {
			data.H1Tags = append(data.H1Tags, strings.TrimSpace(s.Text()))
		})
		doc.Find("h2").Each(func(i int, s *goquery.Selection) {
			data.H2Tags = append(data.H2Tags, strings.TrimSpace(s.Text()))
		})
		doc.Find("h3").Each(func(i int, s *goquery.Selection) {
			data.H3Tags = append(data.H3Tags, strings.TrimSpace(s.Text()))
		})
	}

	// Extract custom data using custom selectors
	for key, selector := range options.CustomSelectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			data.CustomData[key] = strings.TrimSpace(s.Text())
		})
	}

	// Extract text (without HTML tags)
	data.Text = doc.Text()

	s.logger.Infof("Website successfully scraped: %s (Status: %d)", url, resp.StatusCode)

	return data, nil
}

func (s *Service) ScrapeMultipleWebsites(ctx context.Context, urls []string) ([]*ScrapedData, error) {
	return s.ScrapeMultipleWebsitesWithOptions(ctx, urls, &CrawlingOptions{
		MaxDepth:         1,
		MaxPages:         len(urls),
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
	})
}

func (s *Service) ScrapeMultipleWebsitesWithOptions(ctx context.Context, urls []string, options *CrawlingOptions) ([]*ScrapedData, error) {
	s.logger.Infof("Scraping %d websites with options", len(urls))

	results := make([]*ScrapedData, 0, len(urls))
	errors := make([]error, 0)

	// Filter URLs based on patterns
	filteredUrls := s.filterUrls(urls, options)

	// Semaphore for concurrency control
	semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

	for _, url := range filteredUrls {
		semaphore <- struct{}{} // Acquire semaphore

		go func(u string) {
			defer func() { <-semaphore }() // Release semaphore

			// Add delay if specified
			if options.Delay > 0 {
				time.Sleep(options.Delay)
			}

			data, err := s.ScrapeWebsiteWithOptions(ctx, u, options)
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

func (s *Service) filterUrls(urls []string, options *CrawlingOptions) []string {
	if len(options.IncludePatterns) == 0 && len(options.ExcludePatterns) == 0 && len(options.AllowedDomains) == 0 {
		return urls
	}

	var filtered []string

	for _, u := range urls {
		// Check if URL matches include patterns
		if len(options.IncludePatterns) > 0 {
			matched := false
			for _, pattern := range options.IncludePatterns {
				if isMatched, _ := regexp.MatchString(pattern, u); isMatched {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Check if URL matches exclude patterns
		if len(options.ExcludePatterns) > 0 {
			excluded := false
			for _, pattern := range options.ExcludePatterns {
				if isMatched, _ := regexp.MatchString(pattern, u); isMatched {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		// Check if URL domain is allowed
		if len(options.AllowedDomains) > 0 {
			parsedURL, err := url.Parse(u)
			if err != nil {
				continue
			}
			allowed := false
			for _, domain := range options.AllowedDomains {
				if parsedURL.Hostname() == domain {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		filtered = append(filtered, u)
	}

	return filtered
}

func (s *Service) GetWebsiteStats(ctx context.Context, url string) (map[string]interface{}, error) {
	return s.GetWebsiteStatsWithOptions(ctx, url, &CrawlingOptions{
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
	})
}

func (s *Service) GetWebsiteStatsWithOptions(ctx context.Context, url string, options *CrawlingOptions) (map[string]interface{}, error) {
	data, err := s.ScrapeWebsiteWithOptions(ctx, url, options)
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
		// New stats
		"form_count":        len(data.Forms),
		"table_count":       len(data.Tables),
		"script_count":      len(data.Scripts),
		"style_count":       len(data.Styles),
		"h1_count":          len(data.H1Tags),
		"h2_count":          len(data.H2Tags),
		"h3_count":          len(data.H3Tags),
		"custom_data_count": len(data.CustomData),
	}

	return stats, nil
}
