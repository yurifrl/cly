package ompsummary

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yurifrl/cly/pkg/ai"
)

// Job is one summarization request in the queue.
type Job struct {
	Target      Target
	Fingerprint string
	Focused     bool
}

// Summarizer runs AI summaries over a priority queue. The focused surface's
// job always jumps ahead of background ones; a stale background run for the
// same session is cancelled when a focused job arrives.
type Summarizer struct {
	override map[string]interface{}
	Provider string // summarizer provider (for the badge)
	Model    string // summarizer model (for the badge)
	HasKey   bool

	cfg Config

	mu      sync.Mutex
	queue   []Job
	running map[string]context.CancelFunc // session_id → cancel
	wg      sync.WaitGroup
	notify  func() // wakes the UI when a summary lands
}

// NewSummarizer resolves the module's AI override and the summarizer's
// provider/model badge once.
func NewSummarizer(cfg Config) *Summarizer {
	s := &Summarizer{
		cfg:     cfg,
		running: map[string]context.CancelFunc{},
	}
	if ov := ai.LookupModuleOverride("omp.summary"); ov != nil {
		s.override = ov
	}
	if r := ai.LoadConfigWith(s.override); r != nil {
		s.Provider = r.Provider
		s.Model = r.Model
	}
	s.HasKey = ai.HasAPIKeyFor("omp.summary")
	return s
}

// SetNotify wires the UI wake-up callback.
func (s *Summarizer) SetNotify(fn func()) {
	s.notify = fn
}

func (s *Summarizer) wake() {
	if s.notify != nil {
		s.notify()
	}
}

// Enqueue adds a job unless the session is already queued. A focused job
// arriving for a session that is currently running (stale fingerprint)
// cancels that run so the fresh one can fire on the next tick.
func (s *Summarizer) Enqueue(j Job) {
	if !s.HasKey {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if cancel, busy := s.running[j.Target.SessionID]; busy {
		if j.Focused {
			cancel()
		}
		return
	}
	for i := range s.queue {
		if s.queue[i].Target.SessionID == j.Target.SessionID {
			s.queue[i].Fingerprint = j.Fingerprint
			s.queue[i].Target = j.Target
			if j.Focused {
				s.queue[i].Focused = true
			}
			return
		}
	}
	s.queue = append(s.queue, j)
}

// pending pops the next job: focused first, then FIFO.
func (s *Summarizer) pending() (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) == 0 {
		return Job{}, false
	}
	idx := 0
	for i, j := range s.queue {
		if j.Focused && !s.queue[idx].Focused {
			idx = i
			break
		}
	}
	j := s.queue[idx]
	s.queue = append(s.queue[:idx], s.queue[idx+1:]...)
	return j, true
}

// Start launches worker goroutines; blocks until ctx is cancelled.
func (s *Summarizer) Start(ctx context.Context) {
	s.wg.Add(s.cfg.Workers)
	for range s.cfg.Workers {
		go s.worker(ctx)
	}
	s.wg.Wait()
}

func (s *Summarizer) worker(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		job, ok := s.pending()
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}
		s.run(ctx, job)
	}
}

func (s *Summarizer) run(ctx context.Context, job Job) {
	t := job.Target
	runCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.running[t.SessionID] = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, t.SessionID)
		s.mu.Unlock()
		cancel()
	}()

	state := LoadState(t.CWD)
	state.Update(t.SessionID, &Summary{
		Status:      "running",
		Fingerprint: job.Fingerprint,
		UpdatedAt:   nowUnix(),
	})
	_ = SaveState(t.CWD, state)
	s.wake()

	text, turns, err := func() (string, int, error) {
		path := transcriptPath(t.SessionID)
		digest, turns, err := Extract(path, s.cfg.MaxTail, s.cfg.MaxTail)
		if err != nil {
			return "", turns, err
		}
		out, err := ai.Complete(runCtx, s.override, systemPrompt, buildUserPrompt(t, digest))
		if err != nil {
			return "", turns, err
		}
		return normalize(out), turns, nil
	}()

	if runCtx.Err() != nil {
		return // superseded; the next tick re-enqueues with the fresh fingerprint
	}
	res := &Summary{
		Fingerprint: job.Fingerprint,
		UpdatedAt:   nowUnix(),
		Turns:       turns,
		Provider:    s.Provider,
		Model:       s.Model,
	}
	if err != nil {
		res.Status, res.Error = "error", err.Error()
	} else {
		res.Status, res.Text = "ok", text
	}
	state = LoadState(t.CWD)
	state.Update(t.SessionID, res)
	_ = SaveState(t.CWD, state)
	s.wake()
}

// ShouldSummarize decides whether a target needs (re)summarizing.
func (s *Summarizer) ShouldSummarize(t Target, fp string, now float64) bool {
	if !s.HasKey || fp == "" {
		return false
	}
	cur, ok := LoadState(t.CWD).Summaries[t.SessionID]
	if !ok {
		return true
	}
	if cur.Fingerprint == fp && cur.Status == "ok" {
		return false
	}
	if cur.Status == "running" {
		if cur.Fingerprint != fp && t.Focused {
			return true // stale run superseded by a focused change
		}
		return now-cur.UpdatedAt >= s.cfg.Debounce.Seconds()
	}
	if cur.Status == "error" {
		if cur.Fingerprint != fp {
			return true
		}
		return now-cur.UpdatedAt >= s.cfg.Debounce.Seconds()
	}
	return cur.Fingerprint != fp
}

const systemPrompt = `You compress coding-agent session transcripts into a single line for a narrow sidebar.
Rules:
- Reply with ONE line, at most 180 characters, no quotes, no markdown, no trailing period.
- Present tense, name the concrete task or change being worked on.
- Prefer specifics (file names, feature names) over generic phrases like "working on code".
- If the transcript is empty or unreadable, reply exactly: idle`

var wsRe = regexp.MustCompile(`\s+`)

// normalize collapses whitespace and hard-caps the summary.
func normalize(s string) string {
	s = wsRe.ReplaceAllString(strings.TrimSpace(s), " ")
	r := []rune(s)
	const max = 200
	if len(r) > max {
		r = r[:max]
	}
	return string(r)
}

func buildUserPrompt(t Target, digest string) string {
	var b strings.Builder
	b.WriteString("Session: " + t.Title + "\n")
	if t.CWD != "" {
		b.WriteString("Project: " + t.CWD + "\n")
	}
	b.WriteString("Transcript tail (oldest first):\n\n")
	b.WriteString(digest)
	return b.String()
}

// Drain drops queued jobs whose fingerprint changed en route or whose
// session vanished (their target re-enqueues on the next tick if needed).
func (s *Summarizer) Drain(current map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.queue[:0]
	for _, j := range s.queue {
		fp, ok := current[j.Target.SessionID]
		if ok && fp != "" && fp != j.Fingerprint {
			continue
		}
		kept = append(kept, j)
	}
	s.queue = kept
}
