package mut_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yurifrl/cly/pkg/mut"
)

func TestDryRunSkipsMutations(t *testing.T) {
	tmp := t.TempDir()
	var logs []string
	mut.Init(mut.Options{
		DryRun: true,
		Logger: func(action, target string) {
			logs = append(logs, action+" "+target)
		},
	})
	t.Cleanup(func() { mut.Init(mut.Options{}) })

	target := filepath.Join(tmp, "a", "b", "file")
	require.NoError(t, mut.MkdirAll(filepath.Dir(target), 0755))
	require.NoError(t, mut.WriteFile(target, []byte("hello"), 0644))
	require.NoError(t, mut.Symlink(target, target+".link"))
	require.NoError(t, mut.Remove(target))
	require.NoError(t, mut.Exec("false"))

	// Nothing was actually created.
	_, err := os.Stat(filepath.Join(tmp, "a"))
	assert.True(t, os.IsNotExist(err))

	assert.Len(t, logs, 5)
}

func TestRealModePerformsMutations(t *testing.T) {
	tmp := t.TempDir()
	mut.Init(mut.Options{DryRun: false})
	t.Cleanup(func() { mut.Init(mut.Options{}) })

	target := filepath.Join(tmp, "a", "b", "file")
	require.NoError(t, mut.MkdirAll(filepath.Dir(target), 0755))
	require.NoError(t, mut.WriteFile(target, []byte("hello"), 0644))

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestSetDryRunToggles(t *testing.T) {
	mut.Init(mut.Options{DryRun: false})
	t.Cleanup(func() { mut.Init(mut.Options{}) })

	assert.False(t, mut.DryRun())
	mut.SetDryRun(true)
	assert.True(t, mut.DryRun())
	mut.SetDryRun(false)
	assert.False(t, mut.DryRun())
}
