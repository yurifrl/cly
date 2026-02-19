package zl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsInsideZellij(t *testing.T) {
	t.Run("true when ZELLIJ is set", func(t *testing.T) {
		t.Setenv("ZELLIJ", "some-session-id")
		assert.True(t, IsInsideZellij())
	})

	t.Run("false when ZELLIJ is empty", func(t *testing.T) {
		t.Setenv("ZELLIJ", "")
		assert.False(t, IsInsideZellij())
	})
}

func TestBuildPluginArgs(t *testing.T) {
	t.Run("session only", func(t *testing.T) {
		got := BuildPluginArgs(SwitchOpts{Session: "work"})
		assert.Equal(t, "--session work", got)
	})

	t.Run("with cwd", func(t *testing.T) {
		got := BuildPluginArgs(SwitchOpts{Session: "work", Cwd: "/home/user"})
		assert.Equal(t, "--session work --cwd /home/user", got)
	})

	t.Run("with layout", func(t *testing.T) {
		got := BuildPluginArgs(SwitchOpts{Session: "work", Layout: "compact"})
		assert.Equal(t, "--session work --layout compact", got)
	})

	t.Run("all options", func(t *testing.T) {
		got := BuildPluginArgs(SwitchOpts{Session: "dev", Cwd: "/tmp", Layout: "default"})
		assert.Equal(t, "--session dev --cwd /tmp --layout default", got)
	})
}

func TestBuildAttachArgs(t *testing.T) {
	t.Run("session only", func(t *testing.T) {
		got := BuildAttachArgs(SwitchOpts{Session: "work"})
		assert.Equal(t, []string{"attach", "-c", "work"}, got)
	})
}

func TestParseSwitchFlags(t *testing.T) {
	t.Run("session name as positional", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"work"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.Empty(t, opts.Cwd)
		assert.Empty(t, opts.Layout)
		assert.False(t, opts.Window)
		assert.False(t, opts.Interactive)
		assert.False(t, opts.SaveMapping)
	})

	t.Run("session with cwd", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"work", "-c", "/home/user"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.Equal(t, "/home/user", opts.Cwd)
	})

	t.Run("session with layout", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"work", "-l", "compact"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.Equal(t, "compact", opts.Layout)
	})

	t.Run("session with window flag", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"work", "-w"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.True(t, opts.Window)
	})

	t.Run("session with interactive flag -i", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"work", "-i"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.True(t, opts.Interactive)
	})

	t.Run("session with interactive flag -z", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"work", "-z"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.True(t, opts.Interactive)
	})

	t.Run("session with save-mapping flag", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"work", "--save-mapping"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.True(t, opts.SaveMapping)
	})

	t.Run("all flags", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"dev", "-c", "/tmp", "-l", "default", "-w", "-i", "--save-mapping"})
		require.NoError(t, err)
		assert.Equal(t, "dev", opts.Session)
		assert.Equal(t, "/tmp", opts.Cwd)
		assert.Equal(t, "default", opts.Layout)
		assert.True(t, opts.Window)
		assert.True(t, opts.Interactive)
		assert.True(t, opts.SaveMapping)
	})

	t.Run("flags before positional", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"-l", "compact", "work"})
		require.NoError(t, err)
		assert.Equal(t, "work", opts.Session)
		assert.Equal(t, "compact", opts.Layout)
	})

	t.Run("no session returns empty", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{})
		require.NoError(t, err)
		assert.Empty(t, opts.Session)
	})

	t.Run("interactive only without session", func(t *testing.T) {
		opts, err := ParseSwitchFlags([]string{"-i"})
		require.NoError(t, err)
		assert.Empty(t, opts.Session)
		assert.True(t, opts.Interactive)
	})
}

func TestParseCurrentSession(t *testing.T) {
	t.Run("finds current session", func(t *testing.T) {
		// Simulates `zellij list-sessions` output with ANSI codes
		input := "\x1b[32mwork\x1b[0m (current)\ndev\ntest"
		got := ParseCurrentSession(input)
		assert.Equal(t, "work", got)
	})

	t.Run("returns empty when no current", func(t *testing.T) {
		input := "work\ndev\ntest"
		got := ParseCurrentSession(input)
		assert.Empty(t, got)
	})

	t.Run("handles empty input", func(t *testing.T) {
		got := ParseCurrentSession("")
		assert.Empty(t, got)
	})

	t.Run("handles ANSI in session name with extra info", func(t *testing.T) {
		input := "\x1b[1;32mmy-session\x1b[0m [Created 2h ago] (current)"
		got := ParseCurrentSession(input)
		assert.Equal(t, "my-session", got)
	})
}

func TestGenerateCompletions(t *testing.T) {
	out := GenerateCompletionsString()
	assert.Contains(t, out, "complete -c zl")
	assert.Contains(t, out, "switch")
	assert.Contains(t, out, "zellij list-sessions")
	assert.Contains(t, out, "nuke")
}
