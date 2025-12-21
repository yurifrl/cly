# Spec: Browser Automation

## ADDED Requirements

### Requirement: Browser Controller

The system SHALL provide a browser controller that manages chromedp lifecycle for web scraping operations.

#### Scenario: Launch new browser instance

```
GIVEN the browser controller is initialized with default options
WHEN Launch() is called
THEN a Chrome instance starts with remote debugging on port 9222
AND the user-data-dir is set to ~/.cly/scraper/chrome
AND the browser is visible (not headless)
```

#### Scenario: Connect to existing browser

```
GIVEN a Chrome instance is running with remote debugging on port 9222
WHEN Connect("http://localhost:9222") is called
THEN the controller connects to the existing browser
AND can execute navigation and extraction actions
```

#### Scenario: Navigate to product page

```
GIVEN a connected browser controller
WHEN NavigateToProduct("1005003618976317") is called
THEN the browser navigates to https://www.aliexpress.com/item/1005003618976317.html
AND waits for network idle
AND waits additional 15 seconds for JavaScript execution
```

### Requirement: CAPTCHA Detection

The system SHALL detect when AliExpress presents a CAPTCHA challenge.

#### Scenario: Detect reCAPTCHA iframe

```
GIVEN a browser has navigated to AliExpress
WHEN DetectCAPTCHA() is called
AND the page contains an iframe with src matching "recaptcha"
THEN return true
```

#### Scenario: Detect challenge text

```
GIVEN a browser has navigated to AliExpress
WHEN DetectCAPTCHA() is called
AND the page contains text "Are you a robot?" or "验证"
THEN return true
```

#### Scenario: No CAPTCHA present

```
GIVEN a browser has navigated to AliExpress
WHEN DetectCAPTCHA() is called
AND no CAPTCHA indicators are present
THEN return false
```

### Requirement: Session Persistence

The system SHALL persist browser sessions to avoid repeated CAPTCHA challenges.

#### Scenario: User-data-dir persistence

```
GIVEN the browser controller is launched with user-data-dir set
WHEN the browser is closed and relaunched
THEN cookies and session data are preserved
AND the user does not need to solve CAPTCHA again
```

#### Scenario: Custom user-data-dir location

```
GIVEN config specifies scraper.aliexpress.browser.user_data_dir = "/custom/path"
WHEN the browser launches
THEN it uses /custom/path for session storage
```

### Requirement: Browser Launcher Command

The system SHALL provide a command to launch a persistent browser for manual CAPTCHA solving.

#### Scenario: Launch persistent browser

```
GIVEN the user runs "cly scraper browser"
WHEN the command executes
THEN a Chrome instance launches with debugging enabled
AND navigates to https://www.aliexpress.com
AND displays a message: "Browser ready. Solve CAPTCHA if needed, then run scraper command."
AND keeps the browser running until Ctrl+C
```

#### Scenario: Browser already running

```
GIVEN a browser is already running on port 9222
WHEN the user runs "cly scraper browser"
THEN an error message displays: "Browser already running on port 9222"
AND suggests using --port to specify a different port
```

### Requirement: Wait Strategy

The system SHALL implement a robust wait strategy for AliExpress pages to fully load.

#### Scenario: Network idle wait

```
GIVEN the browser has navigated to a product page
WHEN WaitForLoad() is called
THEN it waits for network idle state (no network activity for 500ms)
```

#### Scenario: Minimum wait time

```
GIVEN the browser has navigated to a product page
WHEN WaitForLoad() is called
THEN it waits at least 15 seconds after network idle
BECAUSE AliExpress renders data progressively via JavaScript
```

#### Scenario: Timeout handling

```
GIVEN the browser has navigated to a product page
WHEN WaitForLoad() is called
AND the page doesn't load within 60 seconds (configurable timeout)
THEN an error is returned: "Page load timeout"
```
