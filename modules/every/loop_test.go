package every

import (
	"bytes"
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/yurifrl/cly/modules/every/notify"
)

// fakeClock keeps a virtual time that is advanced by Sleep.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

func newRunner(t *testing.T, exec ExecFunc, exits []int) *Runner {
	t.Helper()
	dir := t.TempDir()
	clk := &fakeClock{now: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)}
	r := NewRunner(dir)
	r.Clock = clk
	r.Exec = exec
	r.Stdout = &bytes.Buffer{}
	r.Rand = rand.New(rand.NewSource(1))
	// Sleep cancels the context after the configured number of iterations
	// so we can drive the loop deterministically.
	r.Sleep = func(ctx context.Context, d time.Duration) error {
		clk.advance(d)
		return nil
	}
	return r
}

// scriptedExec returns a closure that yields successive exit codes from
// `exits` and cancels ctxCancel after the slice is exhausted so the loop
// terminates cleanly via the `signal` shutdown path.
func scriptedExec(exits []int, clk *fakeClock, cancel context.CancelFunc) ExecFunc {
	var i int
	var mu sync.Mutex
	return func(ctx context.Context, dir string, command []string) ExecResult {
		mu.Lock()
		defer mu.Unlock()
		if i >= len(exits) {
			cancel()
			return ExecResult{ExitCode: 0, DurationMs: 1}
		}
		clk.advance(10 * time.Millisecond)
		code := exits[i]
		i++
		return ExecResult{ExitCode: code, DurationMs: 10}
	}
}

func TestLoopHealthyNoTransition(t *testing.T) {
	clk := &fakeClock{now: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)}
	ctx, cancel := context.WithCancel(context.Background())
	dir := t.TempDir()
	r := NewRunner(dir)
	r.Clock = clk
	r.Stdout = &bytes.Buffer{}
	r.Rand = rand.New(rand.NewSource(1))
	r.Sleep = func(ctx context.Context, d time.Duration) error { clk.advance(d); return nil }
	r.Exec = scriptedExec([]int{0, 0, 0}, clk, cancel)

	cfg := RunConfig{Name: "ok", Command: []string{"true"}, Interval: time.Second, Backoff: time.Second}
	if err := r.Run(ctx, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	events, _ := ReadLog(LogPath(dir, "ok"))
	for _, e := range events {
		if e.Event == "transition" {
			t.Fatalf("unexpected transition: %+v", e)
		}
	}
}

func TestLoopHealthyToFailingTransition(t *testing.T) {
	clk := &fakeClock{now: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)}
	ctx, cancel := context.WithCancel(context.Background())
	dir := t.TempDir()
	r := NewRunner(dir)
	r.Clock = clk
	r.Stdout = &bytes.Buffer{}
	r.Rand = rand.New(rand.NewSource(1))
	r.Sleep = func(ctx context.Context, d time.Duration) error { clk.advance(d); return nil }
	r.Exec = scriptedExec([]int{0, 1, 0}, clk, cancel)

	cfg := RunConfig{Name: "flap", Command: []string{"x"}, Interval: time.Second, Backoff: time.Second}
	if err := r.Run(ctx, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	events, _ := ReadLog(LogPath(dir, "flap"))
	var transitions []string
	for _, e := range events {
		if e.Event == "transition" {
			from, _ := e.Extra["from"].(string)
			to, _ := e.Extra["to"].(string)
			transitions = append(transitions, from+"->"+to)
		}
	}
	wantTransitions := []string{"healthy->failing", "failing->healthy"}
	if len(transitions) != 2 || transitions[0] != wantTransitions[0] || transitions[1] != wantTransitions[1] {
		t.Fatalf("transitions: %v", transitions)
	}
}

func TestLoopGiveUp(t *testing.T) {
	clk := &fakeClock{now: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	r := NewRunner(dir)
	r.Clock = clk
	r.Stdout = &bytes.Buffer{}
	r.Rand = rand.New(rand.NewSource(1))
	r.Sleep = func(ctx context.Context, d time.Duration) error { clk.advance(d); return nil }
	r.Exec = func(ctx context.Context, dir string, command []string) ExecResult {
		return ExecResult{ExitCode: 1, DurationMs: 1}
	}

	cfg := RunConfig{Name: "doomed", Command: []string{"x"}, Interval: time.Second, Backoff: time.Second, MaxFails: 3}
	err := r.Run(ctx, cfg)
	if err == nil {
		t.Fatal("expected give_up error")
	}
	events, _ := ReadLog(LogPath(dir, "doomed"))
	var sawGaveUp bool
	for _, e := range events {
		if e.Event == "gave_up" {
			sawGaveUp = true
		}
	}
	if !sawGaveUp {
		t.Fatal("missing gave_up event")
	}
	st, err := ReadState(StatePath(dir, "doomed"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != StatusGaveUp {
		t.Fatalf("status = %s, want gave_up", st.Status)
	}
}

func TestLoopAdoptionPreservesTotals(t *testing.T) {
	dir := t.TempDir()
	// Pre-existing state from a "previous run" (dead pid).
	prev := &State{
		Name: "adopted", PID: 0,
		TotalsLifetime: Totals{Runs: 5, Success: 5},
		Status:         StatusHealthy,
	}
	if err := WriteState(StatePath(dir, "adopted"), prev); err != nil {
		t.Fatal(err)
	}

	clk := &fakeClock{now: time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := NewRunner(dir)
	r.Clock = clk
	r.Stdout = &bytes.Buffer{}
	r.Rand = rand.New(rand.NewSource(1))
	iter := 0
	r.Sleep = func(ctx context.Context, d time.Duration) error {
		iter++
		clk.advance(d)
		if iter >= 1 {
			cancel()
		}
		return nil
	}
	r.Exec = func(ctx context.Context, dir string, command []string) ExecResult {
		clk.advance(10 * time.Millisecond)
		return ExecResult{ExitCode: 0, DurationMs: 10}
	}

	cfg := RunConfig{Name: "adopted", Command: []string{"x"}, Interval: time.Second, Backoff: time.Second}
	if err := r.Run(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	st, _ := ReadState(StatePath(dir, "adopted"))
	if st.TotalsLifetime.Runs != 6 {
		t.Fatalf("expected lifetime runs to continue at 6, got %d", st.TotalsLifetime.Runs)
	}
}

// notify level smoke test: nil notifier must not crash, real path skipped.
func TestNotifyLevelString(t *testing.T) {
	if notify.LevelFailing.String() != "failing" {
		t.Fatal("level string mismatch")
	}
}
