package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Client wraps mods binary for sending messages
type Client struct {
	model string
}

// NewClient creates a new mods client wrapper
func NewClient(apiKey, model string) (*Client, error) {
	// Default model if not specified
	if model == "" {
		model = "claude-sonnet-4-5"
	}

	// Check if mods is available
	if _, err := exec.LookPath("mods"); err != nil {
		return nil, fmt.Errorf("mods binary not found in PATH: %w", err)
	}

	return &Client{
		model: model,
	}, nil
}

// SendMessage sends a message via mods and returns the response
func (c *Client) SendMessage(ctx context.Context, conversationID, userMsg string) (string, error) {
	args := []string{"-m", c.model}

	if conversationID != "" {
		// Continue existing conversation
		args = append(args, "-c", conversationID)
	} else {
		// Start new conversation (mods will auto-generate ID)
		args = append(args, "-t")
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
