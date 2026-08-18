package dotfiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsJsoncToJson(t *testing.T) {
	tests := []struct {
		name   string
		source string
		dest   string
		want   bool
	}{
		{"jsonc to json", "foo.jsonc", "foo.json", true},
		{"json to json", "foo.json", "foo.json", false},
		{"jsonc to jsonc", "foo.jsonc", "foo.jsonc", false},
		{"txt to json", "foo.txt", "foo.json", false},
		{"jsonc to txt", "foo.jsonc", "foo.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Mapping{Source: tt.source, Destination: tt.dest}
			assert.Equal(t, tt.want, IsJsoncToJson(m))
		})
	}
}

func TestStripJSONC(t *testing.T) {
	t.Run("strips line comments", func(t *testing.T) {
		input := []byte(`{
  // This is a comment
  "key": "value"
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))

		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out, &m))
		assert.Equal(t, "value", m["key"])
	})

	t.Run("strips block comments", func(t *testing.T) {
		input := []byte(`{
  /* block comment */
  "key": "value"
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))

		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out, &m))
		assert.Equal(t, "value", m["key"])
	})

	t.Run("strips trailing commas", func(t *testing.T) {
		input := []byte(`{
  "a": 1,
  "b": 2,
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))

		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out, &m))
		assert.Equal(t, float64(1), m["a"])
		assert.Equal(t, float64(2), m["b"])
	})

	t.Run("strips trailing commas in arrays", func(t *testing.T) {
		input := []byte(`{
  "items": [1, 2, 3,]
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))
	})

	t.Run("preserves strings with slashes", func(t *testing.T) {
		input := []byte(`{
  "url": "https://example.com/path",
  "pattern": "a//b"
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))

		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out, &m))
		assert.Equal(t, "https://example.com/path", m["url"])
		assert.Equal(t, "a//b", m["pattern"])
	})

	t.Run("handles escaped quotes in strings", func(t *testing.T) {
		input := []byte(`{
  "msg": "he said \"hello\"" // comment
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))
	})

	t.Run("handles multiline block comments", func(t *testing.T) {
		input := []byte(`{
  /*
   * This is a
   * multiline comment
   */
  "key": "value"
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))
	})

	t.Run("handles all features together", func(t *testing.T) {
		input := []byte(`{
  // MCP server configuration
  "mcpServers": {
    /* Main server */
    "server1": {
      "command": "node",
      "args": ["--inspect", "server.js",],
    },
    // Backup server
    "server2": {
      "command": "python",
    },
  },
}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))

		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(out, &m))
		servers := m["mcpServers"].(map[string]interface{})
		assert.Contains(t, servers, "server1")
		assert.Contains(t, servers, "server2")
	})

	t.Run("returns error for invalid json after stripping", func(t *testing.T) {
		input := []byte(`{invalid`)
		_, err := StripJSONC(input)
		assert.Error(t, err)
	})

	t.Run("valid json passes through", func(t *testing.T) {
		input := []byte(`{"key": "value"}`)
		out, err := StripJSONC(input)
		require.NoError(t, err)
		assert.True(t, json.Valid(out))
	})
}

func TestCopyJsoncToJson(t *testing.T) {
	t.Run("copies jsonc to json", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "config.jsonc")
		dest := filepath.Join(tmpDir, "config.json")

		content := []byte(`{
  // Server config
  "host": "localhost",
  "port": 8080,
}`)
		require.NoError(t, os.WriteFile(source, content, 0644))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)

		assert.Equal(t, StateLinked, result.State)
		assert.Empty(t, result.Error)

		// Destination should be a regular file, not a symlink
		info, err := os.Lstat(dest)
		require.NoError(t, err)
		assert.Zero(t, info.Mode()&os.ModeSymlink)

		// Content should be valid JSON
		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.True(t, json.Valid(data))

		var m2 map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &m2))
		assert.Equal(t, "localhost", m2["host"])
		assert.Equal(t, float64(8080), m2["port"])
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "config.jsonc")
		dest := filepath.Join(tmpDir, "nested", "deep", "config.json")

		require.NoError(t, os.WriteFile(source, []byte(`{"key": "value"}`), 0644))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)

		assert.Equal(t, StateLinked, result.State)
		assert.True(t, result.CreatedDir)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "config.jsonc")
		dest := filepath.Join(tmpDir, "config.json")

		require.NoError(t, os.WriteFile(source, []byte(`{"new": true}`), 0644))
		require.NoError(t, os.WriteFile(dest, []byte(`{"old": true}`), 0644))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)

		assert.Equal(t, StateLinked, result.State)
		assert.True(t, result.RemovedExisting)

		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		var m2 map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &m2))
		assert.Equal(t, true, m2["new"])
	})

	t.Run("dest is symlink back to source — source must not be clobbered", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "config.jsonc")
		dest := filepath.Join(tmpDir, "config.json")

		orig := []byte("{\n  // keep me\n  \"a\": 1\n}")
		require.NoError(t, os.WriteFile(source, orig, 0644))
		require.NoError(t, os.Symlink(source, dest))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)
		require.Equal(t, StateLinked, result.State)

		gotSrc, err := os.ReadFile(source)
		require.NoError(t, err)
		assert.Equal(t, orig, gotSrc, "source jsonc must retain comments")

		info, err := os.Lstat(dest)
		require.NoError(t, err)
		assert.Zero(t, info.Mode()&os.ModeSymlink, "dest should be a regular file, not symlink")
	})

	t.Run("expands env vars", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "config.jsonc")
		dest := filepath.Join(tmpDir, "config.json")

		t.Setenv("CLY_TEST_HOME", "/Users/test")
		t.Setenv("CLY_TEST_PATH", "/opt/tool")

		content := []byte(`{
  "home": "${CLY_TEST_HOME}",
  "bin": "${CLY_TEST_PATH}/bin",
  "bare": "looks like a $CLY_TEST_HOME ref",
}`)
		require.NoError(t, os.WriteFile(source, content, 0644))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)
		require.Equal(t, StateLinked, result.State, "error: %s", result.Error)

		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.True(t, json.Valid(data))

		var m2 map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &m2))
		assert.Equal(t, "/Users/test", m2["home"])
		assert.Equal(t, "/opt/tool/bin", m2["bin"])
		assert.Equal(t, "looks like a /Users/test ref", m2["bare"])
	})

	t.Run("no-interpolation marker skips expansion", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "config.jsonc")
		dest := filepath.Join(tmpDir, "config.json")

		t.Setenv("CLY_TEST_HOME", "/Users/test")

		content := []byte(`// @no-interpolation
{
  "home": "${CLY_TEST_HOME}"
}`)
		require.NoError(t, os.WriteFile(source, content, 0644))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)
		require.Equal(t, StateLinked, result.State, "error: %s", result.Error)

		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.True(t, json.Valid(data))

		var m2 map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &m2))
		assert.Equal(t, "${CLY_TEST_HOME}", m2["home"])
	})

	t.Run("expansion that breaks JSON surfaces as error", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "config.jsonc")
		dest := filepath.Join(tmpDir, "config.json")

		// A value containing a quote would break the resulting JSON string.
		t.Setenv("CLY_TEST_BAD", `has "quote" inside`)
		content := []byte(`{"key": "${CLY_TEST_BAD}"}`)
		require.NoError(t, os.WriteFile(source, content, 0644))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)
		require.Equal(t, StateError, result.State)
		assert.Contains(t, result.Error, "expansion produced invalid JSON")
	})

	t.Run("missing source", func(t *testing.T) {
		tmpDir := t.TempDir()
		m := Mapping{
			Source:      filepath.Join(tmpDir, "nope.jsonc"),
			Destination: filepath.Join(tmpDir, "out.json"),
		}
		result := CopyJsoncToJson(m)
		assert.Equal(t, StateMissing, result.State)
	})

	t.Run("invalid jsonc content", func(t *testing.T) {
		tmpDir := t.TempDir()
		source := filepath.Join(tmpDir, "bad.jsonc")
		dest := filepath.Join(tmpDir, "bad.json")

		require.NoError(t, os.WriteFile(source, []byte(`{not valid at all`), 0644))

		m := Mapping{Source: source, Destination: dest}
		result := CopyJsoncToJson(m)
		assert.Equal(t, StateError, result.State)
		assert.Contains(t, result.Error, "not valid JSON")
	})
}

func TestApplyJsoncMapping_RespectsNoInterpolation(t *testing.T) {
	tmpDir := t.TempDir()

	confPath := filepath.Join(tmpDir, "dotfiles.conf")
	require.NoError(t, os.WriteFile(confPath, []byte("# empty config\n"), 0644))
	prevConfigFlag := configFlag
	configFlag = confPath
	t.Cleanup(func() { configFlag = prevConfigFlag })

	source := filepath.Join(tmpDir, "config.jsonc")
	dest := filepath.Join(tmpDir, "config.json")

	t.Setenv("CLY_TEST_DOTFILES_DIR", "/Users/test/dotfiles")

	content := []byte("// @no-interpolation\n{\n  \"home\": \"${CLY_TEST_DOTFILES_DIR}/foo\"\n}")
	require.NoError(t, os.WriteFile(source, content, 0644))

	m := Mapping{Source: source, Destination: dest}
	res, err := ApplyJsoncMapping(m)
	require.NoError(t, err)
	assert.Equal(t, StateLinked, res.State)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "${CLY_TEST_DOTFILES_DIR}/foo", got["home"])
}

func TestApplyJsoncMapping_ExpandsEnvVars(t *testing.T) {
	tmpDir := t.TempDir()

	// lockFilePath() derives the lock path from whatever dotfiles.conf the
	// loader picks up. Point it at a config inside t.TempDir() so the test
	// lock lands there instead of ~/DotFiles.
	confPath := filepath.Join(tmpDir, "dotfiles.conf")
	require.NoError(t, os.WriteFile(confPath, []byte("# empty config\n"), 0644))
	prevConfigFlag := configFlag
	configFlag = confPath
	t.Cleanup(func() { configFlag = prevConfigFlag })

	source := filepath.Join(tmpDir, "config.jsonc")
	dest := filepath.Join(tmpDir, "config.json")

	t.Setenv("CLY_TEST_DOTFILES_DIR", "/Users/test/dotfiles")

	content := []byte("{\n  // comment\n  \"home\": \"${CLY_TEST_DOTFILES_DIR}/foo\",\n}")
	require.NoError(t, os.WriteFile(source, content, 0644))

	m := Mapping{Source: source, Destination: dest}
	res, err := ApplyJsoncMapping(m)
	require.NoError(t, err)
	assert.Equal(t, StateLinked, res.State)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.True(t, json.Valid(data))

	var got map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "/Users/test/dotfiles/foo", got["home"])
}
