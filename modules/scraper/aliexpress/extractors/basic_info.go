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
	var categoryID int64
	var orders string

	err := chromedp.Run(ctx,
		// Extract title - find longest h1 that's not "aliexpress"
		chromedp.Evaluate(`
			(() => {
				const h1Elements = document.querySelectorAll('h1');
				let productTitle = '';
				h1Elements.forEach(h1 => {
					const text = h1.textContent.trim();
					if (text.length > productTitle.length && text.toLowerCase() !== 'aliexpress') {
						productTitle = text;
					}
				});
				return productTitle || '';
			})()
		`, &title),
		// Extract category ID
		chromedp.Evaluate(`
			(() => {
				try {
					const categoryLink = document.querySelector('[data-category-id]');
					return categoryLink ? parseInt(categoryLink.getAttribute('data-category-id'), 10) : 0;
				} catch {
					return 0;
				}
			})()
		`, &categoryID),
		// Extract orders count
		chromedp.Evaluate(`
			(() => {
				const ordersText = document.body.innerText.match(/(\d+[\d,]*)\+?\s*(orders?|sold)/i);
				return ordersText ? ordersText[1] + '+' : '';
			})()
		`, &orders),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to extract basic info: %w", err)
	}

	if title == "" {
		return nil, fmt.Errorf("title not found - critical field")
	}

	pid, _ := strconv.ParseInt(productID, 10, 64)

	return &BasicInfoData{
		ProductID:  pid,
		Title:      title,
		CategoryID: categoryID,
		Orders:     orders,
	}, nil
}
