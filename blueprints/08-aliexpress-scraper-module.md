# Blueprint: AliExpress Scraper Module for Cly

## Overview

Add `cly scraper aliexpress` command to scrape AliExpress product data using persistent browser automation (Chromedp). Supports batch processing, flexible input formats, and JSON output streaming.

## CLI Interface

```bash
# Single/multiple URLs or IDs
cly scraper aliexpress --url 1005003618976317,1005010081760632
cly scraper aliexpress --url "https://www.aliexpress.com/item/1005003618976317.html"

# From file (CSV, newline-separated, JSON, YAML)
cly scraper aliexpress -f products.txt
cly scraper aliexpress -f products.json
cly scraper aliexpress -f products.yaml

# Output options
cly scraper aliexpress --url ID --output results.json          # All in one file (default)
cly scraper aliexpress --url ID --output-per-url               # One file per URL
cly scraper aliexpress --url ID --output-dir ./scraped/        # Custom output directory

# Browser management
cly scraper browser                                             # Launch persistent browser with UI
cly scraper aliexpress --browser-url http://localhost:9222     # Connect to existing browser
```

## Architecture

### Module Structure

```
modules/
  scraper/
    cmd/
      root.go              # scraper subcommand
      aliexpress.go        # aliexpress subcommand
      browser.go           # browser launcher subcommand

    browser/
      controller.go        # Chromedp browser controller
      captcha.go           # CAPTCHA detection & handling
      session.go           # Session persistence

    aliexpress/
      scraper.go           # Main scraper orchestration
      extractors/
        basic_info.go      # Title, ID, category
        images.go          # High-res image URLs
        variants.go        # Price, SKU, options
        reviews.go         # Reviews with photos
        ratings.go         # Star ratings breakdown
        store.go           # Store information
        specs.go           # Product specifications
        shipping.go        # Shipping options
        description.go     # Product description
        currency.go        # Currency info
      types.go             # Product data structures

    input/
      parser.go            # Parse URLs from various formats
      validator.go         # Validate product IDs/URLs

    output/
      writer.go            # JSON output writer
      streamer.go          # Streaming JSON appender

    tui/
      progress.go          # Progress bars (bubbletea)
      status.go            # Real-time status display
```

### Key Components

#### 1. Browser Controller (`browser/controller.go`)

```go
type BrowserController struct {
    ctx    context.Context
    cancel context.CancelFunc

    // Chromedp allocator & context
    allocCtx context.Context
    execCtx  context.Context

    // Config
    debugPort int
    headless  bool
    userDataDir string
}

func NewBrowserController(opts BrowserOptions) *BrowserController
func (bc *BrowserController) Launch() error
func (bc *BrowserController) Connect(debugURL string) error
func (bc *BrowserController) NavigateToProduct(productID string) error
func (bc *BrowserController) WaitForLoad() error
func (bc *BrowserController) DetectCAPTCHA() (bool, error)
func (bc *BrowserController) GetPage() (chromedp.Context, error)
func (bc *BrowserController) Close() error
```

**Implementation notes:**
- Use `chromedp` for browser automation (native Go, no Node.js)
- Launch with `--remote-debugging-port=9222` for persistence
- Detect CAPTCHA: check for reCAPTCHA iframe or challenge text
- Wait strategy: networkidle + 15s minimum (AliExpress needs time)
- Session persistence via `--user-data-dir` to avoid repeated CAPTCHAs

#### 2. Scraper Orchestration (`aliexpress/scraper.go`)

```go
type Scraper struct {
    browser    *browser.BrowserController
    extractors []Extractor
    output     *output.Writer
}

type Extractor interface {
    Extract(ctx context.Context, productID string) (interface{}, error)
}

func (s *Scraper) ScrapeProduct(productID string) (*ProductData, error) {
    // Navigate to product page
    // Wait for load
    // Run extractors in parallel (goroutines)
    // Aggregate results
    // Write to output
}

func (s *Scraper) ScrapeBatch(productIDs []string) error {
    // Iterate through IDs
    // Scrape each with same browser instance
    // Stream results to output file
    // Update TUI progress
}
```

**Parallel extraction:**
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

    // Aggregate results
    product := &ProductData{}
    for result := range results {
        if result.err != nil {
            log.Warn().Err(result.err).Str("extractor", result.name).Msg("extraction failed")
            continue
        }
        // Map result to product struct fields
    }

    return product, nil
}
```

#### 3. Input Parser (`input/parser.go`)

```go
type InputParser struct{}

func (p *InputParser) ParseURLs(input string) ([]string, error)
func (p *InputParser) ParseFile(filePath string) ([]string, error)
func (p *InputParser) ParseCSV(content string) ([]string, error)
func (p *InputParser) ParseJSON(content []byte) ([]string, error)
func (p *InputParser) ParseYAML(content []byte) ([]string, error)

func ExtractProductID(urlOrID string) string {
    // Handle both:
    // - https://www.aliexpress.com/item/1005003618976317.html
    // - 1005003618976317
}
```

**File format support:**
- **TXT**: Newline-separated or comma-separated
- **CSV**: Single column or multiple columns (detect ID column)
- **JSON**: `["id1", "id2"]` or `[{"url": "..."}, ...]`
- **YAML**: List of IDs or objects

#### 4. Output Writer (`output/writer.go`)

```go
type OutputMode int

const (
    SingleFile OutputMode = iota  // All products in one JSON
    PerURL                         // One file per product
)

type Writer struct {
    mode      OutputMode
    outputDir string
    file      *os.File
    encoder   *json.Encoder
}

func (w *Writer) WriteProduct(product *ProductData) error {
    if w.mode == PerURL {
        return w.writeToSeparateFile(product)
    }
    return w.appendToSingleFile(product)
}

func (w *Writer) appendToSingleFile(product *ProductData) error {
    // Read existing file
    // Append new product to array
    // Write back (atomic)
    // OR: Use streaming JSON encoder
}
```

**Streaming approach for single file:**
```go
// Start with opening bracket
// Append each product with comma
// Close with bracket on finalize
type StreamWriter struct {
    file    *os.File
    isFirst bool
}

func (sw *StreamWriter) WriteProduct(p *ProductData) error {
    if !sw.isFirst {
        sw.file.Write([]byte(",\n"))
    }
    sw.isFirst = false

    data, _ := json.MarshalIndent(p, "  ", "  ")
    sw.file.Write(data)
    return nil
}

func (sw *StreamWriter) Finalize() error {
    sw.file.Write([]byte("\n]\n"))
    return sw.file.Close()
}
```

#### 5. TUI Progress Display (`tui/progress.go`)

```go
type ProgressModel struct {
    total     int
    completed int
    failed    int
    current   string
    products  []ProductStatus
}

type ProductStatus struct {
    ID     string
    Status string  // "pending", "scraping", "done", "failed"
    Error  string
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m ProgressModel) View() string
```

**Display:**
```
Scraping AliExpress Products

Progress: ████████████░░░░░░░░ 12/20 (60%)

Current: 1005003618976317

✓ 1005010081760632 (32.4s)
✓ 1005004455545666 (28.1s)
⏳ 1005003618976317 (scraping...)
○ 1005008596183124 (pending)
✗ 1005008633928425 (CAPTCHA required)

Output: products.json (12 products, 2.4 MB)
```

### Data Structures

```go
type ProductData struct {
    ProductID   int64           `json:"productId"`
    Title       string          `json:"title"`
    CategoryID  int64           `json:"categoryId,omitempty"`
    Images      []string        `json:"images"`
    Variants    VariantsData    `json:"variants"`
    Reviews     []Review        `json:"reviews"`
    Ratings     RatingsData     `json:"ratings"`
    StoreInfo   StoreInfo       `json:"storeInfo"`
    Specs       []Specification `json:"specs"`
    Shipping    []ShippingInfo  `json:"shipping"`
    Description string          `json:"description"`
    Currency    CurrencyInfo    `json:"currencyInfo"`
    Quantity    QuantityInfo    `json:"quantity"`

    // Metadata
    ScrapedAt   time.Time       `json:"scrapedAt"`
    SourceURL   string          `json:"sourceUrl"`
}

type VariantsData struct {
    Options []VariantOption `json:"options"`
    Prices  []SKUPrice      `json:"prices"`
}

type Review struct {
    ID         string    `json:"id"`
    Rating     int       `json:"rating"`
    Content    string    `json:"content"`
    AuthorName string    `json:"authorName"`
    Country    string    `json:"country"`
    Date       time.Time `json:"date"`
    Photos     []string  `json:"photos"`
    Variant    string    `json:"variant,omitempty"`
}

type RatingsData struct {
    AverageStar     float64           `json:"averageStar"`
    TotalCount      int64             `json:"totalCount"`
    FiveStarCount   int64             `json:"fiveStarCount"`
    FourStarCount   int64             `json:"fourStarCount"`
    ThreeStarCount  int64             `json:"threeStarCount"`
    TwoStarCount    int64             `json:"twoStarCount"`
    OneStarCount    int64             `json:"oneStarCount"`
}
```

### Configuration

Add to `config/config.yaml`:

```yaml
scraper:
  aliexpress:
    # Browser settings
    browser:
      headless: false
      debug_port: 9222
      user_data_dir: ~/.cly/scraper/chrome
      timeout: 60s
      wait_time: 15s

    # Extraction settings
    reviews_count: 20
    filter_reviews_by: "all"  # all, 5stars, 4stars, etc.

    # Output settings
    output_mode: "single"  # single, per-url
    output_dir: "./scraped"
    output_format: "json"

    # Rate limiting
    delay_between_products: 5s
    max_retries: 3
```

## Implementation Plan

### Phase 1: Browser Foundation
1. Create `modules/scraper` directory structure
2. Implement `BrowserController` with chromedp
3. Add `cly scraper browser` command to launch persistent browser
4. Test CAPTCHA detection and manual solving workflow

### Phase 2: Basic Scraper
1. Implement input parser (URL/file parsing)
2. Create basic `Scraper` orchestration
3. Implement 3 core extractors: basicInfo, images, variants
4. Add simple console output (no TUI yet)
5. Test with single product

### Phase 3: All Extractors
1. Implement remaining extractors (reviews, ratings, store, specs, shipping, description, currency, quantity)
2. Add parallel extraction with goroutines
3. Handle missing data gracefully (don't fail entire scrape)
4. Test with various product types

### Phase 4: Output System
1. Implement output writer with streaming JSON
2. Add per-URL output mode
3. Add output directory management
4. Test with batch scraping (10+ products)

### Phase 5: TUI & Polish
1. Implement bubbletea progress UI
2. Add color-coded status messages
3. Show real-time stats (completed/failed/current)
4. Add graceful shutdown (Ctrl+C handling)

### Phase 6: Configuration & Testing
1. Add config file support
2. Implement retry logic for failed scrapes
3. Add rate limiting between products
4. Integration tests with real AliExpress pages (manual CAPTCHA)
5. Error handling improvements

## Dependencies

```go
// Browser automation
github.com/chromedp/chromedp

// TUI
github.com/charmbracelet/bubbletea
github.com/charmbracelet/lipgloss

// Config
github.com/spf13/viper  // Already in cly

// CLI
github.com/spf13/cobra  // Already in cly

// Logging
github.com/rs/zerolog   // Already in cly
```

## Key Decisions

### Why Chromedp over Puppeteer?
- **Native Go**: No Node.js dependency
- **Performance**: Lower memory footprint
- **Integration**: Better fits cly's Go architecture
- **Control**: Direct Chrome DevTools Protocol access

### Streaming JSON vs In-Memory
- **Streaming**: Handle large batches without memory issues
- **Append mode**: Real-time output (can tail -f the file)
- **Trade-off**: Slightly more complex implementation

### Persistent Browser Strategy
- **User solves CAPTCHA once**: In separate `cly scraper browser` command
- **Reuse session**: All subsequent scrapes use same browser
- **User-visible browser**: Easier debugging, manual intervention possible

### Error Handling Philosophy
- **Partial success**: If 8/10 extractors succeed, still output product (with warnings)
- **Continue on failure**: One failed product doesn't stop batch
- **Detailed errors**: Log which extractor failed and why

## Testing Strategy

### Unit Tests
- Input parser with various formats
- ProductID extraction from URLs
- Output writer (mock file system)
- Individual extractors (mock chromedp context)

### Integration Tests
- Requires manual CAPTCHA solving
- Test with known product IDs
- Validate output JSON structure
- Test with different product types (variants/no variants, reviews/no reviews)

### Manual Testing Checklist
1. Launch persistent browser: `cly scraper browser`
2. Solve CAPTCHA on AliExpress homepage
3. Scrape single product: `cly scraper aliexpress --url ID`
4. Verify JSON output matches expected schema
5. Scrape batch from file: `cly scraper aliexpress -f products.txt`
6. Verify streaming output (tail -f results.json)
7. Test per-URL output mode
8. Test with invalid IDs (graceful failure)
9. Test network interruption recovery
10. Test Ctrl+C graceful shutdown

## Migration from Node.js Version

**Reference**: Current Node.js implementation in `.references/aliexpress-scraper/`

### Direct Ports
- **DOM selectors**: Translate JS selectors to chromedp equivalents
- **Image URL conversion**: Same logic (_220x220.jpg_.avif → _960x960q75.jpg)
- **Wait strategy**: Same 15s + networkidle approach
- **Output format**: Maintain exact JSON structure for compatibility

### Improvements
- **Performance**: Go's concurrency (goroutines) for parallel extraction
- **Memory**: Streaming output instead of loading all in memory
- **Integration**: Native cly command (no subprocess)
- **Distribution**: Single binary (no npm install)

### Breaking Changes
- **No PDF generation** (initially - can add later with Go PDF library)
- **CLI interface different** (cly-style flags, not Node.js script)

## Future Enhancements

### v2 Features
- **PDF catalog generation**: Use `gofpdf` or `pdfcpu`
- **Multiple sites**: Extend to Amazon, eBay, etc.
- **Proxy support**: Rotate proxies to avoid rate limiting
- **CAPTCHA service integration**: 2Captcha, Anti-Captcha APIs

### Advanced
- **Distributed scraping**: Multiple browsers in parallel
- **Database storage**: PostgreSQL instead of JSON
- **API mode**: Run as HTTP server, scrape on-demand
- **Scheduled scraping**: Cron-like product monitoring

## Open Questions

1. **CAPTCHA solving service integration?**
   - Pro: Fully automated scraping
   - Con: Costs money, adds complexity
   - Decision: Manual solving for MVP, optional service later

2. **Headless browser support?**
   - Pro: Faster, no UI overhead
   - Con: CAPTCHA impossible to solve manually
   - Decision: Headless=false by default, make configurable

3. **Output format alternatives?**
   - JSON is standard, but could add CSV, SQLite, etc.
   - Decision: JSON only for MVP

4. **Rate limiting strategy?**
   - Too fast → more CAPTCHAs
   - Too slow → user frustration
   - Decision: Configurable delay (default 5s), user can adjust

## Success Metrics

- ✅ Can scrape 10 products without CAPTCHA after initial solve
- ✅ Extraction success rate >90% for core fields (title, images, price)
- ✅ Memory usage <200MB for 100-product batch
- ✅ Output JSON validates against schema
- ✅ TUI updates in real-time, responsive to Ctrl+C
- ✅ Single binary distribution, no external dependencies
