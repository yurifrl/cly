// Package llm provides a unified interface for LLM providers (Anthropic, OpenAI).
// It supports streaming responses and maintains conversation history.
package llm

import (
	"context"
	"fmt"
	"os"
)

// Role represents a message role in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    Role
	Content string
}

// StreamChunk is sent for each token chunk during streaming.
type StreamChunk struct {
	Text string
	Done bool
	Err  error
}

// Client is the interface for LLM providers.
type Client interface {
	// Stream sends messages with a system prompt and streams the response.
	// The returned channel receives StreamChunk values until Done is true or an error occurs.
	Stream(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error)

	// Complete sends messages with a system prompt and returns the full response.
	// Use for structured output (JSON) where streaming is unnecessary.
	Complete(ctx context.Context, systemPrompt string, messages []Message) (string, error)
}

// Provider identifies the LLM provider.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
)

// Config holds the configuration for creating an LLM client.
type Config struct {
	Provider     Provider
	Model        string
	APIKey       string
	APIKeyEnv    string
	SystemPrompt string
}

// NewClient creates a new LLM client based on the config.
func NewClient(cfg Config) (Client, error) {
	apiKey := resolveAPIKey(cfg)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key found for provider %s: set api_key in config, %s env var, or default env var", cfg.Provider, cfg.APIKeyEnv)
	}

	switch cfg.Provider {
	case ProviderAnthropic:
		return newAnthropicClient(apiKey, cfg.Model)
	case ProviderOpenAI:
		return newOpenAIClient(apiKey, cfg.Model)
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: anthropic, openai)", cfg.Provider)
	}
}

// resolveAPIKey resolves the API key from config, env var, or default.
func resolveAPIKey(cfg Config) string {
	// 1. Direct config value
	if cfg.APIKey != "" {
		return cfg.APIKey
	}

	// 2. Custom env var from config
	if cfg.APIKeyEnv != "" {
		if key := os.Getenv(cfg.APIKeyEnv); key != "" {
			return key
		}
	}

	// 3. Default env var for provider
	switch cfg.Provider {
	case ProviderAnthropic:
		return os.Getenv("ANTHROPIC_API_KEY")
	case ProviderOpenAI:
		return os.Getenv("OPENAI_API_KEY")
	}

	return ""
}
