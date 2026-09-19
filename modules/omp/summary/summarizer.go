package ompsummary

import (
	"context"
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

// Idle reports whether every job has run to completion (queue drained, no
// running summary). Used by --once mode to know when to exit.
func (s *Summarizer) Idle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue) == 0 && len(s.running) == 0
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
	marker := state.Get(t.SessionID)
	if marker == nil {
		marker = &Entry{}
	}
	marker.Status = StatusRunning
	marker.Fingerprint = job.Fingerprint
	marker.UpdatedAt = nowUnix()
	state.Update(t.SessionID, marker)
	_ = SaveState(t.CWD, state)

	text, dg, err := func() (string, Digest, error) {
		path := transcriptPath(t.SessionID)
		dg, tail, err := DigestTail(path, s.cfg.MaxTail, s.cfg.MaxTail)
		if err != nil {
			return "", dg, err
		}
		out, err := ai.Complete(runCtx, s.override, systemPrompt, buildUserPrompt(t, tail))
		if err != nil {
			return "", dg, err
		}
		return out, dg, nil
	}()

	if runCtx.Err() != nil {
		return // superseded; the next tick re-enqueues with the fresh fingerprint
	}
	// Residual fields always carry over so a failed run never blanks the card.
	res := *marker
	res.Fingerprint = job.Fingerprint
	res.UpdatedAt = nowUnix()
	res.Turns = dg.Turns
	res.Provider, res.Model = s.Provider, s.Model
	res.SurfaceID, res.Title, res.Workspace = t.SurfaceID, t.Title, t.Workspace
	if dg.LastUser != "" {
		res.UserAsk, res.UserAskAt = dg.LastUser, dg.LastUserAt
	}
	if dg.LastTool != "" {
		res.LastCmd, res.Exit = dg.LastTool, dg.Exit
	}
	if err != nil {
		res.Status, res.Error = StatusError, err.Error()
	} else {
		goal, status := parseReply(text)
		if goal != "" {
			res.Goal = goal
		}
		res.AIStatus = status
		res.Error = ""
		res.Status = StatusOK
	}
	state = LoadState(t.CWD)
	state.Update(t.SessionID, &res)
	_ = SaveState(t.CWD, state)
}

// ShouldSummarize decides whether a target needs (re)summarizing.
func (s *Summarizer) ShouldSummarize(t Target, fp string, now float64) bool {
	if !s.HasKey || fp == "" {
		return false
	}
	cur, ok := LoadState(t.CWD).Sessions[t.SessionID]
	if !ok {
		return true
	}
	if cur.Fingerprint == fp && cur.Status == StatusOK {
		return false
	}
	if cur.Status == StatusRunning {
		if cur.Fingerprint != fp && t.Focused {
			return true // stale run superseded by a focused change
		}
		return now-cur.UpdatedAt >= s.cfg.Debounce.Seconds()
	}
	if cur.Status == StatusError {
		if cur.Fingerprint != fp {
			return true
		}
		return now-cur.UpdatedAt >= s.cfg.Debounce.Seconds()
	}
	return cur.Fingerprint != fp
}

const systemPrompt = `You summarize a live coding-agent session for a tiny sidebar card.
Reply with EXACTLY two lines and nothing else:
Line 1 - the session's overall goal: what it is trying to accomplish, imperative.
Line 2 - what the assistant is doing or asking for right now, present tense.
Rules:
- Max 100 characters per line, plain terse text.
- No markdown, no quotes, no labels or prefixes like "Goal:".
- Prefer specifics (file names, feature names) over generic phrases.
- If the transcript is empty or unreadable, reply exactly:
idle
waiting for activity`

// parseReply splits the model's two-line reply into goal and status.
func parseReply(out string) (goal, status string) {
	const cap = 120
	for _, ln := range strings.Split(out, "\n") {
		ln = capRunes(ln, cap)
		if ln == "" {
			continue
		}
		if goal == "" {
			goal = ln
		} else if status == "" {
			status = ln
			break
		}
	}
	return goal, status
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
