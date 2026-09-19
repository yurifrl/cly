package ompsummary

import (
	"context"
	"time"
)

// Engine drives the daemon loop: resolve the active surface → enqueue a
// summary when its transcript changed → push the cached card into the
// workspace description slot the sidebar reads. Active-only: everything the
// daemon touches belongs to the surface the user currently has frontmost,
// falling back to the freshest live session in the same workspace when the
// frontmost surface has no session of its own (sidebar, shell tabs — see
// ActiveTarget).
type Engine struct {
	cfg Config
	sum *Summarizer
}

func NewEngine(cfg Config, sum *Summarizer) *Engine {
	return &Engine{cfg: cfg, sum: sum}
}

// Run blocks: tick loop until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	e.Tick(ctx) // immediate first pass: load shows the cached card
	t := time.NewTicker(e.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.Tick(ctx)
		}
	}
}

// Tick is one pass: resolve the active surface, enqueue staleness, push the
// cached card. Exported for --once mode.
func (e *Engine) Tick(ctx context.Context) {
	now := nowUnix()
	t, err := ActiveTarget(ctx, now, e.cfg.MaxAge.Seconds())
	if err != nil || t == nil {
		return // nothing to show (no live omp surface, outside cmux)
	}
	if fp := Fingerprint(transcriptPath(t.SessionID)); fp != "" && e.sum.ShouldSummarize(*t, fp, now) {
		e.sum.Enqueue(Job{Target: *t, Fingerprint: fp, Focused: t.Focused})
	}
	e.push(ctx, *t)
}

// push renders the active session's cached entry into the workspace
// description. Missing entry → no push; the card appears once the first AI
// round-trip lands. Pushes unconditionally every tick: descriptions can go
// stale in the sidebar binding, and one small exec per tick keeps the card
// self-healing.
func (e *Engine) push(ctx context.Context, t Target) {
	entry := LoadState(t.CWD).Get(t.SessionID)
	if entry == nil {
		return
	}
	_ = Push(ctx, t.CWD, t.WorkspaceID, entry)
}
