package notify

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZellijNotifier_Available(t *testing.T) {
	tests := []struct {
		name          string
		zellijEnv     string
		expected      bool
	}{
		{"in zellij session", "1", true},
		{"not in zellij", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			orig := os.Getenv("ZELLIJ")
			defer func() {
				if orig != "" {
					os.Setenv("ZELLIJ", orig)
				} else {
					os.Unsetenv("ZELLIJ")
				}
			}()

			// Set test env
			if tt.zellijEnv != "" {
				os.Setenv("ZELLIJ", tt.zellijEnv)
			} else {
				os.Unsetenv("ZELLIJ")
			}

			zn := NewZellijNotifier("test")
			assert.Equal(t, tt.expected, zn.Available())
		})
	}
}

func TestZellijNotifier_UnavailableWhenNotInSession(t *testing.T) {
	orig := os.Getenv("ZELLIJ")
	defer func() {
		if orig != "" {
			os.Setenv("ZELLIJ", orig)
		} else {
			os.Unsetenv("ZELLIJ")
		}
	}()

	os.Unsetenv("ZELLIJ")

	zn := NewZellijNotifier("test")
	assert.False(t, zn.Available())
}
