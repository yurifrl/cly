package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

// Controller manages chromedp browser lifecycle for web scraping
type Controller struct {
	// Contexts
	allocCtx context.Context
	ctx      context.Context
	cancel   context.CancelFunc

	// Configuration
	debugPort   int
	headless    bool
	userDataDir string
	timeout     time.Duration
	waitTime    time.Duration
	browserURL  string // For connecting to existing browser
}

// Options configures the browser controller
type Options struct {
	DebugPort   int
	Headless    bool
	UserDataDir string
	Timeout     time.Duration
	WaitTime    time.Duration
	BrowserURL  string
}

// NewController creates a new browser controller with given options
func NewController(opts Options) *Controller {
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.WaitTime == 0 {
		opts.WaitTime = 15 * time.Second
	}
	if opts.DebugPort == 0 {
		opts.DebugPort = 9222
	}

	return &Controller{
		debugPort:   opts.DebugPort,
		headless:    opts.Headless,
		userDataDir: opts.UserDataDir,
		timeout:     opts.Timeout,
		waitTime:    opts.WaitTime,
		browserURL:  opts.BrowserURL,
	}
}

// Launch starts a new Chrome instance or connects to existing one
func (c *Controller) Launch(ctx context.Context) error {
	if c.browserURL != "" {
		return c.Connect(ctx, c.browserURL)
	}

	// Setup allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", c.headless),
		chromedp.Flag("disable-gpu", false),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// Add user data dir if specified
	if c.userDataDir != "" {
		opts = append(opts, chromedp.UserDataDir(c.userDataDir))
	}

	// Add remote debugging port
	opts = append(opts, chromedp.Flag("remote-debugging-port", fmt.Sprintf("%d", c.debugPort)))

	// Create allocator context
	c.allocCtx, c.cancel = chromedp.NewExecAllocator(ctx, opts...)

	// Create browser context with timeout
	c.ctx, _ = chromedp.NewContext(c.allocCtx)
	c.ctx, _ = context.WithTimeout(c.ctx, c.timeout)

	// Initialize browser
	return chromedp.Run(c.ctx)
}

// Connect connects to an existing Chrome instance via remote debugging
func (c *Controller) Connect(ctx context.Context, url string) error {
	// Connect to remote allocator
	c.allocCtx, c.cancel = chromedp.NewRemoteAllocator(ctx, url)

	// Create browser context
	c.ctx, _ = chromedp.NewContext(c.allocCtx)
	c.ctx, _ = context.WithTimeout(c.ctx, c.timeout)

	return chromedp.Run(c.ctx)
}

// NavigateToProduct navigates to an AliExpress product page
func (c *Controller) NavigateToProduct(productID string) error {
	if c.ctx == nil {
		return fmt.Errorf("browser not launched, call Launch() first")
	}

	url := fmt.Sprintf("https://www.aliexpress.com/item/%s.html", productID)

	// Navigate and wait for load
	err := chromedp.Run(c.ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(c.waitTime), // Wait for JavaScript execution
	)

	return err
}

// WaitForLoad waits for page to fully load
func (c *Controller) WaitForLoad() error {
	// Additional wait after network idle
	return chromedp.Run(c.ctx, chromedp.Sleep(c.waitTime))
}

// GetContext returns the chromedp context for executing actions
func (c *Controller) GetContext() context.Context {
	return c.ctx
}

// Close closes the browser
func (c *Controller) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}
