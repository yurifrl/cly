package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeBin(t *testing.T, dir, name string) string {
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755))
	return p
}

func TestClaudeExec_HeadroomWhenInstalled(t *testing.T) {
	dir := t.TempDir()
	hr := fakeBin(t, dir, "headroom")
	fakeBin(t, dir, "claude")
	t.Setenv("PATH", dir)

	path, argv, err := claudeExec([]string{"-r", "x"}, execConfig{headroom: true})
	require.NoError(t, err)
	assert.Equal(t, hr, path)
	assert.Equal(t, []string{"headroom", "wrap", "claude", "--", "-r", "x"}, argv)

	_, argv, err = claudeExec(nil, execConfig{headroom: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"headroom", "wrap", "claude"}, argv)
}

func TestClaudeExec_FallsBackWithoutHeadroom(t *testing.T) {
	dir := t.TempDir()
	cl := fakeBin(t, dir, "claude")
	t.Setenv("PATH", dir)

	path, argv, err := claudeExec([]string{"-p"}, execConfig{headroom: true})
	require.NoError(t, err)
	assert.Equal(t, cl, path)
	assert.Equal(t, []string{"claude", "-p"}, argv)
}

func TestClaudeExec_DisabledSkipsHeadroom(t *testing.T) {
	dir := t.TempDir()
	fakeBin(t, dir, "headroom")
	cl := fakeBin(t, dir, "claude")
	t.Setenv("PATH", dir)

	path, argv, err := claudeExec([]string{"-p"}, execConfig{})
	require.NoError(t, err)
	assert.Equal(t, cl, path)
	assert.Equal(t, []string{"claude", "-p"}, argv)
}
