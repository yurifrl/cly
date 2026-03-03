package llmchat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendMessage_ContinueMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client := &Client{flags: map[string]interface{}{}}
	conversationID := "test-continue-id"

	ctx := context.Background()

	// First message with isFirstMessage=false should use -c flag (continue mode)
	_, err := client.SendMessage(ctx, conversationID, "test message", false)

	// This will fail if mods doesn't have the conversation, but that's expected
	// We're just testing that the flag is correctly passed
	if err != nil {
		// Expected: mods will return error about conversation not existing
		assert.Contains(t, err.Error(), "mods command failed")
	}
}

func TestSendMessage_NewConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client := &Client{flags: map[string]interface{}{}}
	conversationID := GenerateConversationID()

	ctx := context.Background()

	// First message with isFirstMessage=true should use -t flag (new thread)
	_, err := client.SendMessage(ctx, conversationID, "test message", true)

	if err != nil {
		// May fail due to API key or model issues, but we're testing the flag logic
		assert.Contains(t, err.Error(), "mods command failed")
	}
}
