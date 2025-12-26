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


# Expected output

https://github.com/sudheer-ranga/aliexpress-product-scraper?tab=readme-ov-file#sample-json-response

```
{
  "title": "Belts Famous Brand Belt Men Mens Belts Quality Genuine Luxury Leather Belt For Men Belt Male Strap Male Metal Automatic Buckle",
  "categoryId": 200000298,
  "productId": 1005005167379524,
  "quantity": {
    "total": 2915,
    "available": 2915
  },
  "description": "<div class=\"detailmodule_image\"><span style=\"font-size:0px\">Belts Famous Brand Belt Men Mens belts Quality Genuine Luxury Leather belt for Men Belt male Strap Male Metal Automatic Buckle</span><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A67e7e00c22a34c03a2fd6abd8335ca94o.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A4eb2fb78352e4a75a5d5af99291bf3c0O.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A724b1b680ea144ec97177a675436afe0X.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A3a884f6c417443f294445df4caabe2d8g.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A66887b28887f4a9bb7cd413c87871e4b3.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A6953b334519346ea8a5e70b1bc7e4d30x.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A43c07db96f144772aab4b867348c5042K.jpg\" slate-data-type=\"image\"></div><div class=\"detailmodule_image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A8a5a2cc0cbff47f19839818817af211df.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Abee28c446a5b4a5cb2e5e37597a0950eZ.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A092cc17e08c74e5d92658ebbb01cb1a6W.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A89f6efe8339944e1a6981a3660575c01l.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Aca2a75bbd4514c9ea128ebf6eedbf8f6K.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Aed711ad0c4e844c7ba50a37ba2a2f3cbt.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Ab5ba00118c274f1ebe3ad2a69b1c3ce8U.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A00ca9a7d4d0b4c8bb76f0554bc1d6d85x.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Aeac414ae848a4c6b8ca23c35651363a9Q.jpg\" slate-data-type=\"image\"></div><div class=\"detailmodule_image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A3babd7dd86594ffa8ea7b913e561e003C.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A9f119a0b783746539e75d123e9aed02br.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A57418fd34de9439c99c81c36291b3c00P.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A54705ead34374411bd209b1da3683111p.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Aebecf8b1b81d47e3b0e95935211b65a56.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Acd3c6355f8dd443ca4ce764072497b95A.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A41516c4213b2410db8f0e7326ebe0bb27.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A6dbe49a93a5843049fd6bfa90868b6c0b.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A6fac50b7b99c487890eddb1b55862009z.jpg\" slate-data-type=\"image\"><img style=\"margin-bottom:10px\" class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Ae37039e03add42e691707f66bcf42137m.jpg\" slate-data-type=\"image\"></div><div class=\"detailmodule_image\"><img class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Aa8adab71a10a4d27ab4c6826970d0319L.jpg\" slate-data-type=\"image\"><img class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/Ade85a37af1dd41b8b58061d37ccfb2e4c.jpg\" slate-data-type=\"image\"><img class=\"detail-desc-decorate-image\" src=\"https://ae01.alicdn.com/kf/A7af7c1061767464081a8d458c9755b8aj.jpg\" slate-data-type=\"image\"></div><p><br></p>\n<script>window.adminAccountId=2673771248;</script>\n",
  "orders": "5,000+",
  "storeInfo": {
    "name": "SaengQ Belt Store",
    "logo": "https://ae01.alicdn.com/kf/S7f770946de0d4e8c80e7d06d15f6009d7.png",
    "companyId": 2673771248,
    "storeNumber": 1102598020,
    "isTopRated": false,
    "hasPayPalAccount": false,
    "ratingCount": 5267,
    "rating": "96.1"
  },
  "ratings": {
    "totalStar": 5,
    "averageStar": "4.7",
    "totalStartCount": 1664,
    "fiveStarCount": 1358,
    "fourStarCount": 221,
    "threeStarCount": 42,
    "twoStarCount": 15,
    "oneStarCount": 28
  },
  "images": [
    "https://ae01.alicdn.com/kf/S06fcac1cfaeb467b94a00e5fadcceebb3/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg",
    "https://ae01.alicdn.com/kf/Sbf8be40921594b3d9143e504403386c6I/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg",
    "https://ae01.alicdn.com/kf/S98ab69e7fab24e6487df043a14e094eel/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg",
    "https://ae01.alicdn.com/kf/S7da80d421c2047aab0f642ea76b435a5U/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg",
    "https://ae01.alicdn.com/kf/Sc40414c08e6f449a8e55a17f2e8700efs/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg",
    "https://ae01.alicdn.com/kf/S7d6212feb2bd403a908e1b2b128f562bD/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg"
  ],
  "reviews": [
    {
      "anonymous": false,
      "name": "o***o",
      "displayName": "Anna Collier",
      "gender": "female",
      "country": "FR",
      "rating": 2,
      "info": "Color:NE309 Belt Length:115CM ",
      "date": "06 Nov 2023",
      "content": "la boucle s'est détachée seule au bout de quelques semaines d'utilisation, logique puisque celle-ci n'est retenue que par deux petits clous de mediocre qualité.",
      "photos": [
        "https://ae01.alicdn.com/kf/Aeafa83229d3447dcaae2060bdfcf92bdM.jpg",
        "https://ae01.alicdn.com/kf/A4976cf60c06d4033b37aee6bbac673efp.jpg",
        "https://ae01.alicdn.com/kf/A7082aa4dbdf74ef6a879161d576d4243w.jpg",
        "https://ae01.alicdn.com/kf/Ab0b320321ed845c3a56156c436b7a7fe5.jpg"
      ],
      "thumbnails": [
        "https://ae01.alicdn.com/kf/Aeafa83229d3447dcaae2060bdfcf92bdM.jpg_220x220.jpg",
        "https://ae01.alicdn.com/kf/A4976cf60c06d4033b37aee6bbac673efp.jpg_220x220.jpg",
        "https://ae01.alicdn.com/kf/A7082aa4dbdf74ef6a879161d576d4243w.jpg_220x220.jpg",
        "https://ae01.alicdn.com/kf/Ab0b320321ed845c3a56156c436b7a7fe5.jpg_220x220.jpg"
      ]
    },
    {
      "anonymous": true,
      "name": "AliExpress Shopper",
      "displayName": "Christy Willms",
      "gender": "female",
      "country": "NG",
      "rating": 2,
      "info": "Color:NE304 silvery Belt Length:115CM ",
      "date": "17 Oct 2023",
      "content": "I received my belt shattered, so sad about it ",
      "photos": [
        "https://ae01.alicdn.com/kf/A8ef4ca261ec54a748e828447a2aaab13K.jpg",
        "https://ae01.alicdn.com/kf/A34003a19cf4249b58f80f7c380497355d.jpg",
        "https://ae01.alicdn.com/kf/Af3ad1dfcf3184128a212ec1a324c324fv.jpg"
      ],
      "thumbnails": [
        "https://ae01.alicdn.com/kf/A8ef4ca261ec54a748e828447a2aaab13K.jpg_220x220.jpg",
        "https://ae01.alicdn.com/kf/A34003a19cf4249b58f80f7c380497355d.jpg_220x220.jpg",
        "https://ae01.alicdn.com/kf/Af3ad1dfcf3184128a212ec1a324c324fv.jpg_220x220.jpg"
      ]
    },
    {
      "anonymous": false,
      "name": "s***r",
      "displayName": "Helen VonRueden",
      "gender": "female",
      "country": "KR",
      "rating": 2,
      "info": "Color:NE305 silvery Belt Length:115CM ",
      "date": "18 Nov 2023",
      "content": "버클 검정부분이 떠서 본드로 붙여서 사용해야되나?싶음",
      "photos": [
        "https://ae01.alicdn.com/kf/Aeb23160bc4b646dd8bea7332c0c4294bn.jpg",
        "https://ae01.alicdn.com/kf/A9ad776ee8955410a9f4053c6b7ed4f2f9.jpg"
      ],
      "thumbnails": [
        "https://ae01.alicdn.com/kf/Aeb23160bc4b646dd8bea7332c0c4294bn.jpg_220x220.jpg",
        "https://ae01.alicdn.com/kf/A9ad776ee8955410a9f4053c6b7ed4f2f9.jpg_220x220.jpg"
      ]
    },
    {
      "anonymous": false,
      "name": "L***D",
      "displayName": "Dr. Lewis Baumbach",
      "gender": "male",
      "country": "KR",
      "rating": 2,
      "info": "Color:NE701 Belt Length:120cm ",
      "date": "02 Dec 2023",
      "content": "절대",
      "photos": [],
      "thumbnails": []
    },
    {
      "anonymous": false,
      "name": "a***a",
      "displayName": "Diana Mertz",
      "gender": "female",
      "country": "CH",
      "rating": 2,
      "info": "Color:NE305 silvery Belt Length:130cm ",
      "date": "26 Sep 2023",
      "content": "Trop grande",
      "photos": [],
      "thumbnails": []
    }
  ],
  "variants": {
    "options": [
      {
        "id": 14,
        "name": "Color",
        "values": [
          {
            "id": 29,
            "name": "WHITE",
            "displayName": "NE336-silvery",
            "image": "https://ae01.alicdn.com/kf/S0b481aeb168146c0bcfc80622a14c3a31/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg_640x640.jpg"
          },
          {
            "id": 496,
            "name": "PURPLE",
            "displayName": "NE701",
            "image": "https://ae01.alicdn.com/kf/A881e74e49d4a466393de16a414955943n/Belts-Famous-Brand-Belt-Men-Mens-Belts-Quality-Genuine-Luxury-Leather-Belt-For-Men-Belt-Male.jpg_640x640.jpg"
          }
        ]
      },
      {
        "id": 200000858,
        "name": "Belt Length",
        "values": [
          {
            "id": 201447587,
            "name": "115CM",
            "displayName": "115CM"
          },
          {
            "id": 200006543,
            "name": "120cm",
            "displayName": "120cm"
          },
          {
            "id": 201447589,
            "name": "130cm",
            "displayName": "130cm"
          }
        ]
      }
    ],
    "prices": [
      {
        "skuId": 12000031946221040,
        "optionValueIds": "193,201447589",
        "availableQuantity": 169,
        "originalPrice": {
          "currency": "GBP",
          "formatedAmount": "￡16.44",
          "value": 16.44
        },
        "salePrice": {
          "currency": "GBP",
          "formatedAmount": "￡0.40",
          "value": 0.4
        }
      },
      {
        "skuId": 12000031946221038,
        "optionValueIds": "193,200006543",
        "availableQuantity": 180,
        "originalPrice": {
          "currency": "GBP",
          "formatedAmount": "￡16.08",
          "value": 16.08
        },
        "salePrice": {
          "currency": "GBP",
          "formatedAmount": "￡0.40",
          "value": 0.4
        }
      },
      {
        "skuId": 12000034561692836,
        "optionValueIds": "1254,201447587",
        "availableQuantity": 12,
        "originalPrice": {
          "currency": "GBP",
          "formatedAmount": "￡15.58",
          "value": 15.58
        },
        "salePrice": {
          "currency": "GBP",
          "formatedAmount": "￡0.40",
          "value": 0.4
        }
      }
    ]
  },
  "specs": [
    {
      "attrValue": "China (Mainland)",
      "attrName": "Place Of Origin"
    },
    {
      "attrValue": "3.5cm",
      "attrName": "Belt Width"
    },
    {
      "attrValue": "7cm",
      "attrName": "Buckle Length"
    },
    {
      "attrValue": "4cm",
      "attrName": "Buckle Width"
    },
    {
      "attrValue": "Plaid",
      "attrName": "Pattern Type"
    },
    {
      "attrValue": "Casual",
      "attrName": "Style"
    },
    {
      "attrValue": "Metal,Cowskin,Faux Leather",
      "attrName": "Belts Material"
    },
    {
      "attrValue": "Adult",
      "attrName": "Department Name"
    },
    {
      "attrValue": "MEN",
      "attrName": "Gender"
    },
    {
      "attrValue": "saengQ",
      "attrName": "Brand Name"
    },
    {
      "attrValue": "Mainland China",
      "attrName": "Origin"
    },
    {
      "attrValue": "Zhejiang",
      "attrName": "CN"
    },
    {
      "attrValue": "BELTS",
      "attrName": "Item Type"
    }
  ],
  "currencyInfo": {
    "baseCurrencyCode": "CNY",
    "enableTransaction": true,
    "currencySymbol": "￡",
    "symbolFront": false,
    "currencyRate": 0.1139,
    "baseSymbolFront": false,
    "currencyCode": "GBP",
    "baseCurrencySymbol": "CN￥",
    "multiCurrency": true
  },
  "originalPrice": {
    "min": {
      "currency": "GBP",
      "formatedAmount": "￡14.99",
      "value": 14.99
    },
    "max": {
      "currency": "GBP",
      "formatedAmount": "￡17.85",
      "value": 17.85
    }
  },
  "salePrice": {
    "min": {
      "currency": "GBP",
      "formatedAmount": "￡0.40",
      "value": 0.4
    },
    "max": {
      "currency": "GBP",
      "formatedAmount": "￡0.40",
      "value": 0.4
    }
  },
  "shipping": [
    {
      "deliveryProviderName": "Aliexpress Selection Premium shipping",
      "tracking": "invisible",
      "provider": "cainiao",
      "company": "Aliexpress Selection Premium shipping",
      "deliveryInfo": {
        "min": 5,
        "max": 5
      },
      "shippingInfo": {
        "from": "China",
        "fromCode": "CN",
        "to": "United Kingdom",
        "toCode": "UK",
        "fees": "￡1.61",
        "displayAmount": 1.61,
        "displayCurrency": "GBP"
      },
      "warehouseType": "own_warehouse"
    }
  ]
}
```
