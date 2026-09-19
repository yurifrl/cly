package ompsummary

import (
	"context"
	"sync"
	"time"
)

// Engine drives the loop: discover targets → decide what needs a summary →
// enqueue jobs → publish snapshots to the TUI. It owns no TUI state; the TUI
// pulls immutable snapshots from it.
type Engine struct {
	cfg Config
	sum *Summarizer

	send func() // wakes the TUI; wired to p.Send after the program starts

	mu      sync.Mutex
	targets []Target
}

// NewEngine builds the engine; call (*Summarizer) ownership stays with the
// caller so the TUI can read the summarizer badge.
func NewEngine(cfg Config, sum *Summarizer) *Engine {
	return &Engine{cfg: cfg, sum: sum}
}

// SetSend wires the TUI wake-up channel.
func (e *Engine) SetSend(fn func()) { e.send = fn }

func (e *Engine) wake() {
	if e.send != nil {
		e.send()
	}
}

// Snapshot returns the current targets and their summaries (session id →
// summary), reading state files from disk. Cheap at sidebar cadence.
func (e *Engine) Snapshot() ([]Target, map[string]*Summary) {
	e.mu.Lock()
	targets := append([]Target(nil), e.targets...)
	e.mu.Unlock()

	bySession := map[string]*Summary{}
	stateByCWD := map[string]*State{}
	for _, t := range targets {
		st, ok := stateByCWD[t.CWD]
		if !ok {
			st = LoadState(t.CWD)
			stateByCWD[t.CWD] = st
		}
		if s, ok := st.Summaries[t.SessionID]; ok {
			bySession[t.SessionID] = s
		}
	}
	return targets, bySession
}

// Run blocks: tick loop until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	e.tick(ctx) // immediate first pass
	t := time.NewTicker(e.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	now := nowUnix()
	targets := Targets(ctx, now, e.cfg.MaxAge.Seconds())

	fps := map[string]string{}
	for _, t := range targets {
		fp := Fingerprint(transcriptPath(t.SessionID))
		fps[t.SessionID] = fp
		if e.sum.ShouldSummarize(t, fp, now) {
			e.sum.Enqueue(Job{Target: t, Fingerprint: fp, Focused: t.Focused})
		}
	}
	e.sum.Drain(fps)

	e.mu.Lock()
	e.targets = targets
	e.mu.Unlock()
	e.wake()
}
