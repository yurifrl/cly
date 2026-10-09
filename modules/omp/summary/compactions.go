package ompsummary

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// compactionRecord mirrors OMP's top-level {"type":"compaction",...} JSONL
// entry: OMP writes one per context fold. Verifying Type is load-bearing —
// the marker text also appears inside message content (assistant thinking
// quotes it verbatim), which must not count as a fold.
type compactionRecord struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Summary   string `json:"summary"`
}

// dropRe extracts the mechanical fold's only quantitative line, e.g.
// "About 326829 characters of older middle history dropped to fit archive
// budget." Mechanical HISTORY bodies are always empty — the prior transcript
// lives in the JSONL itself — so that figure is the only honest distill.
var dropRe = regexp.MustCompile(`About (\d+) characters of older middle history dropped`)

// Fold describes the latest OMP context fold found in a session transcript.
type Fold struct {
	Count   int     // total compaction records seen (append-ordered scan)
	At      float64 // latest fold's unix time
	Note    string  // mechanical one-line fallback: semantic handoff's first Goal line, or the dropped-history size
	Summary string  // latest fold's raw summary, head-capped for the LLM prompt
}

// ScanCompactions counts OMP compaction records in a session transcript and
// captures the latest fold's summary. Records are append-ordered, so the last
// record seen is the latest fold. The mechanical Note lets the card surface
// folds even when the AI round-trip errors; Summary feeds the LLM distillation
// in the summarizer. Returns the zero Fold when the transcript has no folds or
// is unreadable.
func ScanCompactions(path string) Fold {
	f, err := os.Open(path)
	if err != nil {
		return Fold{}
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // fold summaries exceed the default 64k cap
	var best compactionRecord
	var lastAt float64
	count := 0
	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.Contains(line, []byte(`"type":"compaction"`)) {
			continue
		}
		var rec compactionRecord
		if json.Unmarshal(line, &rec) != nil || rec.Type != "compaction" {
			continue
		}
		count++
		best = rec
		if at := parseRFC3339(rec.Timestamp); at > lastAt {
			lastAt = at
		}
	}
	if count == 0 {
		return Fold{}
	}
	// Head-cap: mechanical folds put the drop figure and FILES manifest near
	// the head; semantic handoffs put "## Goal" there. The boilerplate tail is
	// what we can afford to lose.
	return Fold{Count: count, At: lastAt, Note: foldNote(best.Summary), Summary: capRunes(best.Summary, 1200)}
}

// foldNote distills one compaction summary into a single card line.
func foldNote(summary string) string {
	if strings.HasPrefix(summary, "## Goal") {
		for _, ln := range strings.Split(summary, "\n")[1:] {
			if t := strings.TrimSpace(ln); t != "" {
				return stripMD(t)
			}
		}
	}
	if m := dropRe.FindStringSubmatch(summary); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return ""
		}
		if n >= 1000 {
			return "~" + strconv.Itoa((n+999)/1000) + "k older chars dropped"
		}
		return strconv.Itoa(n) + " older chars dropped"
	}
	return ""
}
