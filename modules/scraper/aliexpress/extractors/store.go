package extractors

import (
	"context"

	"github.com/chromedp/chromedp"
	"github.com/yurifrl/cly/modules/scraper/aliexpress"
)

type Store struct{}

func (e *Store) Name() string { return "store" }

func (e *Store) Extract(ctx context.Context, productID string) (interface{}, error) {
	var store aliexpress.StoreInfo

	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(() => {
				const info = {};

				// Store name
				const storeName = document.querySelector('[class*="store"] [class*="name"], .shop-name, .store-header__name');
				info.name = storeName ? storeName.textContent.trim() : '';

				// Store logo
				const storeLogo = document.querySelector('[class*="store"] img[src*="alicdn"], .shop-logo img, .store-header__logo img');
				info.logo = storeLogo ? storeLogo.src : '';

				// Store number from link
				const storeLink = document.querySelector('a[href*="store/"]');
				if (storeLink) {
					const match = storeLink.href.match(/store\/(\d+)/);
					if (match) {
						info.storeNumber = parseInt(match[1], 10);
					}
				}

				info.companyId = 0;

				// Top rated indicator
				const topRatedBadge = document.querySelector('[class*="top-rated"], [class*="topRated"]');
				info.isTopRated = !!topRatedBadge;

				// PayPal indicator
				const paypalIndicator = document.querySelector('[class*="paypal"], img[alt*="PayPal"]');
				info.hasPayPalAccount = !!paypalIndicator;

				// Store ratings
				const ratingText = document.body.innerText.match(/(\d+[\d,]*)\s*(?:store|shop)\s*(?:rating|review)s?/i);
				info.ratingCount = ratingText ? parseInt(ratingText[1].replace(/,/g, ''), 10) : 0;

				const ratingValue = document.body.innerText.match(/(?:store|shop)\s*rating[:\s]+(\d+\.?\d*)%?/i);
				info.rating = ratingValue ? ratingValue[1] : '0';

				return info;
			})()
		`, &store),
	)

	if err != nil {
		return &aliexpress.StoreInfo{}, nil
	}

	return &store, nil
}
