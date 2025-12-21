package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Client wraps mods binary for sending messages
type Client struct {
	flags map[string]interface{}
}

// NewClient creates a new mods client wrapper
func NewClient(flags map[string]interface{}) (*Client, error) {
	// Check if mods is available
	if _, err := exec.LookPath("mods"); err != nil {
		return nil, fmt.Errorf("mods binary not found in PATH: %w", err)
	}

	return &Client{
		flags: flags,
	}, nil
}

// SendMessage sends a message via mods and returns the response
func (c *Client) SendMessage(ctx context.Context, conversationID, userMsg string, isFirstMessage bool) (string, error) {
	var args []string

	// Add flags first
	if model, ok := c.flags["model"].(string); ok {
		args = append(args, "-m", model)
	}

	// Add conversation flags
	if continueLast, ok := c.flags["continue-last"].(bool); ok && continueLast {
		args = append(args, "-C")
	} else if conversationID != "" {
		if isFirstMessage {
			// Start new conversation with specific ID using -t
			args = append(args, "-t", conversationID)
		} else {
			// Continue existing conversation using -c
			args = append(args, "-c", conversationID)
		}
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

