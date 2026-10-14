package gitcommits

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yurifrl/cly/pkg/llm"
)

type stubMsgClient struct {
	resp string
	err  error
	seen []llm.Message
}

func (s *stubMsgClient) Complete(ctx context.Context, system string, messages []llm.Message) (string, error) {
	s.seen = append(s.seen, messages...)
	return s.resp, s.err
}

func (s *stubMsgClient) Stream(ctx context.Context, systemPrompt string, messages []llm.Message) (<-chan llm.StreamChunk, error) {
	return nil, fmt.Errorf("stream not expected in tests")
}

func msgItems(paths ...string) []RawItem {
	items := make([]RawItem, len(paths))
	for i, p := range paths {
		items[i] = RawItem{File: p}
	}
	return items
}

func TestFillMessagesFillsBareGroups(t *testing.T) {
	plan := &RawPlan{Groups: []RawGroup{
		{Type: "feat", Scope: "ai", Items: []RawItem{{File: "pkg/ai/x.go"}}},
		{Type: "fix", Items: []RawItem{{File: "modules/git-commits/y.go"}}},
	}}
	stub := &stubMsgClient{resp: `{"groups":[
		{"id":1,"title":"feat(ai): add decisions client","type":"feat","scope":"ai","summary":"connects the jev decision surface"},
		{"id":2,"title":"fix(planner): drop dead rename pairing","type":"fix","scope":"planner","summary":"renames arrive pre-paired"}
	]}`}
	if err := FillMessages(context.Background(), plan, stub, time.Second, ""); err != nil {
		t.Fatal(err)
	}
	if plan.Groups[0].Title != "feat(ai): add decisions client" || plan.Groups[0].Summary == "" {
		t.Errorf("group 0 = %+v", plan.Groups[0])
	}
	if plan.Groups[1].Title != "fix(planner): drop dead rename pairing" {
		t.Errorf("group 1 = %+v", plan.Groups[1])
	}
}

func TestFillMessagesSkipsTitledGroups(t *testing.T) {
	plan := &RawPlan{Groups: []RawGroup{
		{Title: "chore: already named", Items: []RawItem{{File: "a.go"}}},
	}}
	stub := &stubMsgClient{}
	if err := FillMessages(context.Background(), plan, stub, time.Second, ""); err != nil {
		t.Fatal(err)
	}
	if len(stub.seen) != 0 {
		t.Errorf("client called for fully-titled plan")
	}
	if plan.Groups[0].Title != "chore: already named" {
		t.Errorf("title clobbered: %q", plan.Groups[0].Title)
	}
}

func TestFillMessagesUncoveredGroupAborts(t *testing.T) {
	plan := &RawPlan{Groups: []RawGroup{
		{Type: "docs", Scope: "guide", Items: []RawItem{{File: "docs/g.md"}}},
		{Items: []RawItem{{File: "x.go"}}},
	}}
	stub := &stubMsgClient{resp: `{"groups":[{"id":2,"title":"fix: handle empty input"}]}`}
	if err := FillMessages(context.Background(), plan, stub, time.Second, ""); err == nil {
		t.Fatal("expected response that skips a group to fail the pipeline")
	}
	if plan.Groups[1].Title != "fix: handle empty input" {
		t.Errorf("covered group title lost: %q", plan.Groups[1].Title)
	}
	if strings.TrimSpace(plan.Groups[0].Title) != "" {
		t.Errorf("untitled group must not get a deterministic fallback: %q", plan.Groups[0].Title)
	}
}

func TestFillMessagesHandlesFencesAndCustomPrompt(t *testing.T) {
	plan := &RawPlan{Groups: []RawGroup{{Items: msgItems("a.go")}}}
	stub := &stubMsgClient{resp: "```json\n{\"groups\":[{\"id\":1,\"title\":\"chore: tidy\"}]}\n```"}
	if err := FillMessages(context.Background(), plan, stub, time.Second, "keep it minimal"); err != nil {
		t.Fatal(err)
	}
	if plan.Groups[0].Title != "chore: tidy" {
		t.Errorf("title = %q", plan.Groups[0].Title)
	}
	if len(stub.seen) != 1 || !strings.Contains(stub.seen[0].Content, "keep it minimal") {
		t.Errorf("custom prompt not passed: %+v", stub.seen)
	}
}

func TestFillMessagesEmptyPlan(t *testing.T) {
	stub := &stubMsgClient{}
	if err := FillMessages(context.Background(), &RawPlan{}, stub, time.Second, ""); err != nil {
		t.Fatal(err)
	}
	if len(stub.seen) != 0 {
		t.Errorf("client called on empty plan")
	}
	_ = fmt.Sprint()
}

// Regression: gc -a -y committed nothing because a failed LLM call left every
// group title empty and `git commit -m ""` fails. The fix now goes the other
// way: a failed message generation must abort the pipeline — no AI title, no
// commit, and never a deterministic fallback title.
func TestFillMessagesErrorAborts(t *testing.T) {
	for name, stub := range map[string]*stubMsgClient{
		"complete error": {err: fmt.Errorf("llm down")},
		"bad json":       {resp: "not json at all {"},
	} {
		t.Run(name, func(t *testing.T) {
			plan := &RawPlan{Groups: []RawGroup{
				{Items: []RawItem{{File: "a.go"}}},
				{Type: "docs", Items: []RawItem{{File: "b.md"}}},
			}}
			if err := FillMessages(context.Background(), plan, stub, time.Second, ""); err == nil {
				t.Fatal("expected error to propagate")
			}
			for i, g := range plan.Groups {
				if strings.TrimSpace(g.Title) != "" {
					t.Errorf("group %d got fallback title %q — no deterministic titles allowed", i, g.Title)
				}
			}
		})
	}
}

// Regression: validators must reject an AI plan with an untitled group
// instead of inventing one.
func TestValidatePlanRejectsEmptyTitle(t *testing.T) {
	cs := &Changeset{Files: []FileChange{
		{Path: "a.go", Status: StatusModified},
	}}
	raw := &RawPlan{Groups: []RawGroup{{Items: []RawItem{{File: "a.go"}}}}}
	plan, err := ValidatePlan(raw, cs, false)
	if err == nil {
		t.Fatal("expected empty title to fail validation")
	}
	if plan != nil {
		t.Errorf("plan returned on invalid input: %+v", plan)
	}
}

// Regression: a 50-group plan in one LLM call deadlines while the response
// body streams, so titles fell back to heuristics (gc -a -y pushed 50
// "chore: update N file(s)" commits). Generation must be chunked.
func TestFillMessagesChunksCalls(t *testing.T) {
	groups := make([]RawGroup, 10)
	for i := range groups {
		groups[i] = RawGroup{Items: []RawItem{{File: fmt.Sprintf("f%d.go", i)}}}
	}
	plan := &RawPlan{Groups: groups}
	stub := &stubMsgChunks{}
	if err := FillMessages(context.Background(), plan, stub, time.Second, ""); err != nil {
		t.Fatal(err)
	}
	if stub.calls != 2 { // 10 groups / messageChunkSize 8 → 2 calls
		t.Errorf("Complete calls = %d, want 2", stub.calls)
	}
	for i, g := range plan.Groups {
		if g.Title != fmt.Sprintf("feat: group %d", i+1) {
			t.Errorf("group %d title = %q", i+1, g.Title)
		}
	}
}

// stubMsgChunks echoes one valid group entry per "GROUP n" line seen in the
// prompt, so chunked calls with global ids can be verified end to end.
type stubMsgChunks struct {
	calls int
}

func (s *stubMsgChunks) Complete(ctx context.Context, system string, messages []llm.Message) (string, error) {
	s.calls++
	var entries []string
	for _, m := range messages {
		for _, line := range strings.Split(m.Content, "\n") {
			line = strings.TrimSpace(line)
			if id, ok := strings.CutPrefix(line, "GROUP "); ok {
				entries = append(entries, fmt.Sprintf(`{"id":%s,"title":"feat: group %s"}`, id, id))
			}
		}
	}
	return `{"groups":[` + strings.Join(entries, ",") + `]}`, nil
}

func (s *stubMsgChunks) Stream(ctx context.Context, systemPrompt string, messages []llm.Message) (<-chan llm.StreamChunk, error) {
	return nil, fmt.Errorf("stream not expected in tests")
}
