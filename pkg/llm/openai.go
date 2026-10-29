package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type openaiClient struct {
	client    *openai.Client
	model     string
	maxTokens int64
}

func newOpenAIClient(apiKey, model, baseURL string, maxTokens int64) (*openaiClient, error) {
	if model == "" {
		model = "gpt-4o"
	}

	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	return &openaiClient{
		client:    &client,
		model:     model,
		maxTokens: maxTokens,
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

	params := openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: openaiMsgs,
	}
	if c.maxTokens > 0 {
		params.MaxTokens = openai.Int(c.maxTokens)
	}
	stream := c.client.Chat.Completions.NewStreaming(ctx, params)

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
			ch <- StreamChunk{Err: fmt.Errorf("stream error: %w", err)}
			return
		}

		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}

func (c *openaiClient) Complete(ctx context.Context, systemPrompt string, messages []Message) (string, error) {
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

	params := openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: openaiMsgs,
	}
	if c.maxTokens > 0 {
		params.MaxTokens = openai.Int(c.maxTokens)
	}
	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("complete error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
