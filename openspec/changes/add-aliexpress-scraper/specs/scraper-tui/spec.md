# Spec: Scraper TUI

## ADDED Requirements

### Requirement: Progress Display

The system SHALL display real-time scraping progress using a Bubbletea TUI.

#### Scenario: Initialize progress model

```
GIVEN a batch of 10 products to scrape
WHEN the TUI initializes
THEN it creates a ProgressModel with total = 10
AND all products start in "pending" status
AND the progress bar shows 0%
```

#### Scenario: Update current product

```
GIVEN scraping starts for product "1005003618976317"
WHEN a ProductStarted message is sent to the TUI
THEN the current field updates to "1005003618976317"
AND the product status updates to "scraping"
AND the view displays "⏳ 1005003618976317 (scraping...)"
```

#### Scenario: Complete product

```
GIVEN a product finishes scraping successfully
WHEN a ProductCompleted message is sent to the TUI
THEN the completed count increments by 1
AND the product status updates to "done"
AND the view displays "✓ 1005003618976317 (32.4s)"
AND the progress bar updates
```

#### Scenario: Fail product

```
GIVEN a product fails to scrape
WHEN a ProductFailed message is sent with error "CAPTCHA required"
THEN the failed count increments by 1
AND the product status updates to "failed"
AND the view displays "✗ 1005003618976317 (CAPTCHA required)"
```

### Requirement: Progress Bar

The system SHALL render a progress bar showing scraping completion percentage.

#### Scenario: Progress bar calculation

```
GIVEN 10 total products
AND 6 completed
AND 1 failed
WHEN the View() renders
THEN the progress bar shows 70% (7/10 including failures)
AND displays "Progress: ████████████░░░░░░░░ 7/10 (70%)"
```

### Requirement: Product Status List

The system SHALL display a scrollable list of product statuses.

#### Scenario: Pending status

```
GIVEN a product is pending
WHEN the View() renders
THEN the product line displays "○ 1005008596183124 (pending)"
USING a neutral color (gray)
```

#### Scenario: Scraping status

```
GIVEN a product is currently being scraped
WHEN the View() renders
THEN the product line displays "⏳ 1005003618976317 (scraping...)"
USING an active color (yellow)
```

#### Scenario: Done status

```
GIVEN a product completed successfully in 32.4 seconds
WHEN the View() renders
THEN the product line displays "✓ 1005010081760632 (32.4s)"
USING a success color (green)
```

#### Scenario: Failed status

```
GIVEN a product failed with error "Network timeout"
WHEN the View() renders
THEN the product line displays "✗ 1005008633928425 (Network timeout)"
USING an error color (red)
```

### Requirement: Stats Summary

The system SHALL display summary statistics at the top of the TUI.

#### Scenario: Overall stats

```
GIVEN 20 total products
AND 12 completed
AND 2 failed
AND 1 currently scraping
WHEN the View() renders the header
THEN it displays:
  "Scraping AliExpress Products"
  "Progress: 12/20 completed, 2 failed, 1 in progress"
```

### Requirement: Output Information

The system SHALL display output file information in the TUI.

#### Scenario: Single file output

```
GIVEN output mode is single file
AND output file is "products.json"
AND 12 products scraped
AND file size is 2.4 MB
WHEN the View() renders the footer
THEN it displays "Output: products.json (12 products, 2.4 MB)"
```

#### Scenario: Per-URL output

```
GIVEN output mode is per-URL
AND output directory is "./scraped/"
AND 12 products scraped
WHEN the View() renders the footer
THEN it displays "Output: ./scraped/ (12 files)"
```

### Requirement: Keyboard Controls

The system SHALL support keyboard interactions for TUI navigation.

#### Scenario: Quit with q

```
GIVEN the TUI is running
WHEN the user presses "q"
THEN a quit message is sent
AND the TUI exits gracefully
```

#### Scenario: Quit with Ctrl+C

```
GIVEN the TUI is running
WHEN the user presses Ctrl+C
THEN a quit message is sent
AND the scraper stops current operation
AND partial results are saved
AND the TUI exits
```

### Requirement: Styling Integration

The system SHALL use CLY's shared Lipgloss styles for consistent theming.

#### Scenario: Use theme colors

```
GIVEN pkg/style defines theme colors
WHEN the TUI renders
THEN it uses:
  - style.TitleStyle for header
  - style.SuccessStyle for completed products
  - style.ErrorStyle for failed products
  - style.InfoStyle for pending products
```

### Requirement: TUI Updates

The system SHALL update the TUI in real-time as scraping progresses.

#### Scenario: Update on message

```
GIVEN the TUI is running
WHEN a ProductCompleted message arrives
THEN the Update() method processes it
AND the View() re-renders immediately
AND the terminal displays the updated view
```

#### Scenario: Periodic refresh

```
GIVEN the TUI is running
WHEN no messages arrive for 100ms
THEN a tick message triggers a refresh
AND elapsed times update (e.g., "scraping... 5.2s")
```

### Requirement: Non-Interactive Fallback

The system SHALL provide non-interactive output when TUI is unavailable.

#### Scenario: Non-TTY environment

```
GIVEN stdout is not a TTY (e.g., piped or redirected)
WHEN the scraper runs
THEN it outputs plain text progress:
  "Scraping 1/10: 1005003618976317"
  "✓ 1005003618976317 completed in 32.4s"
  "Scraping 2/10: 1005010081760632"
WITHOUT launching the Bubbletea TUI
```

#### Scenario: Explicit quiet mode

```
GIVEN command flag --quiet
WHEN the scraper runs
THEN it suppresses all TUI output
AND only writes to log file
AND displays final summary at the end
```
