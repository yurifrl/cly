# Spec: AliExpress Scraper

## ADDED Requirements

### Requirement: Product Data Extraction

The system SHALL extract complete product data from AliExpress product pages.

#### Scenario: Extract basic info

```
GIVEN a browser has loaded a product page
WHEN the basic info extractor runs
THEN it extracts productId, title, and categoryId
AND returns structured data matching ProductData.BasicInfo
```

#### Scenario: Extract images

```
GIVEN a browser has loaded a product page
WHEN the images extractor runs
THEN it extracts all gallery image URLs
AND converts thumbnail URLs to high-resolution (960x960) URLs
AND returns array of strings
```

#### Scenario: Extract variants

```
GIVEN a browser has loaded a product page with variants
WHEN the variants extractor runs
THEN it extracts variant options (color, size, etc.)
AND extracts SKU pricing for each variant combination
AND returns structured VariantsData with options and prices
```

#### Scenario: Extract reviews

```
GIVEN config specifies reviewsCount = 20
WHEN the reviews extractor runs
THEN it fetches 20 reviews via AliExpress API endpoint
AND extracts review content, rating, author, country, date, photos
AND returns array of Review structs
```

#### Scenario: Extract ratings

```
GIVEN a browser has loaded a product page
WHEN the ratings extractor runs
THEN it extracts average star rating and total count
AND extracts breakdown by star (5-star count, 4-star count, etc.)
AND returns RatingsData struct
```

### Requirement: Input Parsing

The system SHALL parse product IDs from multiple input formats.

#### Scenario: Parse single product ID

```
GIVEN command flag --url "1005003618976317"
WHEN the input parser processes the flag
THEN it returns ["1005003618976317"]
```

#### Scenario: Parse full URL

```
GIVEN command flag --url "https://www.aliexpress.com/item/1005003618976317.html"
WHEN the input parser processes the flag
THEN it extracts the product ID "1005003618976317"
AND returns ["1005003618976317"]
```

#### Scenario: Parse comma-separated IDs

```
GIVEN command flag --url "1005003618976317,1005010081760632"
WHEN the input parser processes the flag
THEN it returns ["1005003618976317", "1005010081760632"]
```

#### Scenario: Parse TXT file (newline-separated)

```
GIVEN a file products.txt containing:
  1005003618976317
  1005010081760632
WHEN the input parser reads the file
THEN it returns ["1005003618976317", "1005010081760632"]
```

#### Scenario: Parse CSV file

```
GIVEN a file products.csv containing:
  product_id,name
  1005003618976317,Belt
  1005010081760632,Watch
WHEN the input parser reads the file
THEN it detects the product_id column
AND returns ["1005003618976317", "1005010081760632"]
```

#### Scenario: Parse JSON array

```
GIVEN a file products.json containing:
  ["1005003618976317", "1005010081760632"]
WHEN the input parser reads the file
THEN it returns ["1005003618976317", "1005010081760632"]
```

### Requirement: Sequential Extraction

The system SHALL extract product data using sequential extractor calls to avoid browser tab conflicts.

#### Scenario: Run extractors sequentially

```
GIVEN 10 extractors are registered (basic, images, variants, etc.)
WHEN ScrapeProduct() is called
THEN extractors run one at a time in sequential order
AND each extractor completes before the next starts
AND results are aggregated into ProductData struct
BECAUSE a single browser tab cannot handle concurrent chromedp operations
```

#### Scenario: Handle partial failures

```
GIVEN the reviews extractor fails (network error)
AND other extractors succeed
WHEN ScrapeProduct() is called
THEN the ProductData contains all successful extractions
AND the reviews field is empty
AND a warning is logged: "reviews extractor failed: network error"
AND remaining extractors continue to run
```

### Requirement: Output JSON

The system SHALL write scraped product data to JSON files.

#### Scenario: Single file output (default)

```
GIVEN command flag --output "results.json"
WHEN products are scraped
THEN all products append to results.json as a JSON array
USING streaming write (not loading entire file into memory)
```

#### Scenario: Per-URL output

```
GIVEN command flag --output-per-url
WHEN product "1005003618976317" is scraped
THEN a file product_1005003618976317.json is created
AND contains a single ProductData JSON object
```

#### Scenario: Custom output directory

```
GIVEN command flag --output-dir "./scraped"
WHEN products are scraped
THEN output files are created in ./scraped/
AND the directory is created if it doesn't exist
```

### Requirement: Scraper Command

The system SHALL provide a CLI command to scrape AliExpress products.

#### Scenario: Scrape single product

```
GIVEN command "cly scraper aliexpress --url 1005003618976317"
WHEN the command executes
THEN a browser connects to port 9222 (or launches new)
AND navigates to the product page
AND extracts all product data
AND writes to product_1005003618976317.json
AND displays completion message
```

#### Scenario: Scrape batch from file

```
GIVEN command "cly scraper aliexpress -f products.txt"
AND products.txt contains 10 product IDs
WHEN the command executes
THEN it scrapes each product sequentially
AND writes to a single JSON file (default)
AND displays progress via TUI
```

#### Scenario: Connect to custom browser URL

```
GIVEN command "cly scraper aliexpress --url ID --browser-url http://192.168.1.100:9222"
WHEN the command executes
THEN it connects to the browser at the specified URL
AND proceeds with scraping
```

### Requirement: Rate Limiting

The system SHALL implement rate limiting to avoid excessive request rates.

#### Scenario: Delay between products

```
GIVEN config specifies scraper.aliexpress.delay_between_products = 5s
WHEN scraping multiple products
THEN the system waits 5 seconds between each product
```

#### Scenario: Configurable delay

```
GIVEN command flag --delay 10s
WHEN scraping multiple products
THEN the system waits 10 seconds between each product
OVERRIDING config file setting
```

### Requirement: Error Handling

The system SHALL handle errors gracefully and continue batch processing.

#### Scenario: Product not found

```
GIVEN a product ID that doesn't exist
WHEN scraping the product
THEN an error is logged: "Product not found: <ID>"
AND the scraper continues with next product
AND the TUI shows the product as "failed"
```

#### Scenario: CAPTCHA detected during scrape

```
GIVEN CAPTCHA appears during scraping
WHEN DetectCAPTCHA() returns true
THEN scraping pauses
AND a message displays: "CAPTCHA detected. Solve manually in browser, then press Enter"
AND scraping resumes after user input
```

#### Scenario: Maximum retries exceeded

```
GIVEN an extractor fails 3 times (max_retries = 3)
WHEN scraping a product
THEN the extractor is marked as failed
AND scraping continues with other extractors
AND a warning is logged
```

### Requirement: JSON Schema Validation

The system SHALL produce output JSON that matches the expected schema structure with correct keys and types.

#### Scenario: Output matches reference schema

```
GIVEN a product has been scraped successfully
WHEN the JSON output is generated
THEN it contains all required top-level keys: title, categoryId, productId, quantity, description, orders, storeInfo, ratings, images, reviews, variants, specs, currencyInfo, originalPrice, salePrice, shipping
AND each key has the correct type (string, number, object, array)
AND nested objects match the reference structure
```

#### Scenario: Validate storeInfo structure

```
GIVEN the storeInfo extractor succeeds
WHEN the JSON output is generated
THEN storeInfo contains: name, logo, companyId, storeNumber, isTopRated, hasPayPalAccount, ratingCount, rating
AND companyId and storeNumber are integers
AND isTopRated and hasPayPalAccount are booleans
AND rating is a string representation of a number
```

#### Scenario: Validate ratings structure

```
GIVEN the ratings extractor succeeds
WHEN the JSON output is generated
THEN ratings contains: totalStar, averageStar, totalStartCount, fiveStarCount, fourStarCount, threeStarCount, twoStarCount, oneStarCount
AND all star counts are integers
AND averageStar is a string representation of a number
```

#### Scenario: Validate review structure

```
GIVEN the reviews extractor fetches reviews
WHEN the JSON output is generated
THEN each review contains: anonymous, name, displayName, gender, country, rating, info, date, content, photos, thumbnails
AND anonymous is boolean
AND rating is integer (1-5)
AND photos and thumbnails are arrays of strings
```

#### Scenario: Validate variants structure

```
GIVEN the variants extractor succeeds
WHEN the JSON output is generated
THEN variants contains: options (array) and prices (array)
AND each option contains: id, name, values (array)
AND each option value contains: id, name, displayName, optional image
AND each price contains: skuId, optionValueIds, availableQuantity, originalPrice, salePrice
AND each price object contains: currency, formatedAmount, value
```

#### Scenario: Schema validation test

```
GIVEN a schema definition matching the reference JSON structure
WHEN integration tests run
THEN a test validates scraped output against the schema
AND fails if any required key is missing
AND fails if any value has wrong type
AND warns if optional keys are missing
```

### Requirement: Configuration Integration

The system SHALL integrate with CLY's Viper-based configuration system.

#### Scenario: Load browser settings from config

```
GIVEN config file contains:
  scraper:
    aliexpress:
      browser:
        debug_port: 9222
        user_data_dir: ~/.cly/scraper/chrome
WHEN the scraper initializes
THEN it uses port 9222 and specified user-data-dir
```

#### Scenario: Override config with environment variables

```
GIVEN environment variable CLY_SCRAPER_ALIEXPRESS_REVIEWS_COUNT=50
WHEN the scraper initializes
THEN it overrides config.reviewsCount with 50
```

#### Scenario: Override config with flags

```
GIVEN config specifies reviewsCount = 20
AND command flag --reviews 100
WHEN the scraper initializes
THEN it uses 100 reviews (flag takes precedence)
```
