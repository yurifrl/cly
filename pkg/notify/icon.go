package notify

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed assets/icon.png
var embeddedIcon []byte

// GetIconPath returns the path to the notification icon
// Extracts embedded icon to ~/.local/share/cly/notify-icon.png on first use
func GetIconPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use XDG data directory standard: ~/.local/share/cly
	iconPath := filepath.Join(homeDir, ".local", "share", "cly", "notify-icon.png")

	// Check if icon already exists
	if _, err := os.Stat(iconPath); err == nil {
		return iconPath, nil
	}

	// Ensure directory exists
	dir := filepath.Dir(iconPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Write embedded icon to file
	if err := os.WriteFile(iconPath, embeddedIcon, 0644); err != nil {
		return "", err
	}

	return iconPath, nil
}
