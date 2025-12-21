package extractors

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/chromedp"
)

// Images extracts product images
type Images struct{}

func (e *Images) Name() string {
	return "images"
}

func (e *Images) Extract(ctx context.Context, productID string) (interface{}, error) {
	var imageURLs []string

	// Extract image URLs from page
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('img[class*="magnifier"]')).map(img => img.src)
		`, &imageURLs),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to extract images: %w", err)
	}

	// Convert thumbnail URLs to high-res
	var highResURLs []string
	for _, url := range imageURLs {
		// Convert _220x220.jpg to full resolution
		url = strings.ReplaceAll(url, "_220x220", "")
		url = strings.ReplaceAll(url, ".jpg_.webp", ".jpg")
		highResURLs = append(highResURLs, url)
	}

	return highResURLs, nil
}
