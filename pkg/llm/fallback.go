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

// AttemptFailure records a failed candidate before the next candidate is
// tried. Remaining counts candidates still untried (0 after the final
// failure); Next names the candidate about to be attempted ("" after the
// final failure).
type AttemptFailure struct {
	Candidate string
	Err       error
	Remaining int
	Next      string
}

// FallbackClient retries every request once against each configured candidate.
type FallbackClient struct {
	candidates []Candidate
	onFailure  func(AttemptFailure)
}

func NewFallbackClient(candidates []Candidate, onFailure func(AttemptFailure)) Client {
	return &FallbackClient{candidates: candidates, onFailure: onFailure}
}

// failure builds the AttemptFailure for candidate i of n.
func (c *FallbackClient) failure(candidate Candidate, i int, err error) AttemptFailure {
	f := AttemptFailure{Candidate: candidate.Name, Err: err}
	if i+1 < len(c.candidates) {
		f.Next = c.candidates[i+1].Name
		f.Remaining = len(c.candidates) - i - 1
	}
	return f
}

func (c *FallbackClient) Complete(ctx context.Context, systemPrompt string, messages []Message) (string, error) {
	var failures []string
	for i, candidate := range c.candidates {
		response, err := candidate.Client.Complete(ctx, systemPrompt, messages)
		if err == nil {
			return response, nil
		}
		if c.onFailure != nil {
			c.onFailure(c.failure(candidate, i, err))
		}
		failures = append(failures, fmt.Sprintf("%s: %v", candidate.Name, err))
	}
	return "", fmt.Errorf("all AI candidates failed: %s", strings.Join(failures, "; "))
}

func (c *FallbackClient) Stream(ctx context.Context, systemPrompt string, messages []Message) (<-chan StreamChunk, error) {
	var failures []string
	for i, candidate := range c.candidates {
		stream, err := candidate.Client.Stream(ctx, systemPrompt, messages)
		if err == nil {
			return stream, nil
		}
		if c.onFailure != nil {
			c.onFailure(c.failure(candidate, i, err))
		}
		failures = append(failures, fmt.Sprintf("%s: %v", candidate.Name, err))
	}
	return nil, fmt.Errorf("all AI candidates failed: %s", strings.Join(failures, "; "))
}
