package aliases

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var skipCommands = map[string]bool{
	"help":       true,
	"completion": true,
	"i":          true, // too short, would pollute shell namespace
}

type AliasEntry struct {
	Alias   string
	Command string
}

// GenerateAliases walks root subcommands and produces alias entries.
// lookPath checks if a binary exists on PATH (pass exec.LookPath for real usage).
func GenerateAliases(root *cobra.Command, lookPath func(string) (string, error)) []AliasEntry {
	var entries []AliasEntry

	for _, cmd := range root.Commands() {
		name := cmd.Name()

		if skipCommands[name] {
			continue
		}

		// Only alias the command name if it doesn't shadow an existing binary
		if _, err := lookPath(name); err != nil {
			entries = append(entries, AliasEntry{
				Alias:   name,
				Command: fmt.Sprintf("cly %s", name),
			})
		}

		// Cobra aliases always get created (they're short/unique names like "c", "bkp")
		for _, a := range cmd.Aliases {
			if skipCommands[a] {
				continue
			}
			entries = append(entries, AliasEntry{
				Alias:   a,
				Command: fmt.Sprintf("cly %s", name),
			})
		}
	}

	return entries
}

func FormatFish(entries []AliasEntry) string {
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "alias %s \"%s\";\n", e.Alias, e.Command)
	}
	return b.String()
}

// FormatFishCompletions generates `complete -c <alias> -w '<command>'` wrappers.
// skip contains alias names that have custom completions registered elsewhere.
func FormatFishCompletions(entries []AliasEntry, skip ...string) string {
	skipSet := make(map[string]bool, len(skip))
	for _, s := range skip {
		skipSet[s] = true
	}

	var b strings.Builder
	for _, e := range entries {
		if skipSet[e.Alias] {
			continue
		}
		fmt.Fprintf(&b, "complete -c %s -w '%s'\n", e.Alias, e.Command)
	}
	return b.String()
}
