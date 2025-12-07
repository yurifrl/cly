package cmd

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/team/sre/pkg/incidentsio"
)

var (
	startDate    string
	endDate      string
	outputFormat string
	statusFilter string

	incidentsCmd = &cobra.Command{
		Use:   "incidentsio",
		Short: "Fetch and display incidents from incident.io",
		Long:  `Fetch and display incidents and their follow-ups from the incident.io API.`,
		RunE:  runIncidentsIO,
	}
)

func init() {
	incidentsCmd.Flags().StringVar(&startDate, "start-date", "", "Start date in YYYY-MM-DD format (defaults to 7 days ago)")
	incidentsCmd.Flags().StringVar(&endDate, "end-date", "", "End date in YYYY-MM-DD format (defaults to today)")
	incidentsCmd.Flags().StringVar(&outputFormat, "format", "markdown", "Output format: json, markdown, charm, or text")
	incidentsCmd.Flags().StringVar(&statusFilter, "status", "", "Filter incidents by status")

	RootCmd.AddCommand(incidentsCmd)
}

func runIncidentsIO(cmd *cobra.Command, args []string) error {
	// Get API key from environment
	apiKey := os.Getenv("INCIDENTSIO_API_KEY")
	if apiKey == "" {
		return cmd.Usage()
	}

	// Set default date range if not provided (last 7 days)
	now := time.Now()
	parsedStart := now.AddDate(0, 0, -7)
	parsedEnd := now

	if startDate != "" {
		var err error
		parsedStart, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			return err
		}
	}

	if endDate != "" {
		var err error
		parsedEnd, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return err
		}
	}

	// Create handler with configuration
	handler := incidentsio.NewHandler(incidentsio.Config{
		AuthToken: apiKey,
	})

	// Create options from CLI flags
	opts := incidentsio.Options{
		Format:    outputFormat,
		Status:    statusFilter,
		StartDate: parsedStart,
		EndDate:   parsedEnd, // Use exact end date, not adding a day
	}

	// Execute the handler with context and options
	return handler.List(context.Background(), opts)
}
