// Package helpy provides a registry-driven cheat sheet for cly's
// custom flags. Each flag registers itself at package init via
// Register; renderers walk All() to produce text or JSON output.
//
// helpy is rendered when a subcommand sees --helpy in its args. The
// registry is global because it must be populated by independent
// modules without an explicit wiring step.
package helpy

import (
	"sort"
	"sync"
)

// Entry describes a single custom flag or feature.
type Entry struct {
	// Section groups related entries together (e.g., "Naming",
	// "Session Import").
	Section string `json:"section"`

	// Flags lists the flag aliases this entry covers, in the order
	// they should appear in help output. Example: ["-n", "--name"].
	Flags []string `json:"flags"`

	// Value documents the value placeholder shown next to the flag.
	// Example: "<name>", "<UUID|prefix|path>".
	Value string `json:"value,omitempty"`

	// Description is human-readable prose. Multi-line allowed.
	Description string `json:"description"`

	// ConfigKeys lists the dotted config keys that influence this
	// flag's behavior. Pure documentation aid.
	ConfigKeys []string `json:"config_keys,omitempty"`

	// EnvVars lists environment variables this flag reads or writes.
	EnvVars []string `json:"env,omitempty"`

	// Requires lists other flags or conditions that must be present
	// when this flag is used.
	Requires []string `json:"requires,omitempty"`

	// Errors lists the error codes this flag may produce.
	Errors []string `json:"errors,omitempty"`

	// Examples is a list of full command lines demonstrating use.
	Examples []string `json:"examples,omitempty"`

	// Order controls relative ordering within the same Section.
	// Lower runs first. Equal values fall back to registration order.
	Order int `json:"-"`
}

var (
	mu      sync.Mutex
	entries []Entry
)

// Register adds an entry to the global registry. Safe to call from
// init functions across packages.
func Register(e Entry) {
	mu.Lock()
	defer mu.Unlock()
	entries = append(entries, e)
}

// All returns a copy of the registry sorted by Section, then Order.
// Sections preserve their first-registration appearance order.
func All() []Entry {
	mu.Lock()
	defer mu.Unlock()

	sectionOrder := map[string]int{}
	for i, e := range entries {
		if _, ok := sectionOrder[e.Section]; !ok {
			sectionOrder[e.Section] = i
		}
	}

	out := make([]Entry, len(entries))
	copy(out, entries)
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := sectionOrder[out[i].Section], sectionOrder[out[j].Section]
		if si != sj {
			return si < sj
		}
		return out[i].Order < out[j].Order
	})
	return out
}

// Reset clears the registry. Test helper.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	entries = nil
}
