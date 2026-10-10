package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// decisions.go — Decisions API client against the AIHub gateway, the System
// One surface for the typesafe/jev-* decision models (served as
// aihub/jev-latest). This is its own wire contract (POST /api/alpha/decisions):
// a state object plus typed questions in, typed answers with probabilities
// out — no free-form text to parse. The client is domain-blind: it knows the
// HTTP surface only; modules own how they phrase questions.
//
// Requires AIHUB_API_KEY; DECISIONS_BASE_URL overrides the gateway root
// (dev/test harnesses only).

const (
	decisionsAPIPath = "/api/alpha/decisions"
	decisionsBaseURL = "https://ai-llm-gateway.fbr.land"
)

// DecisionQuestion is one typed question in a Decisions request.
type DecisionQuestion struct {
	// ID is the answer key, echoed back in the response's answers map.
	ID string
	// Type is the primitive: "choice" (pick one option), "noul"
	// (probability a condition holds), "score" (position on an ordered scale).
	Type string
	// Instructions is the question text.
	Instructions string
	// Criteria is type-dependent: for choice/noul a map of option id to
	// description (noul uses true/false keys); for score an ordered slice of
	// level descriptions.
	Criteria any
}

// DecisionAnswer is one parsed answer keyed by its question id. Only the
// fields matching the answer type are populated.
type DecisionAnswer struct {
	Type          string             // "choice" | "noul" | "score"
	Choice        string             // choice: the selected option id
	Noul          float64            // noul: P(true)
	ScorePosition float64            // score: weighted position on the scale
	Confidence    float64            // when the API reports it
	Probabilities map[string]float64 // choice: per-option probabilities
}

// DecisionsResult is one request's answers plus the usage facts (cost is
// USD; the gateway reports it on every response).
type DecisionsResult struct {
	Answers map[string]DecisionAnswer
	Cost    float64
}

// DecisionsClient calls the AIHub gateway's Decisions API. Safe for concurrent use.
type DecisionsClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewDecisionsClient builds the client from AIHUB_API_KEY. It errors
// when the key is unset so callers can skip wiring (the feature stays off)
// instead of failing on every call.
func NewDecisionsClient() (*DecisionsClient, error) {
	key := strings.TrimSpace(os.Getenv("AIHUB_API_KEY"))
	if key == "" {
		return nil, errors.New("AIHUB_API_KEY is not set (required for the AIHub Decisions API)")
	}
	baseURL := strings.TrimSpace(os.Getenv("DECISIONS_BASE_URL"))
	if baseURL == "" {
		baseURL = decisionsBaseURL
	}
	return &DecisionsClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  key,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}, nil
}

// NewDecisionsClientWithHTTP is NewDecisionsClient with a custom HTTP client
// (test harnesses).
func NewDecisionsClientWithHTTP(httpClient *http.Client) (*DecisionsClient, error) {
	c, err := NewDecisionsClient()
	if err != nil {
		return nil, err
	}
	if httpClient != nil {
		c.client = httpClient
	}
	return c, nil
}

// Decide submits one state object with the given questions and returns the
// typed answers. Every question id must be unique.
func (c *DecisionsClient) Decide(ctx context.Context, model string, state map[string]string, questions []DecisionQuestion) (DecisionsResult, error) {
	qmap := make(map[string]decisionsQuestionWire, len(questions))
	for _, q := range questions {
		if _, duplicate := qmap[q.ID]; duplicate {
			return DecisionsResult{}, fmt.Errorf("decisions: duplicate question id %q", q.ID)
		}
		qmap[q.ID] = decisionsQuestionWire{Type: q.Type, Instructions: q.Instructions, Criteria: q.Criteria}
	}
	body, err := json.Marshal(decisionsRequestWire{
		Model:     model,
		State:     state,
		Questions: qmap,
	})
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("decisions: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+decisionsAPIPath, bytes.NewReader(body))
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("decisions: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("decisions: call: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("decisions: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return DecisionsResult{}, fmt.Errorf("decisions: status %d: %s", resp.StatusCode, truncateResponse(raw))
	}
	var payload decisionsResponseWire
	if err := json.Unmarshal(raw, &payload); err != nil {
		return DecisionsResult{}, fmt.Errorf("decisions: decode response: %w", err)
	}
	answers := make(map[string]DecisionAnswer, len(payload.Answers))
	for id, answer := range payload.Answers {
		parsed := DecisionAnswer{Type: answer.Type, Choice: answer.Choice}
		if answer.Noul != nil {
			parsed.Noul = *answer.Noul
		}
		if answer.Confidence != nil {
			parsed.Confidence = *answer.Confidence
		}
		if len(answer.Probabilities) > 0 {
			parsed.Probabilities = answer.Probabilities
		}
		if answer.Score != nil {
			if answer.Score.Position != nil {
				parsed.ScorePosition = *answer.Score.Position
			}
			if answer.Score.Confidence != nil && parsed.Confidence == 0 {
				parsed.Confidence = *answer.Score.Confidence
			}
		}
		answers[id] = parsed
	}
	return DecisionsResult{Answers: answers, Cost: payload.Usage.Cost}, nil
}

// truncateResponse caps an error body so a huge error page cannot flood logs.
func truncateResponse(raw []byte) string {
	const max = 512
	if len(raw) > max {
		return string(raw[:max]) + "…"
	}
	return string(raw)
}

type decisionsRequestWire struct {
	Model     string                           `json:"model"`
	State     map[string]string                `json:"state"`
	Questions map[string]decisionsQuestionWire `json:"questions"`
}

type decisionsQuestionWire struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type decisionsResponseWire struct {
	ID       string                         `json:"id"`
	Model    string                         `json:"model"`
	Provider string                         `json:"provider"`
	Answers  map[string]decisionsAnswerWire `json:"answers"`
	Usage    decisionsUsageWire             `json:"usage"`
}

type decisionsAnswerWire struct {
	Type          string              `json:"type"`
	Choice        string              `json:"choice"`
	Noul          *float64            `json:"noul"`
	Score         *decisionsScoreWire `json:"score"`
	Confidence    *float64            `json:"confidence"`
	Probabilities map[string]float64  `json:"probabilities"`
}

type decisionsScoreWire struct {
	Position      *float64           `json:"position"`
	Confidence    *float64           `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type decisionsUsageWire struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}
