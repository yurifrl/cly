package notify

import (
	"os"
	"path/filepath"
	"strings"
)

// isSoundEnabled checks if sound is enabled with priority:
// 1. Sound config file (~/.config/cly/sound)
// 2. SOUND env var
// 3. config.yaml value
// 4. Default: false
func isSoundEnabled(soundFilePath string, configValue bool) bool {
	// Priority 1: Check file
	if content, err := os.ReadFile(soundFilePath); err == nil {
		value := strings.TrimSpace(string(content))
		return value == "on"
	}

	// Priority 2: Check env var
	if envValue := os.Getenv("SOUND"); envValue != "" {
		return envValue == "on"
	}

	// Priority 3: Config value
	return configValue
}

// setSoundEnabled writes the sound preference to the config file
func setSoundEnabled(soundFilePath string, enabled bool) error {
	// Ensure directory exists
	dir := filepath.Dir(soundFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	value := "off"
	if enabled {
		value = "on"
	}

	return os.WriteFile(soundFilePath, []byte(value), 0644)
}

// getSoundFilePath returns the path to the sound config file
func getSoundFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config/cly/sound")
}
