package llm

import (
	"context"
	"fmt"
	"strings"
)

// Candidate identifies one client in a per-request fallback chain.
type Candidate struct {
	Name   string
	Client Client
}

// AttemptFailure records a failed candidate before the next candidate is tried.
type AttemptFailure struct {
	Candidate string
	Err       error
}

// FallbackClient retries every request once against each configured candidate.
type FallbackClient struct {
	candidates []Candidate
	onFailure  func(AttemptFailure)
}

func NewFallbackClient(candidates []Candidate, onFailure func(AttemptFailure)) Client {
	return &FallbackClient{candidates: candidates, onFailure: onFailure}
}

func (c *FallbackClient) Complete(ctx context.Context, systemPrompt string, messages []Message) (string, error) {
	var failures []string
	for _, candidate := range c.candidates {
		response, err := candidate.Client.Complete(ctx, systemPrompt, messages)
		if err == nil {
			return response, nil
		}
		if c.onFailure != nil {
			c.onFailure(AttemptFailure{Candidate: candidate.Name, Err: err})
		}
		failures = append(failures, fmt.Sprintf("%s: %v", candidate.Name, err))
	}
	return "", fmt.Errorf("all AI candidates failed: %s", strings.Join(failures, "; "))
}

func (c *FallbackClient) Stream(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error) {
	var failures []string
	for _, candidate := range c.candidates {
		stream, err := candidate.Client.Stream(ctx, systemPrompt, messages)
		if err == nil {
			return stream, nil
		}
		if c.onFailure != nil {
			c.onFailure(AttemptFailure{Candidate: candidate.Name, Err: err})
		}
		failures = append(failures, fmt.Sprintf("%s: %v", candidate.Name, err))
	}
	return nil, fmt.Errorf("all AI candidates failed: %s", strings.Join(failures, "; "))
}
