package browser

import (
	"os"
	"path/filepath"
)

// GetDefaultUserDataDir returns the default user data directory path
func GetDefaultUserDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".cly", "scraper", "chrome")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}

// EnsureUserDataDir ensures the user data directory exists
func EnsureUserDataDir(path string) error {
	if path == "" {
		return nil
	}

	return os.MkdirAll(path, 0755)
}
