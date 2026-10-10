package gitcommits

import (
	"context"
	"strings"
	"testing"

	"github.com/yurifrl/cly/pkg/ai"
)

// stubDecider records questions and answers them via a fixed table.
type stubDecider struct {
	answers map[string]string // question id -> chosen option
	gotCfg  *ai.DecisionsResult
}

func (s *stubDecider) Decide(_ context.Context, _ string, _ map[string]string, questions []ai.DecisionQuestion) (ai.DecisionsResult, error) {
	ans := map[string]ai.DecisionAnswer{}
	for _, q := range questions {
		choice := s.answers[q.ID]
		if choice == "" {
			choice = jevOwnOption
		}
		ans[q.ID] = ai.DecisionAnswer{Type: "choice", Choice: choice}
	}
	return ai.DecisionsResult{Answers: ans}, nil
}

func TestPlanJevSymmetricVotesMerge(t *testing.T) {
	files := []FileChange{
		{Path: "pkg/a/x.go", Status: StatusModified},
		{Path: "pkg/a/x_test.go", Status: StatusModified},
		{Path: "modules/git-commits/foo.go", Status: StatusModified},
		{Path: "docs/readme.md", Status: StatusModified},
	}
	// s0 = pkg/a (2 files), s1 = modules/git-commits, s2 = docs.
	// s0<->s1 vote for each other -> merged group; s2 stays own.
	stub := &stubDecider{answers: map[string]string{"s0": "s1", "s1": "s0", "s2": jevOwnOption}}
	plan, err := planJevWith(context.Background(), files, JevPlannerConfig{}, stub)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Groups) != 2 {
		t.Fatalf("groups = %d, want 2: %+v", len(plan.Groups), plan.Groups)
	}
	merged := plan.Groups[0]
	if len(merged.Items) != 3 {
		t.Errorf("merged group items = %+v", merged.Items)
	}
	own := plan.Groups[1]
	if len(own.Items) != 1 || own.Items[0].File != "docs/readme.md" {
		t.Errorf("own group items = %+v", own.Items)
	}
}

func TestPlanJevAsymmetricVoteIgnored(t *testing.T) {
	files := []FileChange{
		{Path: "pkg/a/x.go", Status: StatusModified},
		{Path: "pkg/b/y.go", Status: StatusModified},
	}
	stub := &stubDecider{answers: map[string]string{"s0": "s1", "s1": jevOwnOption}}
	plan, err := planJevWith(context.Background(), files, JevPlannerConfig{}, stub)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Groups) != 2 {
		t.Fatalf("groups = %d, want 2 (one-sided vote must not merge)", len(plan.Groups))
	}
}

func TestPlanJevFeedbackInInstructions(t *testing.T) {
	files := []FileChange{
		{Path: "a.go", Status: StatusModified},
		{Path: "docs/guide.md", Status: StatusModified},
	}
	var captured string
	stub := &feedbackDecider{capture: &captured}
	_, err := planJevWith(context.Background(), files, JevPlannerConfig{Feedback: "keep docs separate"}, stub)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(captured, "keep docs separate") {
		t.Errorf("instructions missing feedback: %q", captured)
	}
}

type feedbackDecider struct {
	capture *string
}

func (f *feedbackDecider) Decide(_ context.Context, _ string, _ map[string]string, questions []ai.DecisionQuestion) (ai.DecisionsResult, error) {
	if len(questions) > 0 {
		*f.capture = questions[0].Instructions
	}
	return ai.DecisionsResult{Answers: map[string]ai.DecisionAnswer{
		"s0": {Type: "choice", Choice: jevOwnOption},
	}}, nil
}

func TestPlanJevCoversEveryFileOnce(t *testing.T) {
	files := []FileChange{
		{Path: "m/a.go", Status: StatusAdded},
		{Path: "m/b.go", Status: StatusModified},
		{Path: "n/c.go", Status: StatusDeleted},
		{Path: "n/d.go", Status: StatusRenamed, OldPath: "n/old.go"},
	}
	stub := &stubDecider{answers: map[string]string{"s0": "s1", "s1": "s0"}}
	plan, err := planJevWith(context.Background(), files, JevPlannerConfig{}, stub)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(plan.Groups))
	}
	seen := map[string]int{}
	for _, g := range plan.Groups {
		for _, it := range g.Items {
			seen[it.File]++
		}
	}
	if len(seen) != 4 {
		t.Fatalf("files covered = %v, want 4", seen)
	}
	for p, n := range seen {
		if n != 1 {
			t.Errorf("file %s appears %d times", p, n)
		}
	}
}

func TestPlanJevEmpty(t *testing.T) {
	if _, err := planJevWith(context.Background(), nil, JevPlannerConfig{}, &stubDecider{}); err == nil {
		t.Fatal("expected error for empty changeset")
	}
}

func TestPlanJevSingleSeedSkipsVotes(t *testing.T) {
	// One area = one seed: there is nothing to bundle, so the decider must
	// never be called (fails loudly if it is).
	stub := &failingDecider{}
	files := []FileChange{
		{Path: "pkg/a/x.go", Status: StatusModified},
		{Path: "pkg/a/x_test.go", Status: StatusAdded},
	}
	plan, err := planJevWith(context.Background(), files, JevPlannerConfig{}, stub)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Groups) != 1 || len(plan.Groups[0].Items) != 2 {
		t.Fatalf("groups = %+v, want 1 group with 2 files", plan.Groups)
	}
}

// failingDecider panics if consulted; used to prove vote-free paths.
type failingDecider struct{}

func (failingDecider) Decide(context.Context, string, map[string]string, []ai.DecisionQuestion) (ai.DecisionsResult, error) {
	panic("decider called")
}
