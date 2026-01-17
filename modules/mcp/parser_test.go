package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMCPFile_FlatFormat(t *testing.T) {
	data := []byte(`{
		"server1": {
			"command": "npx",
			"args": ["-y", "some-package"]
		},
		"server2": {
			"command": "uvx",
			"args": ["another-package"]
		}
	}`)

	mcps, err := ParseMCPFile("test.json", data)
	require.NoError(t, err)
	assert.Len(t, mcps, 2)
	assert.Equal(t, "npx", mcps["server1"].Command)
	assert.Equal(t, "uvx", mcps["server2"].Command)
}

func TestParseMCPFile_McpServersWrapper(t *testing.T) {
	data := []byte(`{
		"mcpServers": {
			"server1": {
				"command": "npx",
				"args": ["-y", "some-package"]
			},
			"server2": {
				"command": "uvx",
				"args": ["another-package"]
			}
		}
	}`)

	mcps, err := ParseMCPFile("test.json", data)
	require.NoError(t, err)
	assert.Len(t, mcps, 2)
	assert.Equal(t, "npx", mcps["server1"].Command)
	assert.Equal(t, "uvx", mcps["server2"].Command)
}

func TestParseMCPFile_JSONC_WithComments(t *testing.T) {
	data := []byte(`{
		// This is a comment
		"server1": {
			"command": "npx",
			"args": ["-y", "some-package"]
		}
		/* block comment */
	}`)

	mcps, err := ParseMCPFile("test.jsonc", data)
	require.NoError(t, err)
	assert.Len(t, mcps, 1)
	assert.Equal(t, "npx", mcps["server1"].Command)
}

func TestParseMCPFile_JSONC_McpServersWrapper(t *testing.T) {
	data := []byte(`{
		// MCP servers config
		"mcpServers": {
			"server1": {
				"command": "npx",
				"args": ["-y", "pkg"]
			}
		}
	}`)

	mcps, err := ParseMCPFile("test.jsonc", data)
	require.NoError(t, err)
	assert.Len(t, mcps, 1)
	assert.Equal(t, "npx", mcps["server1"].Command)
}

func TestParseMCPFile_TrailingCommas(t *testing.T) {
	data := []byte(`{
		"server1": {
			"command": "npx",
			"args": ["-y", "pkg",],
		},
	}`)

	mcps, err := ParseMCPFile("test.jsonc", data)
	require.NoError(t, err)
	assert.Len(t, mcps, 1)
}

func TestParseMCPFile_YAML(t *testing.T) {
	data := []byte(`
server1:
  command: npx
  args:
    - "-y"
    - some-package
server2:
  command: uvx
  args:
    - another-package
`)

	mcps, err := ParseMCPFile("test.yaml", data)
	require.NoError(t, err)
	assert.Len(t, mcps, 2)
	assert.Equal(t, "npx", mcps["server1"].Command)
}
