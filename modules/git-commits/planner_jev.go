package gitcommits

import (
	"context"
	"fmt"
	"time"

	"github.com/yurifrl/cly/pkg/ai"
)

// Planner modes.
const (
	PlannerChat = "chat"
	PlannerJev  = "jev"
)

const (
	// jevModelDefault is the AIHub System One model id used for bundling votes.
	jevModelDefault = "aihub/jev-latest"
	// jevOwnOption is the virtual vote meaning "this bundle is its own commit".
	jevOwnOption = "own"
	// jevChunkSize bounds seeds per Decisions call; typical changesets fit in one.
	jevChunkSize = 40
)

// JevPlannerConfig configures the jev bundling pass.
type JevPlannerConfig struct {
	// Model is the AIHub decisions model id (default aihub/jev-latest).
	Model string
	// Timeout bounds each Decisions call.
	Timeout time.Duration
	// Feedback is optional user revision text from the preview loop.
	Feedback string
}

// decider is the subset of the Decisions API client the jev planner needs.
type decider interface {
	Decide(ctx context.Context, model string, state map[string]string, questions []ai.DecisionQuestion) (ai.DecisionsResult, error)
}

// PlanJev bundles files into affinity groups by asking the jev decision model
// to vote on heuristic seed bundles (see buildSeeds). Files are NOT sent to
// the model — only compact bundle labels — so the pass is cheap and fast.
// Message text (title/type/scope/summary) is deliberately left empty; the
// normal chat AI fills it afterwards (FillMessages).
func PlanJev(ctx context.Context, files []FileChange, cfg JevPlannerConfig) (*RawPlan, error) {
	client, err := ai.NewDecisionsClient()
	if err != nil {
		return nil, err
	}
	return planJevWith(ctx, files, cfg, client)
}

func planJevWith(ctx context.Context, files []FileChange, cfg JevPlannerConfig, client decider) (*RawPlan, error) {
	if cfg.Model == "" {
		cfg.Model = jevModelDefault
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}

	seeds := buildSeeds(files)
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no files to group")
	}

	parent := make([]int, len(seeds))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			if ra < rb {
				parent[rb] = ra
			} else {
				parent[ra] = rb
			}
		}
	}

	// Vote in chunks so huge changesets stay bounded per call. Cross-chunk
	// affinity is intentionally not merged, mirroring the chat planner's
	// batch-boundary behavior. A single seed can only form one commit, so
	// skip the votes entirely.
	for start := 0; len(seeds) > 1 && start < len(seeds); start += jevChunkSize {
		end := min(start+jevChunkSize, len(seeds))
		edges, err := voteChunk(ctx, client, cfg, seeds[start:end], start)
		if err != nil {
			return nil, err
		}
		for _, e := range edges {
			union(e[0], e[1])
		}
	}

	groups := make([]RawGroup, 0, len(seeds))
	byRoot := make(map[int]int) // find(i) -> group index
	for i, s := range seeds {
		root := find(i)
		gi, ok := byRoot[root]
		if !ok {
			gi = len(groups)
			byRoot[root] = gi
			groups = append(groups, RawGroup{})
		}
		g := &groups[gi]
		for _, f := range s.Files {
			g.Items = append(g.Items, RawItem{File: f.Path})
		}
	}

	return &RawPlan{Groups: groups}, nil
}

// voteChunk asks jev, for each seed in the chunk, which other seed most
// belongs in the same commit (or "own"). Returns symmetric vote pairs as
// seed indices. Chunked calls run sequentially: they are cheap and the
// union-find merge is order-independent, so extra concurrency buys nothing.
func voteChunk(ctx context.Context, client decider, cfg JevPlannerConfig, seeds []Seed, offset int) ([][2]int, error) {
	state := make(map[string]string, len(seeds))
	questions := make([]ai.DecisionQuestion, 0, len(seeds))
	index := make(map[string]int, len(seeds)) // question id -> seed index
	for i, s := range seeds {
		id := fmt.Sprintf("s%d", offset+i)
		state[id] = s.Label
		index[id] = offset + i

		options := make([]string, 0, len(seeds))
		for j := range seeds {
			if j == i {
				continue
			}
			options = append(options, fmt.Sprintf("s%d", offset+j))
		}
		options = append(options, jevOwnOption)

		instructions := "You are deciding how to bundle changed files into git commits. " +
			"Each option names another bundle of changed files; their contents are described in the state. " +
			"Choose the ONE bundle that most clearly belongs in the same commit as the one you are voting for, " +
			"or \"" + jevOwnOption + "\" if it should be a separate commit. " +
			"Bundle together when changes serve one purpose (implementation + its tests, config + its feature, rename pairs). " +
			"Keep unrelated purposes (different feature, fix vs docs vs build) in separate commits."
		if cfg.Feedback != "" {
			instructions += " The user also directed: " + cfg.Feedback
		}

		questions = append(questions, ai.DecisionQuestion{
			ID:           id,
			Type:         "choice",
			Instructions: instructions,
			Criteria:     options,
		})
	}

	result, err := client.Decide(ctx, cfg.Model, state, questions)
	if err != nil {
		return nil, fmt.Errorf("jev vote failed: %w", err)
	}

	// Symmetric votes union the pair; one-sided votes are ignored.
	votes := make(map[int]string, len(seeds))
	for id, ans := range result.Answers {
		if idx, ok := index[id]; ok && ans.Choice != "" {
			votes[idx] = ans.Choice
		}
	}
	var edges [][2]int
	for from, choice := range votes {
		if choice == jevOwnOption || choice == "" {
			continue
		}
		to, ok := index[choice]
		if !ok {
			continue
		}
		if v, voted := votes[to]; voted && v == fmt.Sprintf("s%d", from) {
			edges = append(edges, [2]int{from, to})
		}
	}
	return edges, nil
}
