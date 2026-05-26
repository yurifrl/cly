package every

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// MaxLogBytes is the soft cap for an NDJSON log file. When exceeded, the
// first ~50% of lines are dropped and the file is rewritten.
const MaxLogBytes = 1 << 20 // 1 MiB

// LogRetention is the per-line max age used when trimming.
const LogRetention = 24 * time.Hour

// Event is one NDJSON record. Optional fields are stored in Extra and merged
// into the JSON object on encode.
type Event struct {
	TS    time.Time      `json:"ts"`
	Event string         `json:"event"`
	Extra map[string]any `json:"-"`
}

// MarshalJSON flattens Extra into the top-level object.
func (e Event) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"ts":    e.TS.UTC().Format(time.RFC3339Nano),
		"event": e.Event,
	}
	for k, v := range e.Extra {
		if k == "ts" || k == "event" {
			continue
		}
		m[k] = v
	}
	return json.Marshal(m)
}

// UnmarshalJSON populates TS, Event and Extra from a generic object.
func (e *Event) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if v, ok := m["ts"].(string); ok {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err == nil {
			e.TS = t
		}
	}
	if v, ok := m["event"].(string); ok {
		e.Event = v
	}
	delete(m, "ts")
	delete(m, "event")
	e.Extra = m
	return nil
}

// AppendLog appends one event to <path>. Atomicity within a single write is
// good enough — appends are line-oriented and small.
func AppendLog(path string, ev Event) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// MaybeTrimLog enforces the size + retention rules. Cheap to call; only does
// real work when the file exceeds the size cap or when retention is non-zero
// and force is true.
func MaybeTrimLog(path string, now time.Time, force bool) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	tooBig := info.Size() > MaxLogBytes
	if !tooBig && !force {
		return nil
	}
	events, _ := ReadLog(path)
	cutoff := now.Add(-LogRetention)
	var kept []Event
	for _, e := range events {
		if !e.TS.IsZero() && e.TS.Before(cutoff) {
			continue
		}
		kept = append(kept, e)
	}
	if tooBig {
		// drop in halves until we're under the cap.
		for len(kept) > 1 {
			kept = kept[len(kept)/2:]
			size := 0
			for _, e := range kept {
				if b, err := json.Marshal(e); err == nil {
					size += len(b) + 1
				}
			}
			if size <= MaxLogBytes/2 {
				break
			}
		}
	}
	return rewriteLog(path, kept)
}

// ReadLog parses a log file, skipping malformed lines. Returns events in
// file order (oldest → newest).
func ReadLog(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip malformed
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

func rewriteLog(path string, events []Event) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, e := range events {
		data, err := json.Marshal(e)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		w.Write(data)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Sync()
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// FilterSince returns events whose ts >= cutoff (or whose ts is zero).
func FilterSince(events []Event, cutoff time.Time) []Event {
	if cutoff.IsZero() {
		return events
	}
	out := events[:0:0]
	for _, e := range events {
		if e.TS.IsZero() || !e.TS.Before(cutoff) {
			out = append(out, e)
		}
	}
	return out
}

// LastN returns the trailing n events (or all of them if n >= len).
func LastN(events []Event, n int) []Event {
	if n <= 0 || n >= len(events) {
		return events
	}
	return events[len(events)-n:]
}

// SortByTS is a defensive helper; logs are normally already sorted by append
// order. Used only when merging multiple sources.
func SortByTS(events []Event) {
	sort.SliceStable(events, func(i, j int) bool { return events[i].TS.Before(events[j].TS) })
}
