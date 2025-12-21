package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/modules/scraper/aliexpress"
	"github.com/yurifrl/cly/modules/scraper/aliexpress/extractors"
	"github.com/yurifrl/cly/modules/scraper/browser"
	"github.com/yurifrl/cly/modules/scraper/input"
	"github.com/yurifrl/cly/modules/scraper/output"
	"github.com/yurifrl/cly/modules/scraper/tui"
)

var (
	urlFlag        string
	fileFlag       string
	outputFlag     string
	outputPerURL   bool
	outputDirFlag  string
	browserURLFlag string
	autoStartFlag  bool
)

// AliExpressCmd scrapes AliExpress products
var AliExpressCmd = &cobra.Command{
	Use:   "aliexpress",
	Short: "Scrape AliExpress product data",
	Long: `Scrape product data from AliExpress using browser automation.

Examples:
  # Scrape single product
  cly scraper aliexpress --url 1005003618976317

  # Scrape multiple products
  cly scraper aliexpress --url "1005003618976317,1005010081760632"

  # Scrape from file
  cly scraper aliexpress -f products.txt

  # Custom output
  cly scraper aliexpress --url ID --output results.json`,
	RunE: runAliExpress,
}

func init() {
	ScraperCmd.AddCommand(AliExpressCmd)

	AliExpressCmd.Flags().StringVar(&urlFlag, "url", "", "Product URL or ID (comma-separated for multiple)")
	AliExpressCmd.Flags().StringVarP(&fileFlag, "file", "f", "", "File containing product IDs/URLs")
	AliExpressCmd.Flags().StringVar(&outputFlag, "output", "products.json", "Output file path")
	AliExpressCmd.Flags().BoolVar(&outputPerURL, "output-per-url", false, "Create separate file per product")
	AliExpressCmd.Flags().StringVar(&outputDirFlag, "output-dir", "./scraped", "Output directory")

	AliExpressCmd.Flags().StringVarP(&browserURLFlag, "browser", "b", "", "Connect to existing browser URL")
	AliExpressCmd.Flags().Lookup("browser").NoOptDefVal = "http://localhost:9222"

	AliExpressCmd.Flags().BoolVar(&autoStartFlag, "auto-start", false, "Auto-start browser and scraping")
}

func runAliExpress(cmd *cobra.Command, args []string) error {
	// Parse input
	parser := input.NewParser()
	var productIDs []string
	var err error

	if fileFlag != "" {
		productIDs, err = parser.ParseFile(fileFlag)
	} else if urlFlag != "" {
		productIDs, err = parser.ParseURLs(urlFlag)
	} else {
		return fmt.Errorf("either --url or --file must be specified")
	}

	if err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	// Validate IDs
	validator := input.NewValidator()
	validIDs, errs := validator.ValidateIDs(productIDs)
	if len(errs) > 0 {
		fmt.Printf("Warning: %d invalid IDs skipped\n", len(errs))
	}
	if len(validIDs) == 0 {
		return fmt.Errorf("no valid product IDs found")
	}

	// Setup browser controller
	browserURL, _ := cmd.Flags().GetString("browser")
	externalBrowser := browserURL != ""

	var ctrl *browser.Controller

	if externalBrowser {
		browserURLFlag = browserURL
		// Connect to existing browser
		ctrl = browser.NewController(browser.Options{
			BrowserURL: browserURLFlag,
		})
		ctx := context.Background()
		if err := ctrl.Connect(ctx, browserURLFlag); err != nil {
			return fmt.Errorf("failed to connect to browser: %w", err)
		}
	} else {
		// TUI will manage browser lifecycle
		userDataDir, err := browser.GetDefaultUserDataDir()
		if err != nil {
			return fmt.Errorf("failed to get user data dir: %w", err)
		}
		ctrl = browser.NewController(browser.Options{
			DebugPort:   9222,
			Headless:    false,
			UserDataDir: userDataDir,
		})
	}

	// Only close browser if we're managing it (not external)
	if !externalBrowser {
		defer ctrl.Close()
	}

	// Setup output
	outputMode := output.SingleFile
	if outputPerURL {
		outputMode = output.PerURL
	}

	outPath := outputFlag
	if outputPerURL {
		outPath = outputDirFlag
	}

	writer, err := output.NewWriter(outputMode, outPath)
	if err != nil {
		return fmt.Errorf("failed to create output writer: %w", err)
	}
	defer writer.Close()

	// Create scraper
	allExtractors := extractors.GetAllExtractors()
	scraper := aliexpress.NewScraper(ctrl, allExtractors)

	// Setup control channel
	controlChan := make(chan tui.ControlMsg, 10)

	// Setup TUI
	progModel := tui.NewDashboardModel(validIDs, controlChan)
	progModel.SetBrowserController(ctrl)
	progModel.SetExternalBrowser(externalBrowser)
	progModel.SetAutoStart(autoStartFlag)

	p := tea.NewProgram(progModel)

	// Track current index for progress callback
	var currentIndex int

	// Set progress callback for extractor updates
	scraper.SetProgressCallback(func(extractorName string) {
		p.Send(tui.ExtractorProgressMsg{
			ProductIndex: currentIndex,
			Extractor:    extractorName,
		})
	})

	// Run scraping in goroutine
	go func() {
		defer close(controlChan)

		// Wait for start signal (unless auto-start)
		if !autoStartFlag {
			waiting := true
			for waiting {
				msg := <-controlChan
				if msg.Type == "start" {
					waiting = false
				} else if msg.Type == "stop" {
					return
				}
			}
		}

		pauseChan := make(chan struct{})
		isPaused := false

		for i, id := range validIDs {
			currentIndex = i

			// Check control messages (non-blocking)
			select {
			case msg := <-controlChan:
				switch msg.Type {
				case "pause":
					isPaused = true
				case "resume":
					isPaused = false
					close(pauseChan)
					pauseChan = make(chan struct{})
				case "skip":
					p.Send(tui.ProductFailMsg{Index: i, Error: "Skipped by user"})
					continue
				case "stop":
					p.Send(tui.LogMsg{Level: "INFO", Message: "Stopped by user"})
					return
				}
			default:
			}

			// Wait if paused
			if isPaused {
				<-pauseChan
			}

			p.Send(tui.ProductStartMsg{Index: i})
			p.Send(tui.LogMsg{Level: "INFO", Message: "Starting product: " + id})

			start := time.Now()

			// Send extractor progress
			p.Send(tui.ExtractorProgressMsg{ProductIndex: i, Extractor: "navigating"})
			product, err := scraper.ScrapeProduct(id)

			if err != nil {
				p.Send(tui.ProductFailMsg{Index: i, Error: err.Error()})

				// Check if CAPTCHA
				if strings.Contains(err.Error(), "CAPTCHA") {
					p.Send(tui.BrowserStatusMsg{Connected: false, Message: "CAPTCHA detected"})
				}
				continue
			}

			p.Send(tui.ExtractorProgressMsg{ProductIndex: i, Extractor: "writing"})

			// Write product
			if err := writer.WriteProduct(product); err != nil {
				p.Send(tui.ProductFailMsg{Index: i, Error: err.Error()})
				continue
			}

			elapsed := time.Since(start)
			p.Send(tui.ProductDoneMsg{Index: i, Elapsed: elapsed})

			// Update stats
			completed := i + 1
			avgTime := time.Since(start) / time.Duration(completed)
			remaining := len(validIDs) - completed
			eta := time.Duration(remaining) * avgTime

			p.Send(tui.StatsUpdateMsg{
				AvgTime:      avgTime,
				ETA:          eta,
				TotalElapsed: time.Since(start),
			})
		}

		p.Send(tui.AllDoneMsg{})
	}()

	// Run TUI
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
