package ompsummary

import (
	"strings"
	"testing"
)

func digestTailTranscript(t *testing.T, lines ...string) (string, Digest) {
	t.Helper()
	p := writeTranscript(t, lines...)
	d, _, err := DigestTail(p, 64<<10, 8<<10)
	if err != nil {
		t.Fatal(err)
	}
	return p, d
}

func TestDigestTimelineOrderCapsMax(t *testing.T) {
	long := strings.Repeat("y", 120)
	toolLine := `{"type":"custom","customType":"tool_execution_start","data":{"toolName":"edit","intent":"x","toolCallId":"c1"}}`
	_, d := digestTailTranscript(t,
		msgLine("user", "first ask"),
		msgLine("assistant", "first reply"),
		toolLine,
		msgLine("toolResult", "build exited with code 0"),
		msgLine("user", "second ask"),
		msgLine("assistant-toolcall", ""), // non-text content, no text field
		msgLine("user", long),
		msgLine("assistant", "final answer"),
	)
	if d.Timeline[0].Who != "user" || d.Timeline[0].Text != "first ask" {
		t.Fatalf("timeline[0] = %+v, want oldest first", d.Timeline[0])
	}
	if d.Timeline[1].Text != "first reply" {
		t.Fatalf("timeline[1] = %+v", d.Timeline[1])
	}
	if got := d.Timeline[2]; got.Who != "user" || got.Text != "second ask" {
		t.Fatalf("timeline[2] = %+v, want second ask", got)
	}
	if got := d.Timeline[3]; got.Who != "user" || got.Text != long {
		t.Fatalf("timeline[3] = %+v, want user text capped at lineCap (%d) runes", got, lineCap)
	}
	if n := len(d.Timeline); n != 5 {
		t.Fatalf("len(Timeline) = %d, want 5 (tool lines skipped)", n)
	}
	last := d.Timeline[len(d.Timeline)-1]
	if last.Who != "ai" || last.Text != "final answer" {
		t.Fatalf("last timeline entry = %+v", last)
	}
}

func TestDigestStripsMarkdown(t *testing.T) {
	_, d := digestTailTranscript(t,
		msgLine("user", "fix **the name** cutting"),
		msgLine("assistant", "fixing `stripMD` now — **bold** *italic* markers"),
	)
	if got := d.Timeline[len(d.Timeline)-1]; got.Text != "fixing stripMD now — bold italic markers" {
		t.Fatalf("assistant timeline = %q, want cleaned prose", got.Text)
	}
	if strings.ContainsAny(d.LastUser, "*`") || strings.ContainsAny(d.LastAssist, "*`") {
		t.Fatalf("digest fields kept markers: %q / %q", d.LastUser, d.LastAssist)
	}
}

func TestDigestTimelineMaxFive(t *testing.T) {
	var lines []string
	for range 6 {
		lines = append(lines, msgLine("user", "ask N"))
		lines = append(lines, msgLine("assistant", "reply N"))
	}
	_, d := digestTailTranscript(t, lines...)
	if n := len(d.Timeline); n != 5 {
		t.Fatalf("len(Timeline) = %d, want 5", n)
	}
	if d.Timeline[0].Text != "reply N" {
		t.Fatalf("timeline[0] = %+v, want the 5th-oldest kept message", d.Timeline[0])
	}
}

func TestDigestActionFromQuestion(t *testing.T) {
	_, d := digestTailTranscript(t,
		msgLine("user", "fix the login bug"),
		msgLine("assistant", "Done. Should I also update the tests?"),
	)
	if want := "Should I also update the tests?"; d.Action != want {
		t.Fatalf("Action = %q, want %q", d.Action, want)
	}
}

func TestDigestActionPicksLastQuestion(t *testing.T) {
	_, d := digestTailTranscript(t,
		msgLine("user", "fix it"),
		msgLine("assistant", "Retry now? That failed. Or should I roll back instead?"),
	)
	if want := "Or should I roll back instead?"; d.Action != want {
		t.Fatalf("Action = %q, want %q", d.Action, want)
	}
}

func TestDigestActionEmptyWithoutQuestion(t *testing.T) {
	_, d := digestTailTranscript(t,
		msgLine("user", "fix the login bug"),
		msgLine("assistant", "Fixed the bug and pushed."),
	)
	if d.Action != "" {
		t.Fatalf("Action = %q, want empty", d.Action)
	}
}

func TestDigestExistingFieldsUnchanged(t *testing.T) {
	_, d := digestTailTranscript(t,
		msgLine("user", "fix the login bug"),
		`{"type":"custom","customType":"tool_execution_start","data":{"toolName":"edit","intent":"fix login","toolCallId":"c1"}}`,
		msgLine("assistant", "fixed it"),
		msgLine("toolResult", "edit exited with code 3"),
		msgLine("user", "now run the build"),
		msgLine("assistant", "build passed"),
	)
	if d.FirstUser != "fix the login bug" {
		t.Fatalf("FirstUser = %q", d.FirstUser)
	}
	if d.LastUser != "now run the build" {
		t.Fatalf("LastUser = %q", d.LastUser)
	}
	if d.LastUserAt == 0 {
		t.Fatal("LastUserAt = 0, want timestamp")
	}
	if d.LastAssist != "build passed" {
		t.Fatalf("LastAssist = %q", d.LastAssist)
	}
	if d.LastTool != "edit: fix login" {
		t.Fatalf("LastTool = %q", d.LastTool)
	}
	if d.Exit != 3 {
		t.Fatalf("Exit = %d, want 3", d.Exit)
	}
	if d.Turns != 4 {
		t.Fatalf("Turns = %d, want 4", d.Turns)
	}
}
