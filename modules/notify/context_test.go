package notify

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildContextString(t *testing.T) {
	tests := []struct {
		name                string
		zellijSession       string
		claudeSession       string
		expected            string
	}{
		{
			name:          "both sessions",
			zellijSession: "my-zellij",
			claudeSession: "my-claude",
			expected:      "[my-zellij] my-claude",
		},
		{
			name:          "only zellij",
			zellijSession: "my-zellij",
			claudeSession: "",
			expected:      "[my-zellij]",
		},
		{
			name:          "only claude",
			zellijSession: "",
			claudeSession: "my-claude",
			expected:      "my-claude",
		},
		{
			name:          "neither",
			zellijSession: "",
			claudeSession: "",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original envs
			origZellij := os.Getenv("ZELLIJ_SESSION_NAME")
			origClaude := os.Getenv("CLAUDE_SESSION_NAME")
			defer func() {
				if origZellij != "" {
					os.Setenv("ZELLIJ_SESSION_NAME", origZellij)
				} else {
					os.Unsetenv("ZELLIJ_SESSION_NAME")
				}
				if origClaude != "" {
					os.Setenv("CLAUDE_SESSION_NAME", origClaude)
				} else {
					os.Unsetenv("CLAUDE_SESSION_NAME")
				}
			}()

			// Set test envs
			if tt.zellijSession != "" {
				os.Setenv("ZELLIJ_SESSION_NAME", tt.zellijSession)
			} else {
				os.Unsetenv("ZELLIJ_SESSION_NAME")
			}
			if tt.claudeSession != "" {
				os.Setenv("CLAUDE_SESSION_NAME", tt.claudeSession)
			} else {
				os.Unsetenv("CLAUDE_SESSION_NAME")
			}

			result := buildContextString()
			assert.Equal(t, tt.expected, result)
		})
	}
}
