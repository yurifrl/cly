package ompsummary

import (
	"context"
	"strconv"
	"strings"

	"github.com/yurifrl/cly/pkg/cmux"
)

// Tagged-line markers the omp-summary sidebar card parses. One line per
// section; the sidebar splits on the markers, so order here is display order.
const (
	tagCmd    = "▸ "
	tagAsk    = "? "
	tagGoal   = "◆ "
	tagAI     = "● "
	tagModel  = "✦ "
	tagStamp  = "⏱ "
	cardCap   = 400 // total card budget in runes
	lineCap   = 140 // per-line budget in runes
)

// Card renders an entry into the tagged-segment sidebar format. Empty
// sections are omitted; segments join with " | " because cmux squashes
// newlines out of workspace descriptions before the sidebar binding sees
// them, and the sidebar splits on the pipe instead. The whole card is
// capped at cardCap runes so a long command or summary can never blow out
// the narrow sidebar.
func Card(e *Entry) string {
	var b strings.Builder
	put := func(tag, text string) {
		if text == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(tag)
		b.WriteString(capRunes(text, lineCap))
	}
	put(tagCmd, lastCommandLine(e))
	put(tagAsk, e.UserAsk)
	put(tagGoal, e.Goal)
	put(tagAI, aiLine(e))
	put(tagModel, modelLine(e))
	put(tagStamp, stampLine(e))
	return capRunes(b.String(), cardCap)
}

// lastCommandLine is `▸ cmd (exit N)`; bare `▸ cmd` while the exit is unknown.
func lastCommandLine(e *Entry) string {
	if e.LastCmd == "" {
		return ""
	}
	if e.Exit != ExitUnknown {
		return e.LastCmd + " (exit " + strconv.Itoa(e.Exit) + ")"
	}
	return e.LastCmd
}

// aiLine renders the AI-status section; a failed run surfaces the error text
// instead of a summary so the card never lies about state.
func aiLine(e *Entry) string {
	if e.Status == StatusError && e.Error != "" {
		return "error: " + e.Error
	}
	return e.AIStatus
}

// modelLine is `provider/model` (bare model when provider unknown).
func modelLine(e *Entry) string {
	switch {
	case e.Provider != "" && e.Model != "":
		return e.Provider + "/" + e.Model
	default:
		return e.Model
	}
}

func stampLine(e *Entry) string {
	if e.UpdatedAt == 0 {
		return ""
	}
	return secStamp(e.UpdatedAt)
}

// secStamp renders an epoch-seconds stamp (integral, no fraction noise).
func secStamp(v float64) string {
	return strconv.FormatInt(int64(v), 10)
}

// Push renders the entry and writes it into the workspace description slot the
// omp-summary sidebar reads. Race guard: the state file is re-read first and
// the push is skipped when any strictly newer entry already exists — a newer
// result pushes its own card right after, so ours would only flicker.
func Push(ctx context.Context, cwd, workspaceID string, e *Entry) error {
	if workspaceID == "" {
		return nil
	}
	if st := LoadState(cwd); st.NewerThan(e.UpdatedAt) {
		return nil
	}
	return cmux.SetDescription(ctx, workspaceID, Card(e))
}
