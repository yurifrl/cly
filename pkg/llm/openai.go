package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type openaiClient struct {
	client *openai.Client
	model  string
}

func newOpenAIClient(apiKey, model string) (*openaiClient, error) {
	if model == "" {
		model = "gpt-4o"
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &openaiClient{
		client: &client,
		model:  model,
	}, nil
}

func (c *openaiClient) Stream(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error) {
	// Convert messages to OpenAI format
	var openaiMsgs []openai.ChatCompletionMessageParamUnion
	if systemPrompt != "" {
		openaiMsgs = append(openaiMsgs, openai.SystemMessage(systemPrompt))
	}
	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			openaiMsgs = append(openaiMsgs, openai.UserMessage(msg.Content))
		case RoleAssistant:
			openaiMsgs = append(openaiMsgs, openai.AssistantMessage(msg.Content))
		}
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: openaiMsgs,
	})

	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)

		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- StreamChunk{Text: chunk.Choices[0].Delta.Content}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("openai stream error: %w", err)}
			return
		}

		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}
