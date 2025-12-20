package ai

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
		apiKey  string
		model   string
		wantErr bool
	}{
		{
			name:    "valid config",
			apiKey:  "sk-ant-test-key",
			model:   "claude-sonnet-4-5",
			wantErr: false,
		},
		{
			name:    "empty model defaults",
			apiKey:  "sk-ant-test-key",
			model:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey, tt.model)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotEmpty(t, client.model)
			}
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	// Skip if mods not available
	if _, err := exec.LookPath("mods"); err != nil {
		t.Skip("mods binary not found in PATH")
	}

	// Skip if no API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	client, err := NewClient(apiKey, "claude-sonnet-4-5")
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
