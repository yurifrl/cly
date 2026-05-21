package llmchat

import (
	"context"
	"strings"

	"github.com/yurifrl/cly/pkg/ai"
	"github.com/yurifrl/cly/pkg/llm"
)

// Client wraps pkg/llm for sending messages
type Client struct {
	llmClient llm.Client
}

// NewClient builds an LLM client honoring (in priority order):
//   1. Per-call flags (`api`, `model`, `api_key`) supplied by the caller.
//   2. Module overrides under `modules.llm-chat.ai` in cly config.
//   3. Global `ai:` defaults from cly config.
//
// Flags are translated into the same shape as a config override block so
// `pkg/ai` does the merge in exactly one place.
func NewClient(flags map[string]interface{}) (*Client, error) {
	override := flagsAsAIOverride(flags)
	client, err := ai.NewClientWith(override)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ai.ErrDisabled
	}
	return &Client{llmClient: client}, nil
}

// flagsAsAIOverride remaps the historical `api`/`model`/`api_key` flag
// names onto the shape `pkg/ai` expects. Empty/missing flags are dropped
// so the override does not clobber config values with zero strings.
func flagsAsAIOverride(flags map[string]interface{}) map[string]interface{} {
	if len(flags) == 0 {
		return nil
	}
	o := map[string]interface{}{}
	if v, ok := flags["api"].(string); ok && v != "" {
		o["provider"] = v
	}
	if v, ok := flags["model"].(string); ok && v != "" {
		o["model"] = v
	}
	if v, ok := flags["api_key"].(string); ok && v != "" {
		o["api_key"] = v
	}
	if len(o) == 0 {
		return nil
	}
	return o
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
