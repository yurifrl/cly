package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fallbackTestClient struct {
	response string
	err      error
	calls    int
}

func (c *fallbackTestClient) Stream(context.Context, string, []Message) (<-chan StreamChunk, error) {
	return nil, errors.New("not used in this test")
}

func (c *fallbackTestClient) Complete(context.Context, string, []Message) (string, error) {
	c.calls++
	return c.response, c.err
}

func TestFallbackClientCompletesWithNextCandidateAfterAnyFailure(t *testing.T) {
	primary := &fallbackTestClient{err: errors.New("quota exhausted")}
	fallback := &fallbackTestClient{response: "fallback response"}
	var failures []AttemptFailure

	client := NewFallbackClient([]Candidate{
		{Name: "primary", Client: primary},
		{Name: "fallback", Client: fallback},
	}, func(f AttemptFailure) { failures = append(failures, f) })

	response, err := client.Complete(context.Background(), "system", []Message{{Role: RoleUser, Content: "hello"}})

	require.NoError(t, err)
	assert.Equal(t, "fallback response", response)
	assert.Equal(t, 1, primary.calls)
	assert.Equal(t, 1, fallback.calls)
	require.Len(t, failures, 1)
	assert.Equal(t, "primary", failures[0].Candidate)
	assert.EqualError(t, failures[0].Err, "quota exhausted")
}

func TestFallbackClientReturnsAllFailuresAfterExhaustingCandidates(t *testing.T) {
	first := &fallbackTestClient{err: errors.New("first unavailable")}
	second := &fallbackTestClient{err: errors.New("second unavailable")}

	client := NewFallbackClient([]Candidate{
		{Name: "first", Client: first},
		{Name: "second", Client: second},
	}, nil)

	_, err := client.Complete(context.Background(), "system", nil)

	require.Error(t, err)
	assert.ErrorContains(t, err, "first")
	assert.ErrorContains(t, err, "second")
	assert.Equal(t, 1, first.calls)
	assert.Equal(t, 1, second.calls)
}
