package extractors

import (
	"context"

	"github.com/chromedp/chromedp"
	"github.com/yurifrl/cly/modules/scraper/aliexpress"
)

type Ratings struct{}

func (e *Ratings) Name() string { return "ratings" }

func (e *Ratings) Extract(ctx context.Context, productID string) (interface{}, error) {
	var ratings aliexpress.RatingsData

	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const result = {
					totalStar: 5,
					averageStar: '0',
					totalStartCount: 0,
					fiveStarCount: 0,
					fourStarCount: 0,
					threeStarCount: 0,
					twoStarCount: 0,
					oneStarCount: 0
				};

				// Find average rating
				const avgRating = document.querySelector('[class*="average"], [class*="rating"] [class*="star"]');
				if (avgRating) {
					const match = avgRating.textContent.match(/(\d+\.?\d*)/);
					if (match) {
						result.averageStar = match[1];
					}
				}

				// Find total rating count
				const totalText = document.body.innerText.match(/(\d+[\d,]*)\s*(?:rating|review)s?/i);
				if (totalText) {
					result.totalStartCount = parseInt(totalText[1].replace(/,/g, ''), 10);
				}

				// Find star breakdown
				const starElements = document.querySelectorAll('[class*="rating-star"], [class*="review-star"]');
				starElements.forEach(el => {
					const text = el.textContent;
					const fiveStar = text.match(/5\s*star[s]?[:\s]+(\d+)/i);
					const fourStar = text.match(/4\s*star[s]?[:\s]+(\d+)/i);
					const threeStar = text.match(/3\s*star[s]?[:\s]+(\d+)/i);
					const twoStar = text.match(/2\s*star[s]?[:\s]+(\d+)/i);
					const oneStar = text.match(/1\s*star[s]?[:\s]+(\d+)/i);

					if (fiveStar) result.fiveStarCount = parseInt(fiveStar[1], 10);
					if (fourStar) result.fourStarCount = parseInt(fourStar[1], 10);
					if (threeStar) result.threeStarCount = parseInt(threeStar[1], 10);
					if (twoStar) result.twoStarCount = parseInt(twoStar[1], 10);
					if (oneStar) result.oneStarCount = parseInt(oneStar[1], 10);
				});

				return result;
			})()
		`, &ratings),
	)

	if err != nil {
		return &aliexpress.RatingsData{}, nil
	}

	return &ratings, nil
}
