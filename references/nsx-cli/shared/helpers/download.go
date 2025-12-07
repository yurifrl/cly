package helpers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/NSXBet/nsx-cli/shared/interact"
)

func DownloadFileToPath(url, destination string, debug bool) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Fetch content from gist
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch gist from %s: %w", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			interact.Error("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch gist: HTTP %d", resp.StatusCode)
	}

	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read gist content: %w", err)
	}

	// Write content to .golangci.yml in project root
	configPath := filepath.Join(destination, ".golangci.yml")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write .golangci.yml: %w", err)
	}

	if debug {
		fmt.Printf("🔧 Created .golangci.yml with %d bytes from %s\n", len(content), url)
	}
	return nil
}
