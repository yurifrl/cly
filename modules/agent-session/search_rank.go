package agentsession

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yurifrl/cly/pkg/ai"
	"github.com/yurifrl/cly/pkg/llm"
)

// candidate is the unit a ranker (local or AI) produces.
type candidate struct {
	Session *indexedSession
	Score   float64
	Hits    int    // number of query occurrences across name + description + body
	Snippet string // context snippet centered on the first body hit (live grep)
	Why     string // AI rationale (only set by rerankAI)
	Source  string // "local", "live", or "ai"
}

// SortMode controls how rankLocal orders candidates after scoring/filtering.
type SortMode int

const (
	SortByDate SortMode = iota // newest jsonl mtime first (default)
	SortByHits                 // most query occurrences first; date breaks ties
	SortByName                 // alphabetic by session name
	SortByPath                 // alphabetic by working directory path
)

func (m SortMode) Label() string {
	switch m {
	case SortByHits:
		return "hits"
	case SortByName:
		return "name"
	case SortByPath:
		return "path"
	default:
		return "date"
	}
}

func (m SortMode) Next() SortMode {
	return (m + 1) % 4
}

// rankLocal applies a cheap substring + fuzzy score across cached metadata
// and body excerpts. It is always runnable and never makes network calls.
// `sort` controls the final ordering; relevance still drives FILTERING (a
// non-empty query keeps only candidates with at least one hit) but not
// necessarily the order the user sees.
func rankLocal(idx *searchIndex, query string, providerFilter string, sort SortMode) []candidate {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []candidate
	for _, s := range idx.Sessions {
		if providerFilter != "" && providerFilter != "all" && s.Provider != providerFilter {
			continue
		}
		score, hits := scoreSession(s, q)
		if q != "" && hits == 0 {
			continue
		}
		out = append(out, candidate{Session: s, Score: score, Hits: hits, Source: "local"})
	}
	sortCandidates(out, sort)
	return out
}

func sortCandidates(out []candidate, mode SortMode) {
	switch mode {
	case SortByHits:
		sortStable(out, func(a, b candidate) bool {
			if a.Score != b.Score {
				return a.Score > b.Score
			}
			return sessionDate(a.Session).After(sessionDate(b.Session))
		})
	case SortByName:
		sortStable(out, func(a, b candidate) bool {
			return strings.ToLower(a.Session.Name) < strings.ToLower(b.Session.Name)
		})
	case SortByPath:
		sortStable(out, func(a, b candidate) bool {
			return strings.ToLower(a.Session.Path) < strings.ToLower(b.Session.Path)
		})
	default: // SortByDate
		sortStable(out, func(a, b candidate) bool {
			return sessionDate(a.Session).After(sessionDate(b.Session))
		})
	}
}

func sortStable(s []candidate, less func(a, b candidate) bool) {
	sort.SliceStable(s, func(i, j int) bool { return less(s[i], s[j]) })
}

// sessionDate falls back to the jsonl mtime when the catalog SavedAt is
// zero. Sessions that were never `/checkpoint`-ed have no catalog entry,
// so SavedAt is the Go zero value ("0001-01-01"). Showing that to the user
// is meaningless; the .jsonl mtime is the real "when did this happen".
func sessionDate(s *indexedSession) time.Time {
	if !s.SavedAt.IsZero() {
		return s.SavedAt
	}
	return s.JsonlMtime
}

// countAll counts case-insensitive non-overlapping occurrences of `q` in s.
func countAll(s, q string) int {
	if q == "" || s == "" {
		return 0
	}
	return strings.Count(strings.ToLower(s), q)
}

func scoreSession(s *indexedSession, q string) (float64, int) {
	if q == "" {
		return 0.001, 0
	}
	hitsName := countAll(s.Name, q)
	hitsDesc := countAll(s.Description, q)
	hitsPath := countAll(s.Path, q)
	hitsFirst := countAll(s.FirstUserMsg, q)
	hitsBody := countAll(s.SearchableText, q)
	total := hitsName + hitsDesc + hitsPath + hitsFirst + hitsBody
	score := float64(hitsName)*3 + float64(hitsDesc)*2 + float64(hitsPath)*1.5 +
		float64(hitsFirst)*1.5 + float64(hitsBody)*1.0

	tokens := strings.Fields(q)
	if len(tokens) > 1 {
		all := strings.ToLower(s.Name + " " + s.Description + " " + s.SearchableText)
		hits := 0
		for _, t := range tokens {
			if t == "" {
				continue
			}
			if strings.Contains(all, t) {
				hits++
			}
		}
		score += float64(hits) * 0.5
	}
	return score, total
}

// rerankPayloadCap is the hard upper bound on the JSON payload sent to the
// re-rank LLM. When the candidate list would exceed this, the builder
// shrinks each per-candidate snippet first; if still too big, it drops
// the lowest-scoring candidates.
const rerankPayloadCap = 30 * 1024

type aiCandidate struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Date        string `json:"date,omitempty"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
}

type aiRerankRequest struct {
	Query      string        `json:"query"`
	Candidates []aiCandidate `json:"candidates"`
}

type aiRerankResponse struct {
	Ranked []struct {
		ID    string  `json:"id"`
		Score float64 `json:"score"`
		Why   string  `json:"why"`
	} `json:"ranked"`
}

// buildPayload constructs the AI rerank JSON for a candidate slice while
// keeping the marshalled size ≤ rerankPayloadCap. Strategy:
//  1. Start with full snippets (≤ 600 bytes each).
//  2. If too big, halve snippet budget repeatedly down to 100 bytes.
//  3. If still too big, drop tail candidates until it fits.
func buildPayload(query string, cands []candidate) ([]byte, []candidate) {
	const baseSnippetMax = 600
	snippet := baseSnippetMax
	for {
		req := aiRerankRequest{Query: query}
		for _, c := range cands {
			s := c.Session
			req.Candidates = append(req.Candidates, aiCandidate{
				ID:          s.ID,
				Name:        s.Name,
				Date:        s.SavedAt.Format("2006-01-02"),
				Path:        s.Path,
				Description: truncateText(s.Description, 200),
				Snippet:     truncateText(firstNonEmpty(s.FirstUserMsg, s.SearchableText), snippet),
			})
		}
		data, err := json.Marshal(req)
		if err != nil {
			return nil, cands
		}
		if len(data) <= rerankPayloadCap {
			return data, cands
		}
		if snippet > 100 {
			snippet /= 2
			if snippet < 100 {
				snippet = 100
			}
			continue
		}
		// Snippet floor reached; drop the worst-scoring candidate.
		if len(cands) <= 1 {
			return data, cands
		}
		cands = cands[:len(cands)-1]
	}
}

func firstNonEmpty(parts ...string) string {
	for _, p := range parts {
		if p != "" {
			return p
		}
	}
	return ""
}

// rerankAI calls the configured LLM to reorder candidates. Returns the
// reranked slice (or the original on any error). Never sends full session
// content; only the bounded payload from buildPayload.
func rerankAI(ctx context.Context, query string, cands []candidate) ([]candidate, error) {
	if len(cands) == 0 {
		return cands, nil
	}
	payload, kept := buildPayload(query, cands)
	if len(kept) == 0 {
		return cands, nil
	}
	client, err := ai.NewClientFor("agent_session.search")
	if err != nil {
		return cands, err
	}
	if client == nil {
		return cands, ai.ErrDisabled
	}
	system := "You re-rank a list of past coding/agent sessions for a user search query. " +
		"Return ONLY a JSON object with shape {\"ranked\":[{\"id\":\"...\",\"score\":0.0-1.0,\"why\":\"<one sentence>\"}]}. " +
		"Use only IDs from the input list. Order from best to worst match. Higher score = better."
	user := llm.Message{Role: llm.RoleUser, Content: string(payload)}

	tctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	resp, err := client.Complete(tctx, system, []llm.Message{user})
	if err != nil {
		return cands, err
	}

	var parsed aiRerankResponse
	if jerr := json.Unmarshal([]byte(extractJSON(resp)), &parsed); jerr != nil {
		return cands, fmt.Errorf("parse ai rerank response: %w", jerr)
	}

	byID := make(map[string]candidate, len(cands))
	for _, c := range cands {
		byID[c.Session.ID] = c
	}
	var out []candidate
	used := make(map[string]bool, len(parsed.Ranked))
	for _, r := range parsed.Ranked {
		if c, ok := byID[r.ID]; ok && !used[r.ID] {
			c.Score = r.Score
			c.Why = r.Why
			c.Source = "ai"
			out = append(out, c)
			used[r.ID] = true
		}
	}
	// Append any local candidates the model omitted, ordered by local score.
	for _, c := range cands {
		if !used[c.Session.ID] {
			out = append(out, c)
		}
	}
	return out, nil
}

// extractJSON pulls the first balanced JSON object out of a string. LLMs
// occasionally wrap responses in prose; this is defensive.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if esc {
			esc = false
			continue
		}
		if inStr {
			if c == 0x5C { // backslash
				esc = true
				continue
			}
			if c == 0x22 { // double quote
				inStr = false
			}
			continue
		}
		switch c {
		case 0x22:
			inStr = true
		case 0x7B:
			depth++
		case 0x7D:
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s
}

// loadSearchAIConfig was a translation shim from `ai.Resolved` back to
// `llm.Config`. Removed: callers now use `ai.NewClient` and `ai.HasAPIKey`
// directly so we don't have two parallel ways to express the same thing.

