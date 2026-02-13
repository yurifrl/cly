package agents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconcile_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.md")
	dst := filepath.Join(dir, "target", "output.md")

	require.NoError(t, os.WriteFile(src, []byte("# Hello"), 0644))

	plan := &SyncPlan{
		Items: []SyncItem{
			{Source: src, Target: dst, Transform: TransformNone},
		},
	}

	result, err := Reconcile(plan, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Written)
	assert.Equal(t, 0, result.Skipped)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "# Hello", string(data))
}

func TestReconcile_SkipsUnchanged(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.md")
	dst := filepath.Join(dir, "target.md")

	content := []byte("# Same content")
	require.NoError(t, os.WriteFile(src, content, 0644))
	require.NoError(t, os.WriteFile(dst, content, 0644))

	plan := &SyncPlan{
		Items: []SyncItem{
			{Source: src, Target: dst, Transform: TransformNone},
		},
	}

	result, err := Reconcile(plan, false)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Written)
	assert.Equal(t, 1, result.Skipped)
}

func TestReconcile_DryRun(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.md")
	dst := filepath.Join(dir, "target", "output.md")

	require.NoError(t, os.WriteFile(src, []byte("# Hello"), 0644))

	plan := &SyncPlan{
		Items: []SyncItem{
			{Source: src, Target: dst, Transform: TransformNone},
		},
	}

	result, err := Reconcile(plan, true)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Written)

	// File should NOT exist in dry-run
	_, err = os.Stat(dst)
	assert.True(t, os.IsNotExist(err))
}

func TestReconcile_JSONC(t *testing.T) {
	t.Setenv("TEST_RECONCILE_VAR", "works")

	dir := t.TempDir()
	src := filepath.Join(dir, "settings.jsonc")
	dst := filepath.Join(dir, "settings.json")

	require.NoError(t, os.WriteFile(src, []byte("{\n  // comment\n  \"val\": \"${TEST_RECONCILE_VAR}\"\n}"), 0644))

	plan := &SyncPlan{
		Items: []SyncItem{
			{Source: src, Target: dst, Transform: TransformJSONCSK},
		},
	}

	result, err := Reconcile(plan, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Written)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"works"`)
	assert.NotContains(t, string(data), "//")
}

func TestReconcile_SkillMD(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "SKILL.md")
	dst := filepath.Join(dir, "out", "SKILL.md")

	content := "---\nname: test\nallowed-tools:\n  - Read\n---\n# Skill"
	require.NoError(t, os.WriteFile(src, []byte(content), 0644))

	plan := &SyncPlan{
		Items: []SyncItem{
			{Source: src, Target: dst, Transform: TransformSkillMD},
		},
	}

	result, err := Reconcile(plan, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Written)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "allowed-tools")
	assert.Contains(t, string(data), "name: test")
}

func TestReconcile_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.md")
	dst := filepath.Join(dir, "a", "b", "c", "output.md")

	require.NoError(t, os.WriteFile(src, []byte("deep"), 0644))

	plan := &SyncPlan{
		Items: []SyncItem{
			{Source: src, Target: dst, Transform: TransformNone},
		},
	}

	_, err := Reconcile(plan, false)
	require.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "deep", string(data))
}
