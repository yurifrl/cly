package every

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yurifrl/cly/modules/every/notify"
	"github.com/yurifrl/cly/pkg/mut"
)

// Clock abstracts time so tests can drive the loop deterministically.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// SleepFunc returns nil after d (or after ctx is cancelled). Tests can swap
// this with a deterministic implementation.
type SleepFunc func(ctx context.Context, d time.Duration) error

func realSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ExecResult is the captured outcome of one command execution.
type ExecResult struct {
	ExitCode   int
	DurationMs int64
	Err        error
}

// ExecFunc executes the command and returns its result. Implementations must
// honour mut.DryRun and ctx cancellation.
type ExecFunc func(ctx context.Context, dir string, command []string) ExecResult

// DefaultExec runs the child via os/exec, forwarding stdio. SIGTERM is sent
// to the child when ctx is cancelled, with a 5 second WaitDelay before the
// kernel SIGKILL kicks in.
func DefaultExec(ctx context.Context, dir string, command []string) ExecResult {
	if len(command) == 0 {
		return ExecResult{ExitCode: -1, Err: fmt.Errorf("empty command")}
	}
	if mut.DryRun() {
		mut.Log("exec", strings.Join(command, " "))
		return ExecResult{}
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 5 * time.Second
	err := cmd.Run()
	dur := time.Since(start).Milliseconds()
	if err == nil {
		return ExecResult{ExitCode: 0, DurationMs: dur}
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ExecResult{ExitCode: ee.ExitCode(), DurationMs: dur}
	}
	return ExecResult{ExitCode: -1, DurationMs: dur, Err: err}
}

// Notifier sends a desktop notification. When nil the loop skips notifies.
type Notifier func(ctx context.Context, level notify.Level, taskName, body string) error

// DefaultNotifier wires up modules/every/notify.
func DefaultNotifier(ctx context.Context, level notify.Level, taskName, body string) error {
	title := "cly every: " + taskName
	switch level {
	case notify.LevelFailing:
		title = "❌ " + taskName + " failing"
	case notify.LevelRecovered:
		title = "✅ " + taskName + " recovered"
	case notify.LevelGaveUp:
		title = "💀 " + taskName + " gave up"
	}
	return notify.Send(ctx, notify.Notification{
		TaskName: taskName,
		Level:    level,
		Title:    title,
		Body:     body,
	})
}

// RunConfig is the input for Runner.Run.
type RunConfig struct {
	Name         string
	Command      []string
	Interval     time.Duration
	Backoff      time.Duration
	Jitter       time.Duration
	InitialDelay time.Duration
	MaxFails     int
	Notify       bool
	WorkDir      string // child cwd; empty = inherit
}

// Runner is the injectable run-loop driver.
type Runner struct {
	Dir      string
	Clock    Clock
	Sleep    SleepFunc
	Exec     ExecFunc
	Notifier Notifier
	Stdout   io.Writer
	Rand     *rand.Rand

	// internal: signal-driven shutdown channel populated by Run().
	signaled chan os.Signal
}

// NewRunner builds a Runner with sensible production defaults.
func NewRunner(dir string) *Runner {
	return &Runner{
		Dir:      dir,
		Clock:    realClock{},
		Sleep:    realSleep,
		Exec:     DefaultExec,
		Notifier: DefaultNotifier,
		Stdout:   os.Stdout,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Run executes the supervised loop. It returns nil on graceful shutdown and
// an error on startup/acquire failures (or when the task gives up: see exit
// semantics in cmd.go).
func (r *Runner) Run(ctx context.Context, cfg RunConfig) error {
	if err := EnsureDir(r.Dir); err != nil {
		return err
	}

	// 1. sweep orphans on startup
	if removed, err := SweepOrphans(r.Dir, r.Clock.Now()); err == nil && len(removed) > 0 {
		_ = AppendLog(LogPath(r.Dir, cfg.Name), Event{
			TS:    r.Clock.Now(),
			Event: "swept",
			Extra: map[string]any{"pruned": removed},
		})
	}

	// 2. acquire state file
	statePath := StatePath(r.Dir, cfg.Name)
	prev, err := ReadState(statePath)
	switch {
	case os.IsNotExist(err):
		prev = nil
	case err != nil:
		return fmt.Errorf("read state: %w", err)
	default:
		if PIDAlive(prev.PID) && prev.PID != os.Getpid() {
			return fmt.Errorf("another instance running, pid %d", prev.PID)
		}
	}

	now := r.Clock.Now()
	state := mergeState(prev, cfg, now)
	if err := WriteState(statePath, state); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	// 3. signal handlers
	sigCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	r.signaled = make(chan os.Signal, 1)
	signal.Notify(r.signaled, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer signal.Stop(r.signaled)
	go func() {
		select {
		case <-sigCtx.Done():
		case <-r.signaled:
			cancel()
		}
	}()

	// 4. initial delay
	if cfg.InitialDelay > 0 {
		if err := r.Sleep(sigCtx, cfg.InitialDelay); err != nil {
			return r.shutdown(state, statePath, "signal")
		}
	}

	logPath := LogPath(r.Dir, cfg.Name)
	for {
		if sigCtx.Err() != nil {
			return r.shutdown(state, statePath, "signal")
		}

		// run number
		runNo := state.TotalsLifetime.Runs + 1
		retry := 0
		if state.Status == StatusFailing {
			retry = state.ConsecutiveFails
		}

		// log start
		startEv := Event{TS: r.Clock.Now(), Event: "start", Extra: map[string]any{"run": runNo}}
		if retry > 0 {
			startEv.Extra["retry"] = retry
		}
		_ = AppendLog(logPath, startEv)
		fmt.Fprintln(r.Stdout, FormatRunStart(r.Clock.Now(), runNo, strings.Join(cfg.Command, " "), retry, cfg.MaxFails))

		// stamp pre-run state
		state.LastRunAt = r.Clock.Now()
		_ = WriteState(statePath, state)

		// exec
		res := r.Exec(sigCtx, cfg.WorkDir, cfg.Command)

		// classify
		prevStatus := state.Status
		newStatus := StatusHealthy
		if res.ExitCode != 0 {
			newStatus = StatusFailing
		}
		if newStatus == StatusFailing {
			state.ConsecutiveFails++
			if cfg.MaxFails > 0 && state.ConsecutiveFails >= cfg.MaxFails {
				newStatus = StatusGaveUp
			}
		} else {
			state.ConsecutiveFails = 0
		}

		// transitions
		if newStatus != prevStatus {
			extra := map[string]any{"from": prevStatus, "to": newStatus}
			if newStatus == StatusHealthy && prevStatus == StatusFailing {
				extra["retries"] = state.TotalsLifetime.Fail // best-effort
			}
			_ = AppendLog(logPath, Event{TS: r.Clock.Now(), Event: "transition", Extra: extra})
			if cfg.Notify && r.Notifier != nil {
				level := notify.LevelFailing
				body := strings.Join(cfg.Command, " ")
				if newStatus == StatusHealthy {
					level = notify.LevelRecovered
				} else if newStatus == StatusGaveUp {
					level = notify.LevelGaveUp
				}
				_ = r.Notifier(ctx, level, cfg.Name, body)
			}
		}

		// log end
		_ = AppendLog(logPath, Event{
			TS:    r.Clock.Now(),
			Event: "end",
			Extra: map[string]any{"run": runNo, "exit": res.ExitCode, "duration_ms": res.DurationMs},
		})

		// totals
		state.TotalsLifetime.Runs++
		state.Totals24h.Runs++
		if res.ExitCode == 0 {
			state.TotalsLifetime.Success++
			state.Totals24h.Success++
		} else {
			state.TotalsLifetime.Fail++
			state.Totals24h.Fail++
		}
		state.LastExit = res.ExitCode
		state.LastDurationMs = res.DurationMs
		state.Status = newStatus

		// next sleep duration
		var sleep time.Duration
		switch newStatus {
		case StatusHealthy:
			sleep = withJitter(cfg.Interval, cfg.Jitter, r.Rand)
		case StatusFailing:
			sleep = withJitter(cfg.Backoff, cfg.Jitter, r.Rand)
		case StatusGaveUp:
			sleep = 0
		}
		state.NextRunAt = r.Clock.Now().Add(sleep)
		_ = WriteState(statePath, state)

		// stdout summary line
		nextLabel := "next in " + FormatDuration(sleep)
		if newStatus == StatusFailing {
			nextLabel = "retry in " + FormatDuration(sleep)
		}
		if newStatus == StatusGaveUp {
			nextLabel = "giving up"
		}
		fmt.Fprintln(r.Stdout, FormatRunEnd(r.Clock.Now(), res.ExitCode, time.Duration(res.DurationMs)*time.Millisecond, nextLabel))

		if newStatus == StatusGaveUp {
			_ = AppendLog(logPath, Event{
				TS:    r.Clock.Now(),
				Event: "gave_up",
				Extra: map[string]any{"fails": state.ConsecutiveFails},
			})
			return fmt.Errorf("gave up after %d consecutive failures", state.ConsecutiveFails)
		}

		// trim every ~10 events
		if state.TotalsLifetime.Runs%10 == 0 {
			_ = MaybeTrimLog(logPath, r.Clock.Now(), true)
		}

		// sleep until next run
		if err := r.Sleep(sigCtx, sleep); err != nil {
			return r.shutdown(state, statePath, "signal")
		}
	}
}

func (r *Runner) shutdown(state *State, path, reason string) error {
	_ = AppendLog(LogPath(r.Dir, state.Name), Event{
		TS:    r.Clock.Now(),
		Event: "shutdown",
		Extra: map[string]any{"reason": reason},
	})
	state.PID = 0
	_ = WriteState(path, state)
	return nil
}

func mergeState(prev *State, cfg RunConfig, now time.Time) *State {
	s := &State{
		Name:        cfg.Name,
		Command:     strings.Join(cfg.Command, " "),
		IntervalSec: int(cfg.Interval.Seconds()),
		BackoffSec:  int(cfg.Backoff.Seconds()),
		MaxFails:    cfg.MaxFails,
		JitterSec:   int(cfg.Jitter.Seconds()),
		PID:         os.Getpid(),
		StartedAt:   now,
		Status:      StatusHealthy,
	}
	if prev != nil {
		// adopt history
		s.Totals24h = prev.Totals24h
		s.TotalsLifetime = prev.TotalsLifetime
		s.LastRunAt = prev.LastRunAt
		s.LastExit = prev.LastExit
		s.LastDurationMs = prev.LastDurationMs
		s.ConsecutiveFails = prev.ConsecutiveFails
		if prev.Status != "" {
			s.Status = prev.Status
		}
	}
	return s
}

func withJitter(base, jitter time.Duration, r *rand.Rand) time.Duration {
	if jitter <= 0 || base <= 0 {
		return base
	}
	delta := r.Int63n(int64(jitter)*2+1) - int64(jitter)
	out := base + time.Duration(delta)
	if out < 0 {
		out = 0
	}
	return out
}
