package extractors

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chromedp/chromedp"
)

// BasicInfo extracts basic product information
type BasicInfo struct{}

// BasicInfoData holds basic product information
type BasicInfoData struct {
	ProductID  int64
	Title      string
	CategoryID int64
	Orders     string
}

func (e *BasicInfo) Name() string {
	return "basic_info"
}

func (e *BasicInfo) Extract(ctx context.Context, productID string) (interface{}, error) {
	var title string
	var orders string

	err := chromedp.Run(ctx,
		// Extract title
		chromedp.Evaluate(`document.querySelector('h1')?.innerText || document.querySelector('.product-title-text')?.innerText || ''`, &title),
		// Extract orders count
		chromedp.Evaluate(`document.querySelector('.product-reviewer-soldnum')?.innerText || ''`, &orders),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to extract basic info: %w", err)
	}

	pid, _ := strconv.ParseInt(productID, 10, 64)

	return &BasicInfoData{
		ProductID: pid,
		Title:     title,
		Orders:    orders,
	}, nil
}
