package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkEntry(name string, weight int, def bool, cond string) Entry {
	e := Entry{Name: name, Provider: "openai", Weight: weight, Default: def, Condition: cond}
	if cond != "" {
		c, err := parseCondition(cond)
		if err != nil {
			panic(err)
		}
		e.cond = c
	}
	return e
}

func selCtx() *Context {
	return &Context{User: "yuri", Host: "mac", Arch: "arm64", OS: "darwin", Dir: "/Users/yuri/Workdir/Yuri/cly"}
}

func TestSelectConditionMatch(t *testing.T) {
	entries := []Entry{
		mkEntry("first", 0, true, ""),
		mkEntry("matched", 0, false, `user == "yuri"`),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "matched", e.Name)
	assert.Equal(t, "matched", d.Picked)
	assert.Equal(t, "condition match", d.Reason)
	assert.Equal(t, "picked", d.Entries[1].Note)
	assert.False(t, d.Entries[0].Matched)
}

func TestSelectWeightWins(t *testing.T) {
	entries := []Entry{
		mkEntry("low", 1, false, `user == "yuri"`),
		mkEntry("high", 10, false, `user == "yuri"`),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "high", e.Name)
	assert.Equal(t, "condition match", d.Reason)
}

func TestSelectWeightTiePicksFirst(t *testing.T) {
	entries := []Entry{
		mkEntry("a", 5, false, `user == "yuri"`),
		mkEntry("b", 5, false, `user == "yuri"`),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "a", e.Name)
}

func TestSelectZeroWeightTiePicksFirst(t *testing.T) {
	entries := []Entry{
		mkEntry("a", 0, false, `user == "yuri"`),
		mkEntry("b", 0, false, `user == "yuri"`),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "a", e.Name)
}

func TestSelectDefaultFallback(t *testing.T) {
	entries := []Entry{
		mkEntry("nomatch", 0, false, `user == "root"`),
		mkEntry("fallback", 0, true, ""),
		mkEntry("last", 0, false, ""),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "fallback", e.Name)
	assert.Equal(t, "default", d.Reason)
	assert.Equal(t, "default fallback", d.Entries[1].Note)
}

func TestSelectMultipleDefaultsPicksFirst(t *testing.T) {
	entries := []Entry{
		mkEntry("d1", 0, true, ""),
		mkEntry("d2", 0, true, ""),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "d1", e.Name)
}

func TestSelectNoDefaultPicksFirstEntry(t *testing.T) {
	entries := []Entry{
		mkEntry("first", 0, false, `user == "root"`),
		mkEntry("second", 0, false, ""),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "first", e.Name)
	assert.Equal(t, "first entry", d.Reason)
	assert.Equal(t, "first entry fallback", d.Entries[0].Note)
}

func TestSelectEntryWithoutConditionNeverMatches(t *testing.T) {
	entries := []Entry{
		mkEntry("nocond", 100, false, ""), // weight irrelevant: no condition = no match
		mkEntry("cond", 0, false, `user == "yuri"`),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "cond", e.Name)
}

func TestDecisionRecordsEnvRefs(t *testing.T) {
	t.Setenv("CLY_SEL_TEST", "x")
	entries := []Entry{
		mkEntry("a", 0, false, `env.CLY_SEL_TEST && env.CLY_SEL_UNSET`),
	}
	_, d := selectProvider(entries, selCtx())
	assert.Equal(t, map[string]bool{"CLY_SEL_TEST": true, "CLY_SEL_UNSET": false}, d.EnvRefs)
}

func TestBuildContext(t *testing.T) {
	c := buildContext()
	assert.NotEmpty(t, c.User)
	assert.NotEmpty(t, c.Host)
	assert.NotEmpty(t, c.Arch)
	assert.NotEmpty(t, c.OS)
	assert.NotEmpty(t, c.Dir)
	require.NotNil(t, c)
}
