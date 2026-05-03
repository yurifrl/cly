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

// AnnotationSkipAlias is a cobra command annotation that disables alias
// generation for that command and its cobra aliases. Set it to "true" on
// commands whose names/aliases collide with external binaries you want to
// keep using (e.g. `bd` -> beads CLI).
//
//	cmd.Annotations = map[string]string{aliases.AnnotationSkipAlias: "true"}
const AnnotationSkipAlias = "cly.alias.skip"

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

		// Opt-out annotation: command explicitly declines alias generation
		// (for both its primary name and cobra aliases).
		if v, ok := cmd.Annotations[AnnotationSkipAlias]; ok && v == "true" {
			continue
		}

		// Only alias the command name if it doesn't shadow an existing binary
		if _, err := lookPath(name); err != nil {
			entries = append(entries, AliasEntry{
				Alias:   name,
				Command: fmt.Sprintf("cly %s", name),
			})
		}

		// Cobra aliases always get created (they're short/unique names like "c", "bkp").
		// Use the AnnotationSkipAlias above on the parent command to opt out.
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
