package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type anthropicClient struct {
	client *anthropic.Client
	model  string
}

func newAnthropicClient(apiKey, model string) (*anthropicClient, error) {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &anthropicClient{
		client: &client,
		model:  model,
	}, nil
}

func (c *anthropicClient) Stream(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error) {
	// Convert messages to Anthropic format
	var anthropicMsgs []anthropic.MessageParam
	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case RoleAssistant:
			anthropicMsgs = append(anthropicMsgs, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: int64(4096),
		Messages:  anthropicMsgs,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	stream := c.client.Messages.NewStreaming(ctx, params)

	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)

		for stream.Next() {
			event := stream.Current()

			// Use AsAny() to switch on variant type
			switch variant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				if variant.Delta.Text != "" {
					ch <- StreamChunk{Text: variant.Delta.Text}
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("anthropic stream error: %w", err)}
			return
		}

		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}

func (c *anthropicClient) Complete(ctx context.Context, systemPrompt string, messages []Message) (string, error) {
	var anthropicMsgs []anthropic.MessageParam
	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case RoleAssistant:
			anthropicMsgs = append(anthropicMsgs, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: int64(16384),
		Messages:  anthropicMsgs,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("anthropic complete error: %w", err)
	}

	var result string
	for _, block := range resp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	return result, nil
}
