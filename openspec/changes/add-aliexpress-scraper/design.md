# Design: AliExpress Scraper Module

## Architecture Overview

```
modules/scraper/
  ├── cmd/
  │   ├── root.go           # scraper parent command
  │   ├── browser.go        # browser launcher subcommand
  │   └── aliexpress.go     # aliexpress scraper subcommand
  ├── browser/
  │   ├── controller.go     # chromedp browser controller
  │   ├── captcha.go        # CAPTCHA detection
  │   └── session.go        # session persistence
  ├── aliexpress/
  │   ├── scraper.go        # orchestration
  │   ├── extractors/       # 10 extractors (basic_info, images, etc.)
  │   └── types.go          # data structures
  ├── input/
  │   ├── parser.go         # URL/file parsing
  │   └── validator.go      # ID/URL validation
  ├── output/
  │   ├── writer.go         # JSON output
  │   └── streamer.go       # streaming append
  └── tui/
      ├── progress.go       # progress model
      └── status.go         # status view
```

## Key Components

### Browser Controller
- Manages chromedp lifecycle (launch, connect, close)
- Persistent session via `--user-data-dir`
- Remote debugging on port 9222
- CAPTCHA detection: check for reCAPTCHA iframe or challenge text
- Wait strategy: networkidle + 15s minimum (AliExpress needs full JS load)

### Scraper Orchestration
- Navigate to product page
- Run extractors in parallel (goroutines)
- Aggregate results into ProductData struct
- Handle partial failures (missing data sections)
- Stream results to output

### Input Parser
- Parse product IDs/URLs from CLI flag (comma-separated)
- Parse files: TXT (newline/comma-separated), CSV, JSON, YAML
- Extract product ID from full URLs
- Validate product ID format

### Output Writer
- Single file mode: JSON array, streaming append
- Per-URL mode: one JSON file per product
- Atomic writes
- Progress tracking integration

### TUI Progress Display
- Bubbletea model with Update/View pattern
- Real-time progress bar
- Per-product status (pending, scraping, done, failed)
- Total stats (completed/failed/current)
- Responsive to Ctrl+C (graceful shutdown)

## Data Flow

```
CLI Input
  ↓
Parse URLs/File → Product IDs
  ↓
For each ID:
  Browser.NavigateToProduct(id)
  Browser.WaitForLoad()
  Browser.DetectCAPTCHA() → (if found, warn user)
  ↓
  Parallel Extraction (goroutines):
    - Basic Info     - Reviews
    - Images         - Ratings
    - Variants       - Store
    - Specs          - Shipping
    - Description    - Currency
    - Quantity
  ↓
  Aggregate → ProductData
  ↓
  Output.WriteProduct(data)
  ↓
  TUI.UpdateProgress()
  ↓
Next ID
```

## Configuration

Config file (`config/config.yaml`):

```yaml
scraper:
  aliexpress:
    browser:
      headless: false
      debug_port: 9222
      user_data_dir: ~/.cly/scraper/chrome
      timeout: 60s
      wait_time: 15s
    reviews_count: 20
    filter_reviews_by: "all"
    output_mode: "single"
    output_dir: "./scraped"
    delay_between_products: 5s
    max_retries: 3
```

## Command Interface

```bash
# Browser launcher
cly scraper browser

# Scrape single/multiple
cly scraper aliexpress --url 1005003618976317
cly scraper aliexpress --url 1005003618976317,1005010081760632
cly scraper aliexpress --url "https://www.aliexpress.com/item/1005003618976317.html"

# Scrape from file
cly scraper aliexpress -f products.txt
cly scraper aliexpress -f products.csv
cly scraper aliexpress -f products.json
cly scraper aliexpress -f products.yaml

# Output options
cly scraper aliexpress --url ID --output results.json
cly scraper aliexpress --url ID --output-per-url
cly scraper aliexpress --url ID --output-dir ./scraped/

# Browser connection
cly scraper aliexpress --browser-url http://localhost:9222
```

## Module Registration

Follows CLY's existing pattern:

```go
// modules/scraper/cmd/root.go
func Register(parent *cobra.Command) {
    parent.AddCommand(ScraperCmd)
}

// cmd/root.go
import "github.com/yurifrl/cly/modules/scraper"

func init() {
    scraper.Register(RootCmd)
}
```

## Error Handling Philosophy

- **Partial success**: If 8/10 extractors succeed, output product with warnings
- **Continue on failure**: One failed product doesn't stop batch
- **Detailed errors**: Log which extractor failed and why
- **Graceful degradation**: Missing optional fields don't fail scrape

## Testing Strategy

- Unit tests: Input parser, ID extraction, output writer
- Integration tests: Individual extractors (mock chromedp context)
- Schema validation: Automated tests validate output JSON structure against reference schema
- Manual tests: Full scrape with real browser (requires CAPTCHA solving)
- Table-driven tests for file format parsing

### JSON Schema Validation

All scraped output must match the reference schema from the blueprint example:

**Required top-level keys:**
- title, categoryId, productId, quantity, description, orders
- storeInfo, ratings, images, reviews, variants, specs
- currencyInfo, originalPrice, salePrice, shipping

**Type validation:**
- Strings: title, description, orders
- Integers: categoryId, productId
- Objects: quantity, storeInfo, ratings, variants, currencyInfo, originalPrice, salePrice
- Arrays: images, reviews, specs, shipping

**Nested structure validation:**
- storeInfo: name, logo, companyId (int), storeNumber (int), isTopRated (bool), hasPayPalAccount (bool), ratingCount (int), rating (string)
- ratings: totalStar (int), averageStar (string), totalStartCount (int), fiveStarCount (int), fourStarCount (int), threeStarCount (int), twoStarCount (int), oneStarCount (int)
- reviews[]: anonymous (bool), name, displayName, gender, country, rating (int 1-5), info, date, content, photos (array), thumbnails (array)
- variants: options (array), prices (array)

Schema validation runs automatically in integration tests. Any deviation from the reference structure fails the build.

## Parallel Extraction Pattern

```go
func (s *Scraper) extractAll(ctx context.Context, productID string) (*ProductData, error) {
    var wg sync.WaitGroup
    results := make(chan extractResult, len(s.extractors))

    for _, extractor := range s.extractors {
        wg.Add(1)
        go func(ex Extractor) {
            defer wg.Done()
            data, err := ex.Extract(ctx, productID)
            results <- extractResult{name: ex.Name(), data: data, err: err}
        }(extractor)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    product := &ProductData{}
    for result := range results {
        if result.err != nil {
            log.Warn().Err(result.err).Str("extractor", result.name).Msg("extraction failed")
            continue
        }
        // Map result to product fields
    }

    return product, nil
}
```

## Chromedp Integration

- Use `chromedp.NewRemoteAllocator()` to connect to existing browser
- Or `chromedp.NewExecAllocator()` to launch new instance
- Context hierarchy: allocator → executor → tab
- Actions: `chromedp.Navigate()`, `chromedp.WaitVisible()`, `chromedp.Evaluate()`

## Migration from Node.js Reference

**Direct ports:**
- DOM selectors → chromedp equivalents
- Image URL conversion logic (same)
- Wait strategy (15s + networkidle)
- Output format (maintain JSON structure)

**Improvements:**
- Go concurrency (goroutines) vs Promise.all
- Streaming output vs in-memory
- Native binary vs npm package
- Better integration with CLY config/TUI

## Open Design Questions

1. **Rate limiting strategy**: Default 5s delay, but should be user-configurable?
   - **Decision**: Make configurable via config file and `--delay` flag

2. **Headless mode support**: Allow headless for automated runs?
   - **Decision**: Make configurable but default to `false` (can't solve CAPTCHA in headless)

3. **Retry logic**: How many retries for failed extractions?
   - **Decision**: Configurable `max_retries` (default 3), exponential backoff

4. **Output format alternatives**: Support CSV, SQLite?
   - **Decision**: JSON only for MVP, can add later via output adapters
