package aliexpress

import (
	"context"
	"fmt"
	"time"

	"github.com/yurifrl/cly/modules/scraper/browser"
)

// Scraper orchestrates product data extraction
type Scraper struct {
	browser          *browser.Controller
	extractors       []Extractor
	progressCallback func(extractorName string)
}

// NewScraper creates a new scraper instance
func NewScraper(ctrl *browser.Controller, extractors []Extractor) *Scraper {
	return &Scraper{
		browser:    ctrl,
		extractors: extractors,
	}
}

// SetProgressCallback sets a callback for extractor progress updates
func (s *Scraper) SetProgressCallback(callback func(extractorName string)) {
	s.progressCallback = callback
}


// ScrapeProduct scrapes a single product
func (s *Scraper) ScrapeProduct(productID string) (*ProductData, error) {
	// Navigate to product
	if err := s.browser.NavigateToProduct(productID); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for page load
	if err := s.browser.WaitForLoad(); err != nil {
		return nil, fmt.Errorf("wait failed: %w", err)
	}

	// Check for CAPTCHA
	hasCaptcha, err := s.browser.DetectCAPTCHA()
	if err == nil && hasCaptcha {
		return nil, fmt.Errorf("CAPTCHA detected, please solve manually")
	}

	// Extract all data in parallel
	ctx := s.browser.GetContext()
	return s.extractAll(ctx, productID)
}

// extractAll runs all extractors sequentially (single browser tab)
func (s *Scraper) extractAll(ctx context.Context, productID string) (*ProductData, error) {
	product := &ProductData{
		ScrapedAt: time.Now(),
		SourceURL: fmt.Sprintf("https://www.aliexpress.com/item/%s.html", productID),
	}

	// Run extractors SEQUENTIALLY to avoid browser tab conflicts
	for _, extractor := range s.extractors {
		// Send progress update
		if s.progressCallback != nil {
			s.progressCallback(extractor.Name())
		}

		data, err := extractor.Extract(ctx, productID)

		if err != nil {
			// Log warning but continue with other extractors
			continue
		}

		// Map result to product fields
		switch extractor.Name() {
		case "basic_info":
			if d, ok := data.(*struct {
				ProductID  int64
				Title      string
				CategoryID int64
				Orders     string
			}); ok {
				product.ProductID = d.ProductID
				product.Title = d.Title
				product.CategoryID = d.CategoryID
				product.Orders = d.Orders
			}
		case "images":
			if d, ok := data.([]string); ok {
				product.Images = d
			}
		case "variants":
			if d, ok := data.(*VariantsData); ok {
				product.Variants = *d
			}
		case "reviews":
			if d, ok := data.([]Review); ok {
				product.Reviews = d
			}
		case "ratings":
			if d, ok := data.(*RatingsData); ok {
				product.Ratings = *d
			}
		case "store":
			if d, ok := data.(*StoreInfo); ok {
				product.StoreInfo = *d
			}
		case "specs":
			if d, ok := data.([]Spec); ok {
				product.Specs = d
			}
		case "shipping":
			if d, ok := data.([]Shipping); ok {
				product.Shipping = d
			}
		case "description":
			if d, ok := data.(string); ok {
				product.Description = d
			}
		case "currency":
			if d, ok := data.(*Currency); ok {
				product.Currency = *d
			}
		case "quantity":
			if d, ok := data.(*Quantity); ok {
				product.Quantity = *d
			}
		}
	}

	return product, nil
}

// ScrapeBatch scrapes multiple products
func (s *Scraper) ScrapeBatch(productIDs []string) ([]*ProductData, []error) {
	var products []*ProductData
	var errors []error

	for _, id := range productIDs {
		product, err := s.ScrapeProduct(id)
		if err != nil {
			errors = append(errors, fmt.Errorf("product %s: %w", id, err))
			continue
		}
		products = append(products, product)
	}

	return products, errors
}
