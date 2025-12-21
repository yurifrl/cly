# Tasks: Add AliExpress Scraper Module

## Phase 1: Browser Foundation

1. Create `modules/scraper/` directory structure
   - Validation: Directory exists with correct layout
2. Implement `browser/controller.go` with chromedp integration
   - Validation: Can launch/connect to browser, navigate to URL
3. Implement CAPTCHA detection in `browser/captcha.go`
   - Validation: Unit test detects reCAPTCHA iframe
4. Add session persistence via user-data-dir in `browser/session.go`
   - Validation: Browser retains session across restarts
5. Create `cmd/browser.go` for browser launcher command
   - Validation: `cly scraper browser` launches persistent browser
6. Add chromedp dependency to go.mod
   - Validation: `go mod tidy` completes, `go build` succeeds

## Phase 2: Input Handling

7. Implement `input/parser.go` for URL/ID parsing
   - Validation: Unit tests parse URLs, IDs, comma-separated lists
8. Implement `input/validator.go` for ID validation
   - Validation: Unit tests accept valid IDs, reject invalid
9. Add file parsing for TXT format (newline/comma-separated)
   - Validation: Unit test reads sample TXT file
10. Add file parsing for CSV format
    - Validation: Unit test reads sample CSV file
11. Add file parsing for JSON format
    - Validation: Unit test reads sample JSON file
12. Add file parsing for YAML format
    - Validation: Unit test reads sample YAML file

## Phase 3: Core Extractors

13. Create `aliexpress/types.go` with ProductData struct
    - Validation: Struct matches reference JSON schema
14. Implement `extractors/basic_info.go` (title, ID, category)
    - Validation: Integration test extracts from real page
15. Implement `extractors/images.go` (gallery URLs)
    - Validation: Integration test extracts image URLs
16. Implement `extractors/variants.go` (options, SKUs, prices)
    - Validation: Integration test extracts variant data
17. Implement `extractors/reviews.go` (reviews with photos)
    - Validation: Integration test fetches N reviews
18. Implement `extractors/ratings.go` (star breakdown)
    - Validation: Integration test extracts rating stats
19. Implement `extractors/store.go` (store info)
    - Validation: Integration test extracts store data
20. Implement `extractors/specs.go` (product specifications)
    - Validation: Integration test extracts specs table
21. Implement `extractors/shipping.go` (shipping options)
    - Validation: Integration test extracts shipping data
22. Implement `extractors/description.go` (product description HTML)
    - Validation: Integration test extracts description
23. Implement `extractors/currency.go` (currency info)
    - Validation: Integration test extracts currency data
24. Implement `extractors/quantity.go` (available quantity)
    - Validation: Integration test extracts quantity

## Phase 4: Scraper Orchestration

25. Implement `aliexpress/scraper.go` with Scraper struct
    - Validation: Can initialize with browser controller
26. Implement parallel extraction with goroutines
    - Validation: Unit test aggregates results from multiple extractors
27. Add error handling for partial extraction failures
    - Validation: Unit test continues with missing data
28. Implement ScrapeProduct() method
    - Validation: Integration test scrapes single product
29. Implement ScrapeBatch() method
    - Validation: Integration test scrapes multiple products

## Phase 5: Output System

30. Implement `output/writer.go` with Writer interface
    - Validation: Can write ProductData to file
31. Implement single-file streaming JSON output
    - Validation: Appends to JSON array correctly
32. Implement per-URL output mode
    - Validation: Creates separate file per product
33. Add output directory management
    - Validation: Creates directories as needed
34. Add atomic write operations
    - Validation: No partial writes on interruption

## Phase 6: TUI Progress

35. Implement `tui/progress.go` Bubbletea model
    - Validation: Model initializes with correct state
36. Implement Update() message handling
    - Validation: State updates on messages
37. Implement View() rendering with Lipgloss
    - Validation: Renders progress bar and status
38. Add per-product status tracking
    - Validation: Shows pending/scraping/done/failed
39. Add real-time stats display
    - Validation: Shows completed/failed/current counts
40. Integrate with scraper orchestration
    - Validation: TUI updates during actual scrape

## Phase 7: CLI Commands

41. Create `cmd/root.go` with scraper parent command
    - Validation: `cly scraper --help` shows subcommands
42. Implement `cmd/aliexpress.go` scraper command
    - Validation: `cly scraper aliexpress --help` shows flags
43. Add `--url` flag for single/multiple URLs
    - Validation: Parses comma-separated URLs
44. Add `-f/--file` flag for file input
    - Validation: Reads and parses file
45. Add `--output` flag for output file path
    - Validation: Writes to specified path
46. Add `--output-per-url` flag
    - Validation: Creates separate files
47. Add `--output-dir` flag for custom directory
    - Validation: Uses custom directory
48. Add `--browser-url` flag for remote browser
    - Validation: Connects to specified URL
49. Register scraper module in `cmd/root.go`
    - Validation: `cly scraper` appears in `cly --help`

## Phase 8: Configuration Integration

50. Add scraper config section to `config/config.yaml`
    - Validation: Config file includes scraper settings
51. Integrate Viper config loading in commands
    - Validation: Commands read from config
52. Add environment variable support (CLY_SCRAPER_*)
    - Validation: Env vars override config
53. Add flag precedence (flags > env > config)
    - Validation: Priority order works correctly

## Phase 9: Testing & Polish

54. Create JSON schema definition matching reference output
    - Validation: Schema file exists with all required fields and types
55. Implement schema validation test
    - Validation: Test validates output against schema, fails on missing keys or wrong types
56. Write unit tests for browser controller
    - Validation: Tests pass with mocked chromedp
57. Write unit tests for all extractors
    - Validation: Tests pass with sample HTML
58. Write integration tests for full scrape flow
    - Validation: Scrapes real product successfully AND passes schema validation
59. Add error message improvements
    - Validation: Clear, actionable error messages
60. Add retry logic for failed extractions
    - Validation: Retries N times with backoff
61. Add rate limiting between products
    - Validation: Respects configured delay
62. Add graceful shutdown on Ctrl+C
    - Validation: Cleans up browser, saves partial results
63. Update README with scraper documentation
    - Validation: Instructions complete and accurate
64. Create example config files
    - Validation: Examples work as documented
65. Add reference JSON example to test fixtures
    - Validation: Test data matches blueprint example structure

## Dependencies Between Tasks

**Sequential dependencies:**
- 1 → 2-5 (need directory before implementation)
- 2 → 5 (browser controller before launcher command)
- 13 → 14-24 (types before extractors)
- 25-29 → 30-34 (scraper before output)
- 35-40 → 41-49 (TUI before CLI integration)

**Parallelizable work:**
- 7-12 (input parsers independent)
- 14-24 (extractors independent once types exist)
- 30-34 (output writers independent)
- 54-56 (tests can be written alongside implementation)

**Critical path:**
1 → 2 → 13 → 25 → 41 → 49 (minimum viable scraper)

## Validation Checkpoints

**After Phase 2:** Can parse input from various sources
**After Phase 4:** Can scrape single product, extract all fields
**After Phase 5:** Can output JSON correctly
**After Phase 7:** Complete CLI interface works
**After Phase 9:** Production-ready with tests and documentation
