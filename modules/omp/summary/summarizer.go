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

// Enqueue adds a job unless the session is already queued or an AI
// round-trip for it is in flight. Never cancels the in-flight run: a live
// session's transcript keeps growing, so a "fresher" focused job arrives
// every tick — cancelling on each one starves the session forever. The
// running job finishes and saves; the next tick re-enqueues if the
// fingerprint moved again.
func (s *Summarizer) Enqueue(j Job) {
	if !s.HasKey {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, busy := s.running[j.Target.SessionID]; busy {
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
	// Reserve the running slot before the pop becomes visible. Idle()
	// counts queue and running; without the reservation, --once observes
	// the pop-to-register window (queue empty, running empty) and exits
	// before the worker even starts the job.
	s.running[j.Target.SessionID] = func() {}
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

	// Fold scan runs before the AI call so the latest fold's raw summary can
	// ride in the prompt; the mechanical note still lands on the card even
	// when the summary run errors.
	path := transcriptPath(t.SessionID)
	fold := ScanCompactions(path)

	text, dg, err := func() (string, Digest, error) {
		dg, tail, err := DigestTail(path, s.cfg.MaxTail, s.cfg.MaxTail)
		if err != nil {
			return "", dg, err
		}
		out, err := ai.Complete(runCtx, s.override, systemPrompt, buildUserPrompt(t, tail, fold))
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
	// Compactions always surface on the card, even when the summary run
	// errors — that's why the mechanical note exists.
	if fold.Count > 0 {
		res.Compactions, res.CompactedAt, res.FoldNote = fold.Count, fold.At, fold.Note
	}
	if dg.LastUser != "" {
		res.UserAsk, res.UserAskAt = dg.LastUser, dg.LastUserAt
	}
	if dg.LastTool != "" {
		res.LastCmd, res.Exit = dg.LastTool, dg.Exit
	}
	if err != nil {
		res.Status, res.Error = StatusError, err.Error()
	} else {
		goal, response, action, foldNote := parseReply(text)
		if goal != "" {
			res.Goal = goal
		}
		if response != "" {
			res.Response = response
		} else {
			res.Response = capRunes(dg.LastAssist, 140)
		}
		if action != "" {
			res.Action = action
		} else {
			res.Action = dg.Action
		}
		// The LLM's distillation of the fold summary beats the mechanical
		// fallback; a missing or "none" line 4 keeps the mechanical note.
		if foldNote != "" {
			res.FoldNote = foldNote
		}
		res.Timeline = dg.Timeline
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
Reply with EXACTLY four lines and nothing else:
Line 1 - the session's overall goal: what it is trying to accomplish, imperative.
Line 2 - what the assistant last did or produced, present tense.
Line 3 - what the user must answer or do next, or exactly none.
Line 4 - one line distilling the "Compacted history" section of the prompt: what the pre-fold context contained (work done, key files, where things left off); exactly none when the prompt has no such section.
Rules:
- Max 100 characters per line, plain terse text.
- No markdown, no quotes, no labels or prefixes like "Goal:".
- Prefer specifics (file names, feature names) over generic phrases.
- If the transcript is empty or unreadable, reply exactly:
idle
waiting for activity
none`

// parseReply splits the model's reply into goal, response, action and the
// optional fold distillation (line 4, present only when a compaction exists).
func parseReply(out string) (goal, response, action, fold string) {
	const cap = 120
	slot := 0
	var fields [4]string
	for _, ln := range strings.Split(out, "\n") {
		ln = capRunes(ln, cap)
		if ln == "" {
			continue
		}
		if slot > 3 {
			break
		}
		if strings.EqualFold(ln, "none") {
			ln = ""
		}
		// Line 4 echoing the prompt's own instruction is not a distillation;
		// drop it so the mechanical note survives.
		if slot == 3 {
			low := strings.ToLower(ln)
			if strings.Contains(low, "line 4") || strings.Contains(low, "exactly none") {
				ln = ""
			}
		}
		fields[slot] = ln
		slot++
	}
	return fields[0], fields[1], fields[2], fields[3]
}

func buildUserPrompt(t Target, digest string, fold Fold) string {
	var b strings.Builder
	b.WriteString("Session: " + t.Title + "\n")
	if t.CWD != "" {
		b.WriteString("Project: " + t.CWD + "\n")
	}
	b.WriteString("Transcript tail (oldest first):\n\n")
	b.WriteString(digest)
	if fold.Summary != "" {
		b.WriteString("\n\nCompacted history (the session's older context was folded; latest fold's summary):\n")
		b.WriteString(fold.Summary)
	}
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
