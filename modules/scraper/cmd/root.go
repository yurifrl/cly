package cmd

import (
	"github.com/spf13/cobra"
)

// ScraperCmd is the root command for the scraper module
var ScraperCmd = &cobra.Command{
	Use:   "scraper",
	Short: "Web scraping commands",
	Long:  `Scrape product data from e-commerce websites using browser automation.`,
}

func init() {
	ScraperCmd.AddCommand(BrowserCmd)
}

// Register registers the scraper module with the parent command
func Register(parent *cobra.Command) {
	parent.AddCommand(ScraperCmd)
}
