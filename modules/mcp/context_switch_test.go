package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwitchContextPi(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	require.NoError(t, os.MkdirAll(filepath.Join(tmpHome, ".pi"), 0755))

	m := Model{}
	cmd := m.switchContext("pi", "user")
	require.NotNil(t, cmd)

	msg := cmd()
	switched, ok := msg.(contextSwitchedMsg)
	require.True(t, ok)
	assert.Equal(t, "pi", switched.newContext.AI)
	assert.Equal(t, "user", switched.newContext.Scope)
	assert.NotNil(t, switched.adapter)
}
