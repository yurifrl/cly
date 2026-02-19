package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAgentsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create source .agents structure
	src := filepath.Join(dir, ".agents")
	dirs := []string{
		filepath.Join(src, "commands"),
		filepath.Join(src, "agents"),
		filepath.Join(src, "skills", "my-skill"),
		filepath.Join(src, "ides", "claude"),
		filepath.Join(src, "ides", "opencode"),
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(d, 0755))
	}

	// Files
	files := map[string]string{
		filepath.Join(src, "commands", "test.md"):          "# test command",
		filepath.Join(src, "agents", "helper.md"):          "# helper agent",
		filepath.Join(src, "skills", "my-skill", "SKILL.md"): "---\nname: my-skill\nallowed-tools:\n  - Read\n---\n# Skill",
		filepath.Join(src, "AGENTS.md"):                    "# Agents config",
		filepath.Join(src, "ides", "claude", "settings.jsonc"): "{\n  // claude settings\n  \"key\": \"val\"\n}",
		filepath.Join(src, "ides", "opencode", "opencode.json"): `{"key": "val"}`,
	}
	for path, content := range files {
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	return dir
}

func TestDiscover_Claude(t *testing.T) {
	dir := setupAgentsDir(t)
	src := filepath.Join(dir, ".agents")
	targetBase := filepath.Join(dir, ".claude")

	plan, err := Discover(src, GetIDEDef("claude"), targetBase)
	require.NoError(t, err)

	// Should have: commands/test.md, agents/helper.md, skills/my-skill/SKILL.md,
	// AGENTS.md→CLAUDE.md, ides/claude/settings.jsonc
	assert.True(t, len(plan.Items) >= 4, "expected at least 4 items, got %d", len(plan.Items))

	// Check AGENTS.md → CLAUDE.md rename
	found := false
	for _, item := range plan.Items {
		if filepath.Base(item.Target) == "CLAUDE.md" {
			found = true
			assert.Equal(t, TransformNone, item.Transform)
		}
	}
	assert.True(t, found, "AGENTS.md should map to CLAUDE.md for claude")

	// Check IDE-specific file (settings.jsonc)
	foundSettings := false
	for _, item := range plan.Items {
		if filepath.Base(item.Target) == "settings.json" {
			foundSettings = true
			assert.Equal(t, TransformJSONCSK, item.Transform)
		}
	}
	assert.True(t, foundSettings, "settings.jsonc should produce settings.json")
}

func TestDiscover_OpenCode(t *testing.T) {
	dir := setupAgentsDir(t)
	src := filepath.Join(dir, ".agents")
	targetBase := filepath.Join(dir, ".opencode")

	plan, err := Discover(src, GetIDEDef("opencode"), targetBase)
	require.NoError(t, err)

	// Check dir renames: commands→command, agents→agent, skills→skill
	for _, item := range plan.Items {
		rel, _ := filepath.Rel(targetBase, item.Target)
		parts := filepath.SplitList(rel)
		if len(parts) > 0 {
			first := filepath.Dir(rel)
			assert.NotEqual(t, "commands", first, "should use 'command' not 'commands'")
			assert.NotEqual(t, "agents", first, "should use 'agent' not 'agents'")
		}
	}

	// Check SKILL.md gets TransformSkillMD for opencode
	for _, item := range plan.Items {
		if filepath.Base(item.Source) == "SKILL.md" {
			assert.Equal(t, TransformSkillMD, item.Transform)
		}
	}
}

func TestDiscover_Bidirectional(t *testing.T) {
	dir := setupAgentsDir(t)
	src := filepath.Join(dir, ".agents")
	targetBase := filepath.Join(dir, ".claude")

	plan, err := Discover(src, GetIDEDef("claude"), targetBase)
	require.NoError(t, err)

	for _, item := range plan.Items {
		switch item.Transform {
		case TransformNone:
			assert.True(t, item.Bidirectional, "TransformNone items should be bidirectional: %s", item.Source)
		case TransformJSONCSK, TransformSkillMD:
			assert.False(t, item.Bidirectional, "transformed items should not be bidirectional: %s", item.Source)
		}
	}

	// Check reverse map has entries
	assert.NotEmpty(t, plan.ReverseMap, "ReverseMap should have bidirectional entries")
	for target, source := range plan.ReverseMap {
		assert.NotEmpty(t, target)
		assert.NotEmpty(t, source)
	}
}

func TestDiscover_EmptySource(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, ".agents")
	require.NoError(t, os.MkdirAll(src, 0755))

	plan, err := Discover(src, GetIDEDef("claude"), filepath.Join(dir, ".claude"))
	require.NoError(t, err)
	assert.Empty(t, plan.Items)
}
