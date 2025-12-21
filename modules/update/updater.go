package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

const (
	defaultRepo      = "yurifrl/cly"
	defaultAPIBase   = "https://api.github.com"
	downloadTimeout  = 5 * time.Minute
	checkTimeout     = 30 * time.Second
)

// Updater handles checking for and installing updates
type Updater struct {
	repo       string
	currentVer *Version
	httpClient *http.Client
	apiBaseURL string
}

// ReleaseInfo contains information about a GitHub release
type ReleaseInfo struct {
	Version     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a downloadable release asset
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// New creates a new Updater with the given version
func New(currentVersion string) *Updater {
	return &Updater{
		repo:       defaultRepo,
		currentVer: GetCurrentVersion(currentVersion),
		httpClient: &http.Client{
			Timeout: checkTimeout,
		},
		apiBaseURL: defaultAPIBase,
	}
}

// CheckLatest queries GitHub API for the latest release
func (u *Updater) CheckLatest() (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", u.apiBaseURL, u.repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

// FindAssetForPlatform finds the asset matching the current platform
func (r *ReleaseInfo) FindAssetForPlatform(os, arch string) (Asset, bool) {
	assetName := getAssetName(os, arch)

	for _, asset := range r.Assets {
		if asset.Name == assetName {
			return asset, true
		}
	}

	return Asset{}, false
}

// Download downloads an asset to the specified path
func (u *Updater) Download(asset Asset, destPath string) error {
	// Create HTTP client with longer timeout for downloads
	client := &http.Client{
		Timeout: downloadTimeout,
	}

	req, err := http.NewRequest("GET", asset.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy content
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Make executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	return nil
}

// Install installs a new binary, backing up the current one
func (u *Updater) Install(newPath, currentPath string) error {
	// Verify new binary exists
	if _, err := os.Stat(newPath); err != nil {
		return fmt.Errorf("new binary not found: %w", err)
	}

	// Create backup
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.bak.%s", currentPath, timestamp)

	if _, err := os.Stat(currentPath); err == nil {
		if err := copyFile(currentPath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Install new binary
	if err := os.Rename(newPath, currentPath); err != nil {
		// Attempt rollback
		if backupPath != "" {
			os.Rename(backupPath, currentPath)
		}
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Cleanup backup on success
	if backupPath != "" {
		os.Remove(backupPath)
	}

	return nil
}

// detectPlatform returns the current OS and architecture
func detectPlatform() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

// getAssetName returns the asset name for the given platform
func getAssetName(os, arch string) string {
	return fmt.Sprintf("cly-%s-%s", os, arch)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
