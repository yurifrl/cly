package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Client wraps mods binary for sending messages
type Client struct {
	model string
}

// NewClient creates a new mods client wrapper
func NewClient(apiKey, model string) (*Client, error) {
	// Check if mods is available
	if _, err := exec.LookPath("mods"); err != nil {
		return nil, fmt.Errorf("mods binary not found in PATH: %w", err)
	}

	return &Client{
		model: model, // Empty means use mods default
	}, nil
}

// SendMessage sends a message via mods and returns the response
func (c *Client) SendMessage(ctx context.Context, conversationID, userMsg string, isFirstMessage bool) (string, error) {
	var args []string

	if conversationID != "" {
		if isFirstMessage {
			// Start new conversation with specific ID using -t
			args = []string{"-t", conversationID}
		} else {
			// Continue existing conversation using -c
			args = []string{"-c", conversationID}
		}
	} else {
		// No conversation ID provided
		args = []string{}
	}

	// Only add model if specified (otherwise mods uses its config)
	if c.model != "" {
		args = append([]string{"-m", c.model}, args...)
	}

	cmd := exec.CommandContext(ctx, "mods", args...)
	cmd.Stdin = bytes.NewBufferString(userMsg)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("mods command failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}

// generateConversationID generates a unique conversation ID
func generateConversationID() string {
	return fmt.Sprintf("modsi-%d", time.Now().Unix())
}
