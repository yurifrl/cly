package extractors

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
)

// Images extracts product images
type Images struct{}

func (e *Images) Name() string {
	return "images"
}

func (e *Images) Extract(ctx context.Context, productID string) (interface{}, error) {
	var imageURLs []string

	// Extract image URLs from page - port from Node.js reference
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const urls = [];
				const gallery = document.querySelector('[class*="gallery"], [class*="image-view"], [class*="ImageView"], [class*="slider"]');

				if (gallery) {
					const imageElements = gallery.querySelectorAll('img[src*="/kf/"]');

					imageElements.forEach(img => {
						let src = img.src || img.getAttribute('src');
						if (!src || !src.includes('/kf/')) return;

						// Clean up URL to get base
						let cleanUrl = src
							.replace(/_\d+x\d+q?\d*\.jpg_\.avif?$/i, '')
							.replace(/_\d+x\d+q?\d*\.jpg_\.avi$/i, '')
							.replace(/_\d+x\d+\.png_\.avif?$/i, '')
							.replace(/_\d+x\d+q?\d*\.jpg$/i, '')
							.replace(/_\d+x\d+\.png$/i, '')
							.replace(/\.avif$/i, '')
							.replace(/\.webp$/i, '');

						const kfMatch = cleanUrl.match(/(https?:\/\/[^\/]+\/kf\/[A-Za-z0-9]+)/);
						if (!kfMatch) return;

						let baseUrl = kfMatch[1];
						const isPng = src.match(/\.png/i) && !src.match(/\.jpe?g/i);

						let highRes;
						if (baseUrl.match(/\.(jpe?g|png)$/i)) {
							highRes = baseUrl;
						} else {
							highRes = isPng ? baseUrl + '.png' : baseUrl + '.jpg';
						}

						if (!urls.includes(highRes)) {
							urls.push(highRes);
						}
					});
				}

				return urls;
			})()
		`, &imageURLs),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to extract images: %w", err)
	}

	return imageURLs, nil
}
