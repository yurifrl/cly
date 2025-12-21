package extractors

import (
	"context"

	"github.com/yurifrl/cly/modules/scraper/aliexpress"
)

// Stub extractors for not-yet-implemented features

type Variants struct{}

func (e *Variants) Name() string { return "variants" }
func (e *Variants) Extract(ctx context.Context, productID string) (interface{}, error) {
	return &aliexpress.VariantsData{}, nil
}

type Reviews struct{}

func (e *Reviews) Name() string { return "reviews" }
func (e *Reviews) Extract(ctx context.Context, productID string) (interface{}, error) {
	return []aliexpress.Review{}, nil
}

type ShippingExtractor struct{}

func (e *ShippingExtractor) Name() string { return "shipping" }
func (e *ShippingExtractor) Extract(ctx context.Context, productID string) (interface{}, error) {
	return []aliexpress.Shipping{}, nil
}

type Description struct{}

func (e *Description) Name() string { return "description" }
func (e *Description) Extract(ctx context.Context, productID string) (interface{}, error) {
	return "", nil
}

type CurrencyExtractor struct{}

func (e *CurrencyExtractor) Name() string { return "currency" }
func (e *CurrencyExtractor) Extract(ctx context.Context, productID string) (interface{}, error) {
	return &aliexpress.Currency{}, nil
}

type QuantityExtractor struct{}

func (e *QuantityExtractor) Name() string { return "quantity" }
func (e *QuantityExtractor) Extract(ctx context.Context, productID string) (interface{}, error) {
	return &aliexpress.Quantity{}, nil
}

// GetAllExtractors returns all available extractors
func GetAllExtractors() []aliexpress.Extractor {
	return []aliexpress.Extractor{
		&BasicInfo{},    // Ported with real selectors
		&Images{},       // Ported with real selectors
		&Ratings{},      // Ported with real selectors
		&Store{},        // Ported with real selectors
		&Specs{},        // Ported with real selectors
		&Variants{},     // Stub - TODO
		&Reviews{},      // Stub - TODO
		&ShippingExtractor{},  // Stub - TODO
		&Description{},        // Stub - TODO
		&CurrencyExtractor{},  // Stub - TODO
		&QuantityExtractor{},  // Stub - TODO
	}
}
