package llmchat

import (
	"context"
	"strings"

	"github.com/yurifrl/cly/pkg/llm"
)

// Client wraps pkg/llm for sending messages
type Client struct {
	llmClient llm.Client
}

// NewClient creates a new LLM client using the direct SDK
func NewClient(flags map[string]interface{}) (*Client, error) {
	provider := "anthropic"
	model := ""
	apiKey := ""

	if p, ok := flags["api"].(string); ok && p != "" {
		provider = p
	}
	if m, ok := flags["model"].(string); ok && m != "" {
		model = m
	}
	if k, ok := flags["api_key"].(string); ok && k != "" {
		apiKey = k
	}

	cfg := llm.Config{
		Provider: llm.Provider(provider),
		Model:    model,
		APIKey:   apiKey,
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{llmClient: client}, nil
}

// SendMessage sends a message and returns the full response (non-streaming).
func (c *Client) SendMessage(ctx context.Context, conversationID, userMsg string, isFirstMessage bool) (string, error) {
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: userMsg},
	}

	ch, err := c.llmClient.Stream(ctx, "", msgs)
	if err != nil {
		return "", err
	}

	// Collect all chunks into a single response
	var sb strings.Builder
	for chunk := range ch {
		if chunk.Err != nil {
			return sb.String(), chunk.Err
		}
		if chunk.Done {
			break
		}
		sb.WriteString(chunk.Text)
	}

	return sb.String(), nil
}
