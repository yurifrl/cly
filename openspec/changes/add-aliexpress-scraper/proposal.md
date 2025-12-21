# Proposal: Add AliExpress Scraper Module

## Summary

Add `cly scraper aliexpress` command to scrape AliExpress product data using persistent browser automation (chromedp). Provides batch processing, flexible input formats, and JSON output streaming.

## Problem

The project requires AliExpress product scraping capability to collect product data programmatically. A Node.js reference implementation exists (`.references/aliexpress-scraper/`) but needs Go-native implementation integrated with CLY's modular architecture.

## Solution

Implement a new `scraper` module with:
- Browser automation using chromedp (Go-native, no Node.js dependency)
- Persistent browser session with manual CAPTCHA solving
- Parallel data extraction using goroutines
- Streaming JSON output for memory efficiency
- Bubbletea TUI for progress tracking
- Flexible input: single URLs, CSV/JSON/YAML files, or IDs

## Scope

**In Scope:**
- Browser controller (chromedp) with persistent session support
- Input parser (URLs, files: TXT, CSV, JSON, YAML)
- Core extractors: basic info, images, variants, reviews, ratings, store, specs, shipping, description, currency, quantity
- Output writer with streaming JSON
- TUI progress display (Bubbletea)
- Config file integration (Viper)
- Browser launcher command (`cly scraper browser`)
- AliExpress scraper command (`cly scraper aliexpress`)

**Out of Scope:**
- PDF catalog generation (future enhancement)
- Multiple site support (Amazon, eBay) (future enhancement)
- Proxy rotation (future enhancement)
- CAPTCHA service integration (manual solving only for MVP)
- Distributed scraping (future enhancement)

## Risks

1. **CAPTCHA detection**: AliExpress may change CAPTCHA strategy
   - *Mitigation*: Persistent browser with user-data-dir for session reuse
2. **DOM structure changes**: AliExpress updates may break extractors
   - *Mitigation*: Reference Node.js implementation for DOM patterns, comprehensive error handling
3. **Memory usage**: Large batches could exhaust memory
   - *Mitigation*: Streaming JSON output, configurable batch limits

## Dependencies

**New Go dependencies:**
- `github.com/chromedp/chromedp` - Browser automation

**Existing CLY dependencies:**
- `github.com/charmbracelet/bubbletea` - TUI (already present)
- `github.com/charmbracelet/lipgloss` - Styling (already present)
- `github.com/spf13/cobra` - CLI (already present)
- `github.com/spf13/viper` - Config (already present)

**No breaking changes** to existing modules or APIs.

## Alternatives Considered

1. **Puppeteer/Playwright via Node.js subprocess**
   - Rejected: Adds Node.js runtime dependency, complicates distribution
2. **Headless chromedp**
   - Rejected for MVP: Can't solve CAPTCHA manually, though will be configurable
3. **HTTP-only scraping (no browser)**
   - Rejected: AliExpress requires JavaScript rendering, high CAPTCHA rate

## Success Criteria

- Scrape 10 products without CAPTCHA after initial browser setup
- Extraction success rate >90% for core fields (title, images, price)
- Memory usage <200MB for 100-product batch
- Output JSON validates against expected schema
- TUI updates in real-time, responsive to Ctrl+C
- Single binary distribution (no external dependencies at runtime)
