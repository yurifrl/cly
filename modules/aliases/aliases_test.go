package aliases

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func buildTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "cly"}

	root.AddCommand(&cobra.Command{Use: "uuid", Short: "Generate UUIDs"})
	root.AddCommand(&cobra.Command{Use: "dotfiles", Short: "Manage dotfiles"})
	root.AddCommand(&cobra.Command{
		Use:     "claude",
		Short:   "Run Claude",
		Aliases: []string{"c"},
	})
	root.AddCommand(&cobra.Command{
		Use:     "backup",
		Short:   "Backup stuff",
		Aliases: []string{"bkp"},
	})
	// help is a built-in cobra command, should be skipped
	return root
}

// lookPath that says "claude" exists on system
func mockLookPath(name string) (string, error) {
	if name == "claude" {
		return "/usr/local/bin/claude", nil
	}
	return "", fmt.Errorf("not found")
}

func TestGenerateAliases(t *testing.T) {
	root := buildTestRoot()
	entries := GenerateAliases(root, mockLookPath)

	aliases := map[string]string{}
	for _, e := range entries {
		aliases[e.Alias] = e.Command
	}

	// uuid doesn't exist on PATH, gets an alias
	assert.Equal(t, "cly uuid", aliases["uuid"])

	// dotfiles doesn't exist on PATH, gets an alias
	assert.Equal(t, "cly dotfiles", aliases["dotfiles"])

	// claude EXISTS on PATH, so NO direct alias
	_, hasClaude := aliases["claude"]
	assert.False(t, hasClaude, "claude should be skipped because it exists on PATH")

	// but cobra alias "c" should still be created
	assert.Equal(t, "cly claude", aliases["c"])

	// backup doesn't exist, gets both alias and cobra alias
	assert.Equal(t, "cly backup", aliases["backup"])
	assert.Equal(t, "cly backup", aliases["bkp"])

	// help should not appear
	_, hasHelp := aliases["help"]
	assert.False(t, hasHelp)
}

func TestFormatFish(t *testing.T) {
	entries := []AliasEntry{
		{Alias: "uuid", Command: "cly uuid"},
		{Alias: "c", Command: "cly claude"},
	}

	out := FormatFish(entries)
	assert.Contains(t, out, `alias uuid "cly uuid"`)
	assert.Contains(t, out, `alias c "cly claude"`)
}

func TestFormatFishCompletions(t *testing.T) {
	entries := []AliasEntry{
		{Alias: "uuid", Command: "cly uuid"},
		{Alias: "c", Command: "cly claude"},
		{Alias: "zl", Command: "cly zl"},
	}

	out := FormatFishCompletions(entries)
	assert.Contains(t, out, "complete -c uuid -w 'cly uuid'")
	assert.Contains(t, out, "complete -c c -w 'cly claude'")
	assert.Contains(t, out, "complete -c zl -w 'cly zl'")
}

func TestFormatFishCompletionsSkipsOverrides(t *testing.T) {
	entries := []AliasEntry{
		{Alias: "uuid", Command: "cly uuid"},
		{Alias: "zl", Command: "cly zl"},
	}

	out := FormatFishCompletions(entries, "zl", "mcp")
	assert.Contains(t, out, "complete -c uuid -w 'cly uuid'")
	assert.NotContains(t, out, "complete -c zl")
}

func TestSkipHelpAndCompletion(t *testing.T) {
	root := buildTestRoot()
	root.AddCommand(&cobra.Command{Use: "completion", Short: "Generate completions"})

	entries := GenerateAliases(root, mockLookPath)
	for _, e := range entries {
		assert.NotEqual(t, "help", e.Alias, "help command should not be aliased")
		assert.NotEqual(t, "completion", e.Alias, "completion command should not be aliased")
	}
}

func TestSkipAnnotationDisablesAllAliases(t *testing.T) {
	root := &cobra.Command{Use: "cly"}
	root.AddCommand(&cobra.Command{
		Use:     "beads",
		Short:   "Beads TUI",
		Aliases: []string{"bd"},
		Annotations: map[string]string{
			AnnotationSkipAlias: "true",
		},
	})

	noop := func(string) (string, error) { return "", fmt.Errorf("not found") }

	entries := GenerateAliases(root, noop)
	for _, e := range entries {
		assert.NotEqual(t, "beads", e.Alias)
		assert.NotEqual(t, "bd", e.Alias)
	}
}
