package llmchat

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-123")

	tests := []struct {
		name    string
		flags   map[string]interface{}
		wantErr bool
	}{
		{
			name: "with model",
			flags: map[string]interface{}{
				"model": "claude-sonnet-4-20250514",
				"api":   "anthropic",
			},
			wantErr: false,
		},
		{
			name:    "empty flags defaults to anthropic",
			flags:   map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "openai provider",
			flags: map[string]interface{}{
				"api": "openai",
			},
			wantErr: false, // OPENAI_API_KEY may be set in env
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
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	flags := map[string]interface{}{
		"model": "claude-sonnet-4-20250514",
		"api":   "anthropic",
	}
	client, err := NewClient(flags)
	require.NoError(t, err)

	ctx := context.Background()
	response, err := client.SendMessage(ctx, "", "What is 2+2? Answer with just the number.", true)
	require.NoError(t, err)
	assert.NotEmpty(t, response)
}
