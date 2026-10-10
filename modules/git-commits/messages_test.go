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

func TestFillMessagesFallbackTitle(t *testing.T) {
	plan := &RawPlan{Groups: []RawGroup{
		{Type: "docs", Scope: "guide", Items: []RawItem{{File: "docs/g.md"}}},
		{Items: []RawItem{{File: "x.go"}}},
	}}
	stub := &stubMsgClient{resp: `{"groups":[{"id":2,"title":"fix: handle empty input"}]}`}
	if err := FillMessages(context.Background(), plan, stub, time.Second, ""); err != nil {
		t.Fatal(err)
	}
	if plan.Groups[0].Title == "" || !strings.HasPrefix(plan.Groups[0].Title, "docs") {
		t.Errorf("group 0 fallback = %q", plan.Groups[0].Title)
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
