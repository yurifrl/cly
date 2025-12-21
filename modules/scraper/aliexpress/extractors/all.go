package extractors

import (
	"context"

	"github.com/yurifrl/cly/modules/scraper/aliexpress"
)

// Variants extractor
type Variants struct{}

func (e *Variants) Name() string { return "variants" }
func (e *Variants) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement variant extraction
	return &aliexpress.VariantsData{}, nil
}

// Reviews extractor
type Reviews struct{}

func (e *Reviews) Name() string { return "reviews" }
func (e *Reviews) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement reviews extraction
	return []aliexpress.Review{}, nil
}

// Ratings extractor
type Ratings struct{}

func (e *Ratings) Name() string { return "ratings" }
func (e *Ratings) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement ratings extraction
	return &aliexpress.RatingsData{}, nil
}

// Store extractor
type Store struct{}

func (e *Store) Name() string { return "store" }
func (e *Store) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement store info extraction
	return &aliexpress.StoreInfo{}, nil
}

// Specs extractor
type Specs struct{}

func (e *Specs) Name() string { return "specs" }
func (e *Specs) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement specs extraction
	return []aliexpress.Spec{}, nil
}

// ShippingExtractor extractor
type ShippingExtractor struct{}

func (e *ShippingExtractor) Name() string { return "shipping" }
func (e *ShippingExtractor) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement shipping extraction
	return []aliexpress.Shipping{}, nil
}

// Description extractor
type Description struct{}

func (e *Description) Name() string { return "description" }
func (e *Description) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement description extraction
	return "", nil
}

// CurrencyExtractor extractor
type CurrencyExtractor struct{}

func (e *CurrencyExtractor) Name() string { return "currency" }
func (e *CurrencyExtractor) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement currency extraction
	return &aliexpress.Currency{}, nil
}

// QuantityExtractor extractor
type QuantityExtractor struct{}

func (e *QuantityExtractor) Name() string { return "quantity" }
func (e *QuantityExtractor) Extract(ctx context.Context, productID string) (interface{}, error) {
	// TODO: Implement quantity extraction
	return &aliexpress.Quantity{}, nil
}

// GetAllExtractors returns all available extractors
func GetAllExtractors() []aliexpress.Extractor {
	return []aliexpress.Extractor{
		&BasicInfo{},
		&Images{},
		&Variants{},
		&Reviews{},
		&Ratings{},
		&Store{},
		&Specs{},
		&ShippingExtractor{},
		&Description{},
		&CurrencyExtractor{},
		&QuantityExtractor{},
	}
}
