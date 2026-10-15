package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOmpAdapterGetConfigPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	adapter := &OmpAdapter{}

	userPath, err := adapter.GetConfigPath("user")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpHome, ".omp", "agent", "mcp.json"), userPath)

	localPath, err := adapter.GetConfigPath("local")
	require.NoError(t, err)
	assert.Equal(t, ".omp/mcp.json", localPath)

	projectPath, err := adapter.GetConfigPath("project")
	require.NoError(t, err)
	assert.Equal(t, ".omp/mcp.json", projectPath)

	_, err = adapter.GetConfigPath("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported scope for OMP")
}

func TestGetAdapterOmp(t *testing.T) {
	adapter, err := GetAdapter("omp")
	require.NoError(t, err)
	assert.IsType(t, &OmpAdapter{}, adapter)
}

func TestOmpAdapterReadWriteConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	tmpProject := t.TempDir()
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	require.NoError(t, os.Chdir(tmpProject))

	adapter := &OmpAdapter{}
	mcps := []MCP{{
		Name:    "github",
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-github"},
	}}

	require.NoError(t, adapter.WriteConfig("user", mcps))
	require.NoError(t, adapter.WriteConfig("local", mcps))

	userCfg, err := adapter.ReadConfig("user")
	require.NoError(t, err)
	require.Contains(t, userCfg.MCPServers, "github")
	assert.Equal(t, "npx", userCfg.MCPServers["github"].Command)

	localCfg, err := adapter.ReadConfig("local")
	require.NoError(t, err)
	require.Contains(t, localCfg.MCPServers, "github")
	assert.Equal(t, "npx", localCfg.MCPServers["github"].Command)

	_, err = os.Stat(filepath.Join(tmpHome, ".omp", "agent", "mcp.json"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpProject, ".omp", "mcp.json"))
	require.NoError(t, err)
}

func TestOmpAdapterReadPreservesOmpConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	mcpPath := filepath.Join(tmpHome, ".omp", "agent")
	require.NoError(t, os.MkdirAll(mcpPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(mcpPath, "mcp.json"), []byte(`{
  "mcpServers": {
    "rea": {
      "command": "npx",
      "args": ["-y", "rea-agents@5.0.0", "mcp"],
      "type": "stdio"
    }
  }
}`), 0644))

	adapter := &OmpAdapter{}
	cfg, err := adapter.ReadConfig("user")
	require.NoError(t, err)
	require.Contains(t, cfg.MCPServers, "rea")
	assert.Equal(t, "stdio", cfg.MCPServers["rea"].Type)
	assert.Equal(t, "rea", cfg.MCPServers["rea"].Name)

	// WriteConfig replaces mcpServers wholesale (caller passes full set)
	// but must preserve other top-level omp config keys
	require.NoError(t, os.WriteFile(filepath.Join(mcpPath, "mcp.json"), []byte(`{
  "disabledServers": ["legacy"],
  "mcpServers": {
    "rea": {
      "command": "npx",
      "args": ["-y", "rea-agents@5.0.0", "mcp"],
      "type": "stdio"
    }
  }
}`), 0644))

	mcps := []MCP{{Name: "extra", Type: "stdio", Command: "echo"}}
	require.NoError(t, adapter.WriteConfig("user", mcps))

	data, err := os.ReadFile(filepath.Join(mcpPath, "mcp.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"disabledServers"`)
	assert.Contains(t, string(data), `"extra"`)
}
