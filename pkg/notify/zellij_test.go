package notify

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZellijNotifier_Available(t *testing.T) {
	tests := []struct {
		name      string
		zellijEnv string
		expected  bool
	}{
		{"in zellij session", "1", true},
		{"not in zellij", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := os.Getenv("ZELLIJ")
			defer func() {
				if orig != "" {
					os.Setenv("ZELLIJ", orig)
				} else {
					os.Unsetenv("ZELLIJ")
				}
			}()

			if tt.zellijEnv != "" {
				os.Setenv("ZELLIJ", tt.zellijEnv)
			} else {
				os.Unsetenv("ZELLIJ")
			}

			zn := NewZellijNotifier("test", false, false, false)
			assert.Equal(t, tt.expected, zn.Available())
		})
	}
}

func TestMapEventToAttentionState(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		expected  string
	}{
		{"notification maps to waiting", "notification", "waiting"},
		{"stop maps to completed", "stop", "completed"},
		{"unknown passes through", "posttooluse", "posttooluse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapEventToAttentionState(tt.eventType))
		})
	}
}

func TestBuildNotifyArgs(t *testing.T) {
	tests := []struct {
		name        string
		eventType   string
		paneID      string
		sessionName string
		expected    []string
	}{
		{
			"with pane and session",
			"notification", "42", "my-session",
			[]string{"pipe", "-n", "notify", "-a", "pane_id=42", "-a", "session_name=my-session", "notification"},
		},
		{
			"with pane only",
			"stop", "7", "",
			[]string{"pipe", "-n", "notify", "-a", "pane_id=7", "stop"},
		},
		{
			"no pane no session",
			"notification", "", "",
			[]string{"pipe", "-n", "notify", "notification"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildNotifyArgs(tt.eventType, tt.paneID, tt.sessionName))
		})
	}
}

func TestBuildAttentionPipeName(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		paneID    string
		expected  string
	}{
		{
			"notification with pane ID",
			"notification", "42",
			"zellij-attention::waiting::42",
		},
		{
			"stop with pane ID",
			"stop", "7",
			"zellij-attention::completed::7",
		},
		{
			"empty pane ID",
			"notification", "",
			"zellij-attention::waiting::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildAttentionPipeName(tt.eventType, tt.paneID))
		})
	}
}
