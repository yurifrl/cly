package notify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSoundEnabled(t *testing.T) {
	// Create temp dir for testing
	tmpDir, err := os.MkdirTemp("", "cly-sound-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	soundFile := filepath.Join(tmpDir, "sound")

	tests := []struct {
		name           string
		fileContent    string
		fileExists     bool
		envVar         string
		configValue    bool
		expected       bool
	}{
		{
			name:        "file on takes precedence",
			fileContent: "on",
			fileExists:  true,
			envVar:      "off",
			configValue: false,
			expected:    true,
		},
		{
			name:        "file off takes precedence",
			fileContent: "off",
			fileExists:  true,
			envVar:      "on",
			configValue: true,
			expected:    false,
		},
		{
			name:        "env var on when no file",
			fileExists:  false,
			envVar:      "on",
			configValue: false,
			expected:    true,
		},
		{
			name:        "config value when no file or env",
			fileExists:  false,
			envVar:      "",
			configValue: true,
			expected:    true,
		},
		{
			name:        "default false when nothing set",
			fileExists:  false,
			envVar:      "",
			configValue: false,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up sound file
			os.Remove(soundFile)

			// Save original env
			origEnv := os.Getenv("SOUND")
			defer func() {
				if origEnv != "" {
					os.Setenv("SOUND", origEnv)
				} else {
					os.Unsetenv("SOUND")
				}
			}()

			// Setup file
			if tt.fileExists {
				err := os.WriteFile(soundFile, []byte(tt.fileContent), 0644)
				require.NoError(t, err)
			}

			// Setup env
			if tt.envVar != "" {
				os.Setenv("SOUND", tt.envVar)
			} else {
				os.Unsetenv("SOUND")
			}

			result := isSoundEnabled(soundFile, tt.configValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetSoundEnabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cly-sound-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	soundFile := filepath.Join(tmpDir, "sound")

	tests := []struct {
		name     string
		enabled  bool
		expected string
	}{
		{"enable sound", true, "on"},
		{"disable sound", false, "off"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setSoundEnabled(soundFile, tt.enabled)
			require.NoError(t, err)

			content, err := os.ReadFile(soundFile)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(content))
		})
	}
}
