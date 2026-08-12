package ai

import (
	"os"
	"strings"
)

// EntryResult records how one entry fared in selection.
type EntryResult struct {
	Name      string
	Condition string
	Matched   bool
	Weight    int
	Note      string
}

// Decision is the full record of a provider selection: what context was
// used, how each entry evaluated, and why the winner won.
type Decision struct {
	Context *Context
	EnvRefs map[string]bool
	Entries []EntryResult
	Picked  string
	Reason  string
}

// selectProvider picks the winning entry:
//  1. highest weight among condition matches (ties: list order)
//  2. first entry with default: true
//  3. first entry
func selectProvider(entries []Entry, ctx *Context) (Entry, *Decision) {
	d := &Decision{Context: ctx, EnvRefs: map[string]bool{}}
	d.Entries = make([]EntryResult, len(entries))
	for i, e := range entries {
		matched := e.cond != nil && e.cond.eval(ctx)
		d.Entries[i] = EntryResult{
			Name: e.Name, Condition: e.Condition, Matched: matched, Weight: e.Weight,
		}
		collectEnvRefs(e.Condition, d.EnvRefs)
	}

	best := -1
	for i, e := range entries {
		if !d.Entries[i].Matched {
			continue
		}
		if best == -1 || e.Weight > entries[best].Weight {
			best = i
		}
	}
	if best >= 0 {
		d.Entries[best].Note = "picked"
		d.Picked = entries[best].Name
		d.Reason = "condition match"
		return entries[best], d
	}

	for i, e := range entries {
		if e.Default {
			d.Entries[i].Note = "default fallback"
			d.Picked = e.Name
			d.Reason = "default"
			return entries[i], d
		}
	}

	d.Entries[0].Note = "first entry fallback"
	d.Picked = entries[0].Name
	d.Reason = "first entry"
	return entries[0], d
}

// collectEnvRefs records env var names mentioned in a condition and
// whether each is currently set (never records values).
func collectEnvRefs(cond string, refs map[string]bool) {
	for cond != "" {
		i := strings.Index(cond, "env.")
		if i < 0 {
			return
		}
		rest := cond[i+4:]
		j := 0
		for j < len(rest) && (rest[j] == '_' || (rest[j] >= 'A' && rest[j] <= 'Z') ||
			(rest[j] >= 'a' && rest[j] <= 'z') || (rest[j] >= '0' && rest[j] <= '9')) {
			j++
		}
		if j > 0 {
			name := rest[:j]
			if _, ok := refs[name]; !ok {
				refs[name] = os.Getenv(name) != ""
			}
		}
		cond = rest[j:]
	}
}
