package aliexpress

import "context"

// Extractor is the interface for data extractors
type Extractor interface {
	Name() string
	Extract(ctx context.Context, productID string) (interface{}, error)
}
