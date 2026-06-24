package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

type anthropicClient struct {
	client *anthropic.Client
	model  string
}

func newAnthropicClient(apiKey, model string, useBedrock bool) (*anthropicClient, error) {	if model == "" {
		model = "claude-sonnet-4-5-20250929"
	}

	var opts []option.RequestOption
	if useBedrock {
		// Auth comes from the AWS chain. When AWS_BEARER_TOKEN_BEDROCK is set we
		// force it explicitly: LoadDefaultConfig under an AWS_PROFILE otherwise
		// resolves a different bearer provider and ignores the env token.
		// Also strip X-Api-Key, which the SDK auto-sets from $ANTHROPIC_API_KEY
		// and Bedrock rejects ("Invalid API Key format").
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, fmt.Errorf("bedrock: load aws config: %w", err)
		}
		if tok := os.Getenv("AWS_BEARER_TOKEN_BEDROCK"); tok != "" {
			awsCfg.BearerAuthTokenProvider = bedrock.NewStaticBearerTokenProvider(tok)
		}
		opts = append(opts,
			bedrock.WithConfig(awsCfg),
			option.WithHeaderDel("X-Api-Key"),
		)
	} else {
		opts = append(opts, option.WithAPIKey(apiKey))
	}

	client := anthropic.NewClient(opts...)

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
