package gitcommits

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yurifrl/cly/pkg/llm"
)

const defaultTimeout = 30 * time.Second

const systemPrompt = `You are a git commit planning assistant. Your job is to split a set of file changes into the smallest sensible set of logical, atomic commits.

GROUPING RULES (priority order):
1. Renames: old path deletion + new path addition = one group
2. Same directory prefix = likely same group
3. Implementation + test files = same group
4. Config + implementation = same group
5. Different types (feat vs fix vs chore vs refactor vs docs) = separate groups

HARD CONSTRAINTS:
- Every file should appear in exactly one group (unless user instructions say otherwise)
- If user instructions say to ignore/skip/exclude files or directories, OMIT those files entirely from the output — do not include them in any group
- Each group gets a conventional commit message (feat:, fix:, chore:, refactor:, docs:, test:, style:, build:, ci:, perf:)
- Use AS FEW groups as possible. One group is fine — even ideal — when all files belong to a single logical change. Only split when files are clearly unrelated (different feature, different type, different scope). Hard ceiling: 5 groups.
- Group titles must be concise conventional commit messages
- EVERY group MUST include a stable "scope" key: a short kebab-case identifier of the area touched (e.g. "graphify-cache", "agent-session", "git-commits-planner", "docs"). Groups across different LLM calls that share the same scope+type WILL be merged into one commit, so use consistent scope names — derive from top-level directory or feature area, never include file counts or batch numbers.

BATCH AWARENESS:
- If the user message says "Batch X of Y" with Y > 1, you are seeing only a SUBSET of a larger changeset. Strongly prefer ONE group for the whole batch. Only split if the batch itself contains clearly unrelated changes. Hard ceiling: 3 groups per batch. A downstream pass merges across batches by scope, so picking broad consistent scopes is more important than splitting.

OUTPUT FORMAT: Respond with ONLY a JSON object (no markdown fences, no explanation):
{
  "groups": [
    {
      "title": "feat: add session management",
      "type": "feat",
      "scope": "agent-session",
      "summary": "Brief explanation of what this group covers",
      "items": [
        { "file": "path/to/file.go" }
      ]
    }
  ]
}`

// RawPlan is the raw JSON plan from the LLM.
type RawPlan struct {
	Groups []RawGroup `json:"groups"`
}

// RawGroup is a single commit group from the LLM plan.
type RawGroup struct {
	Title   string    `json:"title"`
	Type    string    `json:"type"`
	Scope   string    `json:"scope,omitempty"`
	Summary string    `json:"summary"`
	Body    string    `json:"body,omitempty"`
	Items   []RawItem `json:"items"`
}

// RawItem is a file reference in the LLM plan.
type RawItem struct {
	File  string   `json:"file"`
	Hunks []string `json:"hunks,omitempty"` // Hunk IDs for line strategy
}

// PlannerConfig holds configuration for the planner.
type PlannerConfig struct {
	Timeout      time.Duration
	CustomPrompt string
	Strategy     string // "file" or "line"
	MaxGroups    int    // hard cap on final group count (0 = no cap)
}

// PlanSplit sends batches to the LLM in parallel and returns a merged plan.
func PlanSplit(ctx context.Context, batches []Batch, client llm.Client, cfg PlannerConfig) (*RawPlan, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}

	prompt := systemPrompt
	if cfg.Strategy == StrategyLine {
		prompt = lineSystemPrompt
	}

	type batchResult struct {
		plan *RawPlan
		err  error
	}

	results := make([]batchResult, len(batches))
	var wg sync.WaitGroup

	for i, batch := range batches {
		wg.Add(1)
		go func(idx int, b Batch) {
			defer wg.Done()

			batchCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
			defer cancel()

			userMsg := b.Text
			if b.TotalCount > 1 {
				userMsg = fmt.Sprintf("Batch %d of %d — plan groups for ONLY these files:\n\n%s",
					b.Index+1, b.TotalCount, b.Text)
			}
			if cfg.CustomPrompt != "" {
				userMsg = fmt.Sprintf("<user_instructions>\n%s\n</user_instructions>\n\n%s", cfg.CustomPrompt, userMsg)
			}

			resp, err := client.Complete(batchCtx, prompt, []llm.Message{
				{Role: llm.RoleUser, Content: userMsg},
			})
			if err != nil {
				results[idx] = batchResult{err: fmt.Errorf("batch %d: %w", idx+1, err)}
				return
			}

			plan, err := extractPlan(resp)
			if err != nil {
				results[idx] = batchResult{err: fmt.Errorf("batch %d parse: %w", idx+1, err)}
				return
			}

			results[idx] = batchResult{plan: plan}
		}(i, batch)
	}

	wg.Wait()

	// Merge successful results
	merged := &RawPlan{}
	var errors []string
	for _, r := range results {
		if r.err != nil {
			errors = append(errors, r.err.Error())
			continue
		}
		if r.plan != nil {
			merged.Groups = append(merged.Groups, r.plan.Groups...)
		}
	}

	// If all failed, return error for fallback handling
	if len(merged.Groups) == 0 {
		if len(errors) > 0 {
			return nil, fmt.Errorf("all batches failed: %s", strings.Join(errors, "; "))
		}
		return nil, fmt.Errorf("planning produced no groups")
	}

	// Cross-batch consolidation: cheap scope+type merge first, then optional
	// LLM consolidation if still over the cap. Single-batch runs trust the LLM.
	if len(batches) > 1 {
		merged.Groups = mergeByScope(merged.Groups)
		if cfg.MaxGroups > 0 && len(merged.Groups) > cfg.MaxGroups {
			if consolidated, err := consolidateViaLLM(ctx, merged, client, prompt, cfg); err == nil {
				merged = consolidated
			}
		}
	}

	return merged, nil
}

// mergeByScope collapses groups that share the same (type, scope) key.
// This is the cheap, deterministic consolidator: when each batch tags its
// groups with stable scopes (per the system prompt), N batches each emitting
// `chore` + `graphify-cache` collapse into one commit with no extra LLM call.
func mergeByScope(groups []RawGroup) []RawGroup {
	if len(groups) <= 1 {
		return groups
	}
	type key struct{ typ, scope string }
	index := make(map[key]int)
	var out []RawGroup
	for _, g := range groups {
		scope := strings.TrimSpace(g.Scope)
		typ := strings.TrimSpace(g.Type)
		if scope == "" || typ == "" {
			// Can't safely merge without both keys — keep as-is.
			out = append(out, g)
			continue
		}
		k := key{typ, scope}
		if idx, ok := index[k]; ok {
			out[idx].Items = append(out[idx].Items, g.Items...)
			if out[idx].Summary != "" && g.Summary != "" && !strings.Contains(out[idx].Summary, g.Summary) {
				out[idx].Summary = out[idx].Summary + "; " + g.Summary
			}
			continue
		}
		index[k] = len(out)
		out = append(out, g)
	}
	return out
}

const consolidatorPrompt = `You are a git commit consolidation assistant. You will receive a list of proposed commit groups produced by independent planning passes over a single changeset. Your job is to merge them into AT MOST the requested number of groups.

MERGE RULES:
- Prefer merging groups with the same conventional-commit type and the same or related scope.
- Keep distinct types separate when possible (don't fold feat into chore).
- Preserve every file: every "file" entry in the input MUST appear in exactly one output group.
- Output titles must remain concise conventional commit messages.
- Output every group with a stable kebab-case "scope" field.

OUTPUT FORMAT: Respond with ONLY a JSON object (no markdown fences, no explanation):
{
  "groups": [
    { "title": "...", "type": "...", "scope": "...", "summary": "...", "items": [{"file": "..."}] }
  ]
}`

// consolidateViaLLM asks the model to fold an over-cap plan into <= MaxGroups.
// Falls back to the original plan on any error so we never make things worse.
func consolidateViaLLM(ctx context.Context, plan *RawPlan, client llm.Client, _ string, cfg PlannerConfig) (*RawPlan, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("marshal plan: %w", err)
	}

	userMsg := fmt.Sprintf("Consolidate the following %d groups into AT MOST %d groups. Preserve every file.\n\n%s",
		len(plan.Groups), cfg.MaxGroups, string(payload))
	if cfg.CustomPrompt != "" {
		userMsg = fmt.Sprintf("<user_instructions>\n%s\n</user_instructions>\n\n%s", cfg.CustomPrompt, userMsg)
	}

	resp, err := client.Complete(cctx, consolidatorPrompt, []llm.Message{
		{Role: llm.RoleUser, Content: userMsg},
	})
	if err != nil {
		return nil, fmt.Errorf("consolidator call: %w", err)
	}

	out, err := extractPlan(resp)
	if err != nil {
		return nil, fmt.Errorf("consolidator parse: %w", err)
	}
	return out, nil
}

// GenerateFallbackMessage creates a single conventional commit message.
func GenerateFallbackMessage(ctx context.Context, cs *Changeset, client llm.Client, timeout time.Duration) (*RawPlan, error) {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	fallbackCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var fileList strings.Builder
	for _, f := range cs.Files {
		fileList.WriteString(fmt.Sprintf("  %s %s\n", f.Status, f.Path))
	}

	resp, err := client.Complete(fallbackCtx,
		"Generate a single conventional commit message (e.g. feat:, fix:, chore:) for these changes. Respond with ONLY the commit message, no explanation.",
		[]llm.Message{
			{Role: llm.RoleUser, Content: fmt.Sprintf("Changed files:\n%s", fileList.String())},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("fallback message generation failed: %w", err)
	}

	title := strings.TrimSpace(resp)
	// Take only first line
	if idx := strings.Index(title, "\n"); idx >= 0 {
		title = title[:idx]
	}

	// Build a single-group plan with all files
	items := make([]RawItem, len(cs.Files))
	for i, f := range cs.Files {
		items[i] = RawItem{File: f.Path}
	}

	return &RawPlan{
		Groups: []RawGroup{
			{
				Title:   title,
				Type:    inferType(title),
				Summary: "All changes in a single commit (fallback)",
				Items:   items,
			},
		},
	}, nil
}

// extractPlan extracts JSON from an LLM response, handling markdown fences.
func extractPlan(response string) (*RawPlan, error) {
	response = strings.TrimSpace(response)

	// Strip markdown code fences if present
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		// Remove first and last fence lines
		start := 1
		end := len(lines)
		if end > 0 && strings.HasPrefix(lines[end-1], "```") {
			end--
		}
		response = strings.Join(lines[start:end], "\n")
		response = strings.TrimSpace(response)
	}

	// Find outermost {}
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")
	if startIdx < 0 || endIdx < 0 || endIdx <= startIdx {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	jsonStr := response[startIdx : endIdx+1]

	var plan RawPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if len(plan.Groups) == 0 {
		return nil, fmt.Errorf("plan has no groups")
	}

	return &plan, nil
}

// inferType extracts the conventional commit type from a title.
func inferType(title string) string {
	if idx := strings.Index(title, ":"); idx > 0 {
		t := strings.TrimSpace(title[:idx])
		// Handle scoped types like "feat(api)"
		if paren := strings.Index(t, "("); paren > 0 {
			t = t[:paren]
		}
		return t
	}
	return "chore"
}

// ScaleMaxTokens returns appropriate max output tokens based on analysis text size.
func ScaleMaxTokens(analysisSize int) int {
	switch {
	case analysisSize < 8000:
		return 4000
	case analysisSize <= 20000:
		return 8000
	default:
		return 16000
	}
}
