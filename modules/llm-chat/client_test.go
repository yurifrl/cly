package llmchat

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	// Check if mods is available
	if _, err := exec.LookPath("mods"); err != nil {
		t.Skip("mods binary not found in PATH")
	}

	tests := []struct {
		name    string
		flags   map[string]interface{}
		wantErr bool
	}{
		{
			name: "with model",
			flags: map[string]interface{}{
				"model": "claude-sonnet-4-5",
			},
			wantErr: false,
		},
		{
			name:    "empty flags",
			flags:   map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.flags)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	// Skip unless explicitly enabled - this makes real API calls
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	// Skip if mods not available
	if _, err := exec.LookPath("mods"); err != nil {
		t.Skip("mods binary not found in PATH")
	}

	// Skip if no API key
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	flags := map[string]interface{}{
		"model": "claude-sonnet-4-5",
		"api":   "anthropic",
	}
	client, err := NewClient(flags)
	require.NoError(t, err)

	tests := []struct {
		name           string
		conversationID string
		userMsg        string
		wantErr        bool
	}{
		{
			name:           "simple question",
			conversationID: "",
			userMsg:        "What is 2+2? Answer with just the number.",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := client.SendMessage(ctx, tt.conversationID, tt.userMsg, true)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, response)
			}
		})
	}
}
