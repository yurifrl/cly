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

func (s *stubDecider) Decide(_ context.Context, _ map[string]string, questions []ai.DecisionQuestion) (ai.DecisionsResult, error) {
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

func (f *feedbackDecider) Decide(_ context.Context, _ map[string]string, questions []ai.DecisionQuestion) (ai.DecisionsResult, error) {
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

func (failingDecider) Decide(context.Context, map[string]string, []ai.DecisionQuestion) (ai.DecisionsResult, error) {
	panic("decider called")
}

type questionsDecider struct {
	qs *[]ai.DecisionQuestion
}

func (q *questionsDecider) Decide(_ context.Context, _ map[string]string, questions []ai.DecisionQuestion) (ai.DecisionsResult, error) {
	*q.qs = questions
	return ai.DecisionsResult{Answers: map[string]ai.DecisionAnswer{
		"s0": {Type: "choice", Choice: jevOwnOption},
		"s1": {Type: "choice", Choice: jevOwnOption},
	}}, nil
}

func TestPlanJevCriteriaAreIdDescriptionMaps(t *testing.T) {
	files := []FileChange{
		{Path: "pkg/a/x.go", Status: StatusModified},
		{Path: "pkg/a/x_test.go", Status: StatusModified},
		{Path: "docs/readme.md", Status: StatusModified},
	}
	var got []ai.DecisionQuestion
	if _, err := planJevWith(context.Background(), files, JevPlannerConfig{}, &questionsDecider{qs: &got}); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no questions sent")
	}
	// The upstream rejects bare id lists with an opaque 400; every choice
	// question must carry criteria as an id-to-description map.
	crit, ok := got[0].Criteria.(map[string]string)
	if !ok {
		t.Fatalf("criteria type = %T, want map[string]string", got[0].Criteria)
	}
	if _, ok := crit["s1"]; !ok {
		t.Errorf("neighbor option s1 missing: %v", crit)
	}
	if crit["own"] == "" {
		t.Errorf("own option missing description: %v", crit)
	}
}

// Regression: jev bundling must never merge the .claude→.agents skill
// migration with unrelated cleanups; unpaired deletions from the same
// area join their move. The planner itself (buildSeeds) owns this
// grouping; jev votes only ever merge across those bundles.
func TestPlanJevSeedBundlingIsolatesMigrationFromUnrelatedAreas(t *testing.T) {
	files := []FileChange{
		{Path: ".agents/skills/a/SKILL.md", OldPath: ".claude/skills/a/SKILL.md", Status: StatusRenamed},
		{Path: ".agents/skills/b/SKILL.md", OldPath: ".claude/skills/b/SKILL.md", Status: StatusRenamed},
		{Path: ".claude/CLAUDE.md", Status: StatusDeleted}, // unpaired: joins the move
		{Path: ".beads/hooks/x", Status: StatusDeleted},    // unrelated cleanup
		{Path: "modules/git-commits/seeds.go", Status: StatusModified},
	}
	plan, err := PlanSeeds(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Groups) < 3 {
		t.Fatalf("groups = %d, want >= 3 (migration, .beads, modules): %+v", len(plan.Groups), plan.Groups)
	}

	var migration, beads, mods []string
	for _, g := range plan.Groups {
		for _, it := range g.Items {
			switch {
			case strings.HasPrefix(it.File, ".agents/") || it.File == ".claude/CLAUDE.md":
				migration = append(migration, it.File)
			case strings.HasPrefix(it.File, ".beads/"):
				beads = append(beads, it.File)
			case strings.HasPrefix(it.File, "modules/"):
				mods = append(mods, it.File)
			}
		}
		if g.Title != "" {
			t.Errorf("group title pre-filled: %q — FillMessages owns titles", g.Title)
		}
	}
	if len(migration) != 3 || len(beads) != 1 || len(mods) != 1 {
		t.Errorf("split = migration:%v beads:%v modules:%v", migration, beads, mods)
	}
}
