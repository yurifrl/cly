package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPiAdapterGetConfigPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	adapter := &PiAdapter{}

	userPath, err := adapter.GetConfigPath("user")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpHome, ".pi", "agent", "mcp.json"), userPath)

	projectPath, err := adapter.GetConfigPath("project")
	require.NoError(t, err)
	assert.Equal(t, ".pi/mcp.json", projectPath)

	_, err = adapter.GetConfigPath("local")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported scope for Pi")
}

func TestGetAdapterPi(t *testing.T) {
	adapter, err := GetAdapter("pi")
	require.NoError(t, err)
	assert.IsType(t, &PiAdapter{}, adapter)
}

func TestPiAdapterReadWriteConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	tmpProject := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	require.NoError(t, os.Chdir(tmpProject))

	adapter := &PiAdapter{}
	mcps := []MCP{{
		Name:    "github",
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
	}}

	require.NoError(t, adapter.WriteConfig("user", mcps))
	require.NoError(t, adapter.WriteConfig("project", mcps))

	userCfg, err := adapter.ReadConfig("user")
	require.NoError(t, err)
	require.Contains(t, userCfg.MCPServers, "github")
	assert.Equal(t, "npx", userCfg.MCPServers["github"].Command)

	projectCfg, err := adapter.ReadConfig("project")
	require.NoError(t, err)
	require.Contains(t, projectCfg.MCPServers, "github")
	assert.Equal(t, "npx", projectCfg.MCPServers["github"].Command)

	_, err = os.Stat(filepath.Join(tmpHome, ".pi", "agent", "mcp.json"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpProject, ".pi", "mcp.json"))
	require.NoError(t, err)
}
