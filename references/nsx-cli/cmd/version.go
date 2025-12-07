package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/interact"
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current version of NSX CLI",
	Long:  `Show the current version of NSX CLI`,
	RunE:  runVersion,
}

// Default proxy URL, can be overridden by environment variable
const defaultProxyURL = "https://nsx-cli-proxy.nsx.services"

// getProxyURL returns the configured proxy URL from environment or default
func getProxyURL() string {
	if url := os.Getenv("NSX_CLI_PROXY_URL"); url != "" {
		return url
	}
	return defaultProxyURL
}

func runVersion(cmd *cobra.Command, args []string) error {
	latestVersion, err := getLatestVersion()
	if err != nil {
		interact.Debug("failed to check for latest version: %+v", err)
		return nil
	}
	if !isSameVersion(version, latestVersion) {
		interact.Debug("You are running version: %s", version)
		interact.Debug("Please update to the latest version: %s", latestVersion)
	}
	interact.Success("NSX CLI version: %s", version)
	return nil
}

func getLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(getProxyURL() + "/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			interact.Error("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest version: status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

func isSameVersion(current, latest string) bool {
	// Clean up version strings (remove 'v' prefix if present)
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	return current == latest
}
