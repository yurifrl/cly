package extractors

import (
	"context"

	"github.com/chromedp/chromedp"
	"github.com/yurifrl/cly/modules/scraper/aliexpress"
)

type Specs struct{}

func (e *Specs) Name() string { return "specs" }

func (e *Specs) Extract(ctx context.Context, productID string) (interface{}, error) {
	var specs []aliexpress.Spec

	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const specList = [];

				// Try multiple selectors for specification section
				const specSelectors = [
					'[class*="specification"]',
					'[class*="product-prop"]',
					'.product-specs',
					'[id*="specification"]'
				];

				let specSection = null;
				for (const selector of specSelectors) {
					specSection = document.querySelector(selector);
					if (specSection) break;
				}

				if (specSection) {
					const rows = specSection.querySelectorAll('tr, li, [class*="prop-item"]');

					rows.forEach(row => {
						try {
							// Try colon-separated format
							const text = row.textContent.trim();
							const match = text.match(/^(.+?):\s*(.+)$/);
							if (match) {
								const attrName = match[1].trim();
								const attrValue = match[2].trim();

								if (attrName && attrValue && attrName !== attrValue) {
									specList.push({ attrName, attrValue });
								}
							} else {
								// Try cell-based extraction
								const cells = row.querySelectorAll('td, span, div');
								if (cells.length >= 2) {
									const attrName = cells[0].textContent.trim();
									const attrValue = cells[1].textContent.trim();

									if (attrName && attrValue && attrName !== attrValue) {
										specList.push({ attrName, attrValue });
									}
								}
							}
						} catch (err) {
							// Skip invalid rows
						}
					});
				}

				return specList;
			})()
		`, &specs),
	)

	if err != nil {
		return []aliexpress.Spec{}, nil
	}

	return specs, nil
}
