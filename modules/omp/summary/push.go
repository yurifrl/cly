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
	// Positional wire format (card v2). cmux's sidebar interpreter mangles
	// multibyte string surgery — hasPrefix and dropFirst mis-split the
	// two-rune glyph tags — so the card carries no tags at all: segments
	// join with "|" and the renderer addresses them strictly by position.
	//   0 title | 1 stamp | 2 ask | 3 goal | 4 response | 5 action | 6 compaction | 7+ timeline
	// Empty hero slots render "~"; the sidebar hides single-char rows.
	timelineMax = 5 // matches state's Timeline cap (oldest→newest)
	cardCap     = 900 // total card budget in runes (wrapped timeline rows need room)
	lineCap     = 140 // per-line budget in runes
)

// pad fills empty segments with the "~" placeholder so positions stay fixed.
func pad(segs []string) {
	for i, s := range segs {
		if s == "" {
			segs[i] = "~"
		}
	}
}

// sanitize strips row-structure hazards from a payload: "|" would spawn a
// fake row, newlines get squashed by cmux anyway, and a literal "~" would be
// hidden as padding ("~ " keeps it visible).
func sanitize(text string) string {
	text = strings.ReplaceAll(text, "|", "/")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	if text == "~" {
		text = "~ "
	}
	return text
}

// Card renders an entry into the positional sidebar format: fixed slots
// joined with "|", empty slots padded with "~" so the renderer can address
// segments by position alone. Timeline rows follow the hero slots in
// chronological order. The whole card is capped at cardCap runes so a long
// command or summary can never blow out the narrow sidebar.
func Card(e *Entry) string {
	head := e.Title
	if head == "" {
		head = e.SurfaceID
	}
	if head == "" {
		return ""
	}
	segs := []string{
		sanitize(capRunes(head, lineCap)),
		sanitize(stampLine(e)),
		sanitize(capRunes(e.UserAsk, lineCap)),
		sanitize(capRunes(e.Goal, lineCap)),
		sanitize(capRunes(e.Response, lineCap)),
		sanitize(capRunes(e.Action, lineCap)),
		sanitize(capRunes(foldLine(e), lineCap)),
	}
	pad(segs)
	for _, turn := range e.Timeline {
		if len(segs) >= 7+timelineMax {
			break
		}
		if row := sanitize(capRunes(turn.Text, lineCap)); row != "" {
			segs = append(segs, row)
		}
	}
	return capRunes(strings.Join(segs, "|"), cardCap)
}

func stampLine(e *Entry) string {
	if e.UpdatedAt == 0 {
		return ""
	}
	return relStamp(e.UpdatedAt, nowUnix())
}

// foldLine renders the compaction row: "<n> folds · last <rel> · <note>".
// Zero folds renders empty (padded to "~" and hidden by the sidebar).
func foldLine(e *Entry) string {
	if e.Compactions == 0 {
		return ""
	}
	parts := []string{strconv.Itoa(e.Compactions) + " folds"}
	if e.CompactedAt > 0 {
		parts = append(parts, "last "+relStamp(e.CompactedAt, nowUnix()))
	}
	if e.FoldNote != "" {
		parts = append(parts, e.FoldNote)
	}
	return strings.Join(parts, " · ")
}

// relStamp renders an epoch-seconds stamp as compact relative time. The
// daemon re-pushes the card every tick, so the value stays live.
func relStamp(v, now float64) string {
	d := now - v
	if d < 0 {
		d = 0
	}
	switch {
	case d < 60:
		return "now"
	case d < 3600:
		return strconv.Itoa(int(d/60)) + "m"
	case d < 86400:
		return strconv.Itoa(int(d/3600)) + "h"
	default:
		return strconv.Itoa(int(d/86400)) + "d"
	}
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
