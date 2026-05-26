package every

import (
	"fmt"
	"strings"
	"time"

	"github.com/yurifrl/cly/modules/every/internal"
)

// FormatTS renders an RFC3339 UTC timestamp.
func FormatTS(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.UTC().Format(time.RFC3339)
}

// FormatDuration renders a duration in compact human form (e.g. 5.2s).
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// FormatAgo renders "Nm ago" / "Nh ago" / "Nd ago".
func FormatAgo(t time.Time, now time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// FormatRunStart renders the live "▶ run #N name" line.
func FormatRunStart(now time.Time, run int, command string, retry, maxFails int) string {
	prefix := internal.BlueStyle.Render("▶")
	tail := ""
	if retry > 0 {
		max := "∞"
		if maxFails > 0 {
			max = fmt.Sprintf("%d", maxFails)
		}
		tail = fmt.Sprintf(" (retry %d/%s)", retry, max)
	}
	return fmt.Sprintf("%s %s run #%d %s%s", FormatTS(now), prefix, run, command, tail)
}

// FormatRunEnd renders the live "✓ done … / ✗ failed …" line.
func FormatRunEnd(now time.Time, exit int, dur time.Duration, nextLabel string) string {
	if exit == 0 {
		return fmt.Sprintf("%s %s done in %s (exit 0) → %s", FormatTS(now), internal.GreenStyle.Render("✓"), FormatDuration(dur), nextLabel)
	}
	return fmt.Sprintf("%s %s failed in %s (exit %d) → %s", FormatTS(now), internal.RedStyle.Render("✗"), FormatDuration(dur), exit, nextLabel)
}

// FormatLogEvent renders one parsed Event for the `every logs` command.
func FormatLogEvent(e Event) string {
	ts := FormatTS(e.TS)
	switch e.Event {
	case "start":
		run, _ := e.Extra["run"].(float64)
		retry, _ := e.Extra["retry"].(float64)
		extra := ""
		if retry > 0 {
			extra = fmt.Sprintf(" retry=%d", int(retry))
		}
		return fmt.Sprintf("%s %s start  run=%d%s", ts, internal.BlueStyle.Render("▶"), int(run), extra)
	case "end":
		run, _ := e.Extra["run"].(float64)
		exit, _ := e.Extra["exit"].(float64)
		dur, _ := e.Extra["duration_ms"].(float64)
		marker := internal.GreenStyle.Render("✓")
		if int(exit) != 0 {
			marker = internal.RedStyle.Render("✗")
		}
		return fmt.Sprintf("%s %s end    run=%d exit=%d duration=%s", ts, marker, int(run), int(exit), FormatDuration(time.Duration(dur)*time.Millisecond))
	case "transition":
		from, _ := e.Extra["from"].(string)
		to, _ := e.Extra["to"].(string)
		return fmt.Sprintf("%s %s transition %s → %s", ts, internal.YellowStyle.Render("↻"), from, to)
	case "gave_up":
		fails, _ := e.Extra["fails"].(float64)
		return fmt.Sprintf("%s %s gave_up fails=%d", ts, internal.RedStyle.Render("✗"), int(fails))
	case "shutdown":
		reason, _ := e.Extra["reason"].(string)
		return fmt.Sprintf("%s ⏹ shutdown reason=%s", ts, reason)
	case "swept":
		var pruned []string
		if v, ok := e.Extra["pruned"].([]any); ok {
			for _, x := range v {
				if s, ok := x.(string); ok {
					pruned = append(pruned, s)
				}
			}
		}
		return fmt.Sprintf("%s 🧹 swept %s", ts, strings.Join(pruned, ", "))
	default:
		return fmt.Sprintf("%s %s %v", ts, e.Event, e.Extra)
	}
}

// FormatStatusTable renders the multi-task table used by `cly every status`.
func FormatStatusTable(rows []StatusRow, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-20s %-12s %-12s %-10s %-14s %-10s\n",
		"TASK", "STATE", "LAST RUN", "24H ✓/✗", "NEXT", "LIFETIME")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-20s %-12s %-12s %-10s %-14s %-10s\n",
			r.Name,
			r.LifecycleLabel(),
			FormatAgo(r.LastRunAt, now),
			fmt.Sprintf("%d/%d", r.Totals24h.Success, r.Totals24h.Fail),
			r.NextLabel(now),
			fmt.Sprintf("%d/%d", r.TotalsLifetime.Success, r.TotalsLifetime.Runs),
		)
	}
	return b.String()
}

// StatusRow is the minimal projection of a State for table rendering.
type StatusRow struct {
	Name           string
	Lifecycle      string
	Status         string
	LastRunAt      time.Time
	NextRunAt      time.Time
	Totals24h      Totals
	TotalsLifetime Totals
	BackoffSec     int
	IntervalSec    int
}

// LifecycleLabel returns the colored "● <state>" string.
func (r StatusRow) LifecycleLabel() string {
	dot := "●"
	state := r.Lifecycle
	if r.Status == StatusFailing {
		state = "failing"
		return internal.YellowStyle.Render(dot+" "+state)
	}
	if r.Status == StatusGaveUp {
		state = "gave_up"
		return internal.RedStyle.Render(dot+" "+state)
	}
	switch r.Lifecycle {
	case LifecycleActive:
		return internal.GreenStyle.Render(dot+" "+state)
	case LifecycleStopped:
		return internal.SubtleStyle.Render(dot+" "+state)
	case LifecycleOrphan:
		return internal.RedStyle.Render(dot+" "+state)
	}
	return dot + " " + state
}

// NextLabel renders the NEXT column for a status row.
func (r StatusRow) NextLabel(now time.Time) string {
	if r.NextRunAt.IsZero() {
		return "—"
	}
	d := r.NextRunAt.Sub(now)
	if d < 0 {
		d = 0
	}
	if r.Status == StatusFailing {
		return fmt.Sprintf("retry %s", FormatDuration(d))
	}
	return fmt.Sprintf("in %s", FormatDuration(d))
}

// RowFromState projects a State + lifecycle into a StatusRow.
func RowFromState(s *State, life string) StatusRow {
	return StatusRow{
		Name:           s.Name,
		Lifecycle:      life,
		Status:         s.Status,
		LastRunAt:      s.LastRunAt,
		NextRunAt:      s.NextRunAt,
		Totals24h:      s.Totals24h,
		TotalsLifetime: s.TotalsLifetime,
		BackoffSec:     s.BackoffSec,
		IntervalSec:    s.IntervalSec,
	}
}
