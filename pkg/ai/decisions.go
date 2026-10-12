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

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

// decisions.go — Decisions API client against the AIHub gateway, the System
// One surface for the typesafe/jev-* decision models (served as
// aihub/jev-latest). This is its own wire contract (POST /api/alpha/decisions):
// a state object plus typed questions in, typed answers with probabilities
// out — no free-form text to parse. The client is domain-blind: it knows the
// HTTP surface only; modules own how they phrase questions.
//
// Endpoints come from the ai.providers entries with `type: jev`; model,
// base_url, and key all live there. The client fails over across the pool
// and never falls back to chat entries.

const (
	decisionsAPIPath = "/api/alpha/decisions"
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

// decisionsEndpoint is one jev-pool entry ready to call.
type decisionsEndpoint struct {
	name    string
	model   string
	baseURL string
	apiKey  string
}

// DecisionsClient calls the Decisions API across the configured jev
// provider pool. Safe for concurrent use.
type DecisionsClient struct {
	endpoints []decisionsEndpoint
	client    *http.Client
}

// decisionsEndpoints resolves entries into callable jev endpoints. Entries
// without a model, base_url, or usable key are skipped with a warning so one
// broken entry can't take the pool down.
func decisionsEndpoints(entries []Entry, ctx *Context) []decisionsEndpoint {
	eps := make([]decisionsEndpoint, 0, len(entries))
	for _, entry := range orderProviders(providersOfType(entries, TypeJev), ctx) {
		if entry.Model == "" {
			fmt.Fprintf(os.Stderr, "ai: decisions: skipping jev provider %q: no model\n", entry.Name)
			continue
		}
		if entry.BaseURL == "" {
			fmt.Fprintf(os.Stderr, "ai: decisions: skipping jev provider %q: no base_url\n", entry.Name)
			continue
		}
		key := entry.APIKey
		if key == "" && entry.APIKeyEnv != "" {
			key = os.Getenv(entry.APIKeyEnv)
		}
		if key == "" {
			fmt.Fprintf(os.Stderr, "ai: decisions: skipping jev provider %q: no api key\n", entry.Name)
			continue
		}
		eps = append(eps, decisionsEndpoint{
			name:    entry.Name,
			model:   entry.Model,
			baseURL: strings.TrimRight(entry.BaseURL, "/"),
			apiKey:  key,
		})
	}
	return eps
}

// jevEndpoints is the config-facing wrapper: parse the ai: block, resolve
// the jev pool.
func jevEndpoints() []decisionsEndpoint {
	entries, err := parseProviders(pkgconfig.Get().AI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai: decisions: bad provider config: %v\n", err)
		return nil
	}
	return decisionsEndpoints(entries, buildContext())
}

// NewDecisionsClient builds the client from the `type: jev` provider pool.
// It errors when no jev entry yields a usable endpoint so callers can skip
// wiring (the feature stays off) instead of failing on every call.
func NewDecisionsClient() (*DecisionsClient, error) {
	return newDecisionsClient(jevEndpoints(), &http.Client{Timeout: 5 * time.Minute})
}

// newDecisionsClient is the testable constructor.
func newDecisionsClient(eps []decisionsEndpoint, httpClient *http.Client) (*DecisionsClient, error) {
	if len(eps) == 0 {
		return nil, errors.New("ai: no jev providers configured (add an ai.providers entry with type: jev)")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	}
	return &DecisionsClient{endpoints: eps, client: httpClient}, nil
}

// Decide submits one state object with the given questions and returns the
// typed answers. Every question id must be unique. The model comes from the
// winning jev entry's config; on failure the next jev endpoint is tried.
func (c *DecisionsClient) Decide(ctx context.Context, state map[string]string, questions []DecisionQuestion) (DecisionsResult, error) {
	qmap, err := encodeQuestions(questions)
	if err != nil {
		return DecisionsResult{}, err
	}
	var lastErr error
	for i, ep := range c.endpoints {
		res, err := c.decideEndpoint(ctx, ep, qmap, state)
		if err == nil {
			return res, nil
		}
		if ctx.Err() != nil {
			return DecisionsResult{}, fmt.Errorf("decisions: call %s: %w", ep.name, ctx.Err())
		}
		lastErr = err
		if i < len(c.endpoints)-1 {
			fmt.Fprintf(os.Stderr, "ai: decisions via %s failed; trying next jev provider: %v\n", ep.name, err)
		}
	}
	return DecisionsResult{}, lastErr
}

// encodeQuestions validates ids and builds the wire shape.
func encodeQuestions(questions []DecisionQuestion) (map[string]decisionsQuestionWire, error) {
	qmap := make(map[string]decisionsQuestionWire, len(questions))
	for _, q := range questions {
		if _, duplicate := qmap[q.ID]; duplicate {
			return nil, fmt.Errorf("decisions: duplicate question id %q", q.ID)
		}
		qmap[q.ID] = decisionsQuestionWire{Type: q.Type, Instructions: q.Instructions, Criteria: q.Criteria}
	}
	return qmap, nil
}

// decideEndpoint is one attempt against one endpoint.
func (c *DecisionsClient) decideEndpoint(ctx context.Context, ep decisionsEndpoint, qmap map[string]decisionsQuestionWire, state map[string]string) (DecisionsResult, error) {
	body, err := json.Marshal(decisionsRequestWire{
		Model:     ep.model,
		State:     state,
		Questions: qmap,
	})
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("decisions: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.baseURL+decisionsAPIPath, bytes.NewReader(body))
	if err != nil {
		return DecisionsResult{}, fmt.Errorf("decisions: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+ep.apiKey)
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
			switch s := any(answer.Score).(type) {
			case float64:
				// Observed live: the gateway returns score as a bare
				// number (often 0 on non-score answers).
				parsed.ScorePosition = s
			case map[string]any:
				if p, ok := s["position"].(float64); ok {
					parsed.ScorePosition = p
				}
				if c, ok := s["confidence"].(float64); ok && parsed.Confidence == 0 {
					parsed.Confidence = c
				}
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
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Noul          *float64           `json:"noul"`
	Score         any                `json:"score"` // number, or object with position/confidence
	Confidence    *float64           `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type decisionsUsageWire struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}
