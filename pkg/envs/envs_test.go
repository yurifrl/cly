package envs_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yurifrl/cly/pkg/envs"
)

// withSource swaps the active source for the duration of t and
// restores the previous one afterwards.
func withSource(t *testing.T, seed map[string]string) *envs.MapSource {
	t.Helper()
	src := envs.NewMapSource(seed)
	prev := envs.Use(src)
	t.Cleanup(func() { envs.Use(prev) })
	return src
}

// -----------------------------------------------------------------------------
// Source contract
// -----------------------------------------------------------------------------

func TestUse_RestoresPrevious(t *testing.T) {
	first := envs.NewMapSource(nil)
	second := envs.NewMapSource(nil)

	prev1 := envs.Use(first)
	prev2 := envs.Use(second)
	assert.Same(t, first, prev2, "second swap must return first as previous")

	envs.Use(prev2)
	envs.Use(prev1)
}

func TestUse_NilResetsToOSSource(t *testing.T) {
	prev := envs.Use(envs.NewMapSource(nil))
	t.Cleanup(func() { envs.Use(prev) })

	// Passing nil must not nil out the source — it should reset to OS.
	envs.Use(nil)
	assert.NotPanics(t, func() { _ = envs.SessionName() })
}

func TestMapSource_LookupSetUnset(t *testing.T) {
	s := envs.NewMapSource(map[string]string{"FOO": "bar"})

	v, ok := s.Lookup("FOO")
	assert.True(t, ok)
	assert.Equal(t, "bar", v)

	require.NoError(t, s.Set("BAZ", "qux"))
	v, ok = s.Lookup("BAZ")
	assert.True(t, ok)
	assert.Equal(t, "qux", v)

	s.Unset("FOO")
	_, ok = s.Lookup("FOO")
	assert.False(t, ok)
}

// NewMapSource must defensively copy the seed.
func TestMapSource_SeedIsCopied(t *testing.T) {
	seed := map[string]string{"FOO": "bar"}
	s := envs.NewMapSource(seed)
	seed["FOO"] = "mutated"

	v, _ := s.Lookup("FOO")
	assert.Equal(t, "bar", v)
}

// -----------------------------------------------------------------------------
// Session: canonical + legacy alias semantics
// -----------------------------------------------------------------------------

func TestSessionName_CanonicalWins(t *testing.T) {
	withSource(t, map[string]string{
		"CLY_SESSION_NAME":    "canonical",
		"CLAUDE_SESSION_NAME": "legacy",
	})

	got, err := envs.SessionName().Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "canonical", got)
}

func TestSessionName_FallsBackToLegacy(t *testing.T) {
	withSource(t, map[string]string{"CLAUDE_SESSION_NAME": "legacy"})

	got, err := envs.SessionName().Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "legacy", got)
}

func TestSessionName_UnsetIsEmpty(t *testing.T) {
	withSource(t, nil)
	r := envs.SessionName()
	assert.True(t, r.IsEmpty())
}

// Empty-string env values must be treated as Empty, not Ok("").
func TestSessionName_EmptyStringIsEmpty(t *testing.T) {
	withSource(t, map[string]string{"CLY_SESSION_NAME": ""})
	r := envs.SessionName()
	assert.True(t, r.IsEmpty(), "empty string env value must surface as Empty")
}

func TestSetSessionName_WritesBoth(t *testing.T) {
	src := withSource(t, nil)

	require.NoError(t, envs.SetSessionName("session-x"))

	v1, ok1 := src.Lookup("CLY_SESSION_NAME")
	v2, ok2 := src.Lookup("CLAUDE_SESSION_NAME")

	assert.True(t, ok1)
	assert.Equal(t, "session-x", v1)
	assert.True(t, ok2, "legacy alias must be written too")
	assert.Equal(t, "session-x", v2)
}

func TestHasSessionName(t *testing.T) {
	src := withSource(t, nil)

	assert.False(t, envs.HasSessionName())

	require.NoError(t, src.Set("CLAUDE_SESSION_NAME", "x"))
	assert.True(t, envs.HasSessionName(), "legacy alias presence must register")
}

func TestUnsetSessionName_ClearsBoth(t *testing.T) {
	src := withSource(t, map[string]string{
		"CLY_SESSION_NAME":    "a",
		"CLAUDE_SESSION_NAME": "b",
	})

	envs.UnsetSessionName()

	_, ok1 := src.Lookup("CLY_SESSION_NAME")
	_, ok2 := src.Lookup("CLAUDE_SESSION_NAME")
	assert.False(t, ok1)
	assert.False(t, ok2)
}

// -----------------------------------------------------------------------------
// cmux + zellij presence helpers
// -----------------------------------------------------------------------------

func TestCmux(t *testing.T) {
	withSource(t, map[string]string{
		"CMUX_SURFACE_ID":   "surface-uuid",
		"CMUX_TAB_ID":       "tab-uuid",
		"CMUX_WORKSPACE_ID": "workspace-uuid",
	})

	sid, err := envs.CmuxSurfaceID().Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "surface-uuid", sid)

	tid, err := envs.CmuxTabID().Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "tab-uuid", tid)

	wid, err := envs.CmuxWorkspaceID().Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "workspace-uuid", wid)

	assert.True(t, envs.InCmux())
}

func TestCmux_Absent(t *testing.T) {
	withSource(t, nil)
	assert.True(t, envs.CmuxSurfaceID().IsEmpty())
	assert.False(t, envs.InCmux())
}

func TestZellij(t *testing.T) {
	withSource(t, map[string]string{
		"ZELLIJ":              "0.40.1",
		"ZELLIJ_SESSION_NAME": "main",
	})

	v, _ := envs.Zellij().Unwrap()
	assert.Equal(t, "0.40.1", v)

	s, _ := envs.ZellijSession().Unwrap()
	assert.Equal(t, "main", s)

	assert.True(t, envs.InZellij())
}

func TestZellij_Absent(t *testing.T) {
	withSource(t, nil)
	assert.False(t, envs.InZellij())
	assert.True(t, envs.Zellij().IsEmpty())
	assert.True(t, envs.ZellijSession().IsEmpty())
}

// -----------------------------------------------------------------------------
// Bool parsing — ClaudeVerbose covers readBool's contract
// -----------------------------------------------------------------------------

func TestClaudeVerbose_Valid(t *testing.T) {
	cases := map[string]bool{
		"1":     true,
		"true":  true,
		"TRUE":  true,
		"True":  true,
		"0":     false,
		"false": false,
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			withSource(t, map[string]string{"CLAUDE_VERBOSE": raw})
			got, err := envs.ClaudeVerbose().Unwrap()
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestClaudeVerbose_InvalidIsParseError(t *testing.T) {
	withSource(t, map[string]string{"CLAUDE_VERBOSE": "yeah"})

	r := envs.ClaudeVerbose()
	require.True(t, r.IsError())

	var pe *envs.ParseError
	require.True(t, errors.As(r.Error(), &pe), "Error must be *ParseError")
	assert.Equal(t, "CLAUDE_VERBOSE", pe.Name)
	assert.Equal(t, "bool", pe.Kind)
	assert.Equal(t, "yeah", pe.Value)
	assert.NotNil(t, pe.Cause)

	// errors.Is must work via Unwrap on ParseError.
	assert.ErrorIs(t, r.Error(), pe.Cause)
}

func TestClaudeVerbose_AbsentIsEmpty(t *testing.T) {
	withSource(t, nil)
	assert.True(t, envs.ClaudeVerbose().IsEmpty())
}

// -----------------------------------------------------------------------------
// Sound
// -----------------------------------------------------------------------------

func TestSound(t *testing.T) {
	withSource(t, map[string]string{"SOUND": "bell.aiff"})
	got, err := envs.Sound().Unwrap()
	require.NoError(t, err)
	assert.Equal(t, "bell.aiff", got)
}

// -----------------------------------------------------------------------------
// ParseError shape
// -----------------------------------------------------------------------------

func TestParseError_String(t *testing.T) {
	cause := errors.New("syntax")
	pe := &envs.ParseError{Name: "X", Kind: "bool", Value: "yeah", Cause: cause}
	assert.Contains(t, pe.Error(), "X")
	assert.Contains(t, pe.Error(), "yeah")
	assert.Contains(t, pe.Error(), "bool")
	assert.Contains(t, pe.Error(), "syntax")
	assert.Same(t, cause, pe.Unwrap())
}
