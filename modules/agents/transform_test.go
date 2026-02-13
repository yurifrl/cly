package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripJSONCComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no comments",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "single line comment",
			input: "{\n  // this is a comment\n  \"key\": \"value\"\n}",
			want:  "{\n  \n  \"key\": \"value\"\n}",
		},
		{
			name:  "inline comment",
			input: "{\n  \"key\": \"value\" // inline\n}",
			want:  "{\n  \"key\": \"value\" \n}",
		},
		{
			name:  "block comment",
			input: "{\n  /* block comment */\n  \"key\": \"value\"\n}",
			want:  "{\n  \n  \"key\": \"value\"\n}",
		},
		{
			name:  "multiline block comment",
			input: "{\n  /* multi\n     line */\n  \"key\": \"value\"\n}",
			want:  "{\n  \n  \"key\": \"value\"\n}",
		},
		{
			name:  "comment inside string preserved",
			input: `{"key": "http://example.com"}`,
			want:  `{"key": "http://example.com"}`,
		},
		{
			name:  "trailing commas removed",
			input: "{\n  \"a\": 1,\n  \"b\": 2,\n}",
			want:  "{\n  \"a\": 1,\n  \"b\": 2\n}",
		},
		{
			name:  "trailing comma in array",
			input: `{"arr": [1, 2, 3,]}`,
			want:  `{"arr": [1, 2, 3]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripJSONCComments(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInterpolateEnv(t *testing.T) {
	t.Setenv("TEST_HOME", "/home/user")
	t.Setenv("TEST_EDITOR", "vim")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no vars",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "single var",
			input: `{"home": "${TEST_HOME}"}`,
			want:  `{"home": "/home/user"}`,
		},
		{
			name:  "multiple vars",
			input: `{"home": "${TEST_HOME}", "editor": "${TEST_EDITOR}"}`,
			want:  `{"home": "/home/user", "editor": "vim"}`,
		},
		{
			name:  "unset var becomes empty",
			input: `{"val": "${UNSET_VAR_XXXXXX}"}`,
			want:  `{"val": ""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interpolateEnv(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasNoInterpolation(t *testing.T) {
	assert.True(t, hasNoInterpolation("// @no-interpolation\n{}\n"))
	assert.True(t, hasNoInterpolation("{\n// @no-interpolation\n}\n"))
	assert.False(t, hasNoInterpolation("{\"key\": \"value\"}\n"))

	// Past line 10 should not count
	lines := ""
	for i := 0; i < 12; i++ {
		lines += "// line\n"
	}
	lines += "// @no-interpolation\n"
	assert.False(t, hasNoInterpolation(lines))
}

func TestTransformJSONC(t *testing.T) {
	t.Setenv("TEST_VAL", "hello")

	input := []byte("{\n  // comment\n  \"key\": \"${TEST_VAL}\",\n}")
	got, err := TransformJSONC(input)
	require.NoError(t, err)

	assert.Contains(t, string(got), `"key"`)
	assert.Contains(t, string(got), `"hello"`)
	assert.NotContains(t, string(got), "//")
}

func TestTransformJSONC_NoInterpolation(t *testing.T) {
	t.Setenv("TEST_VAL", "hello")

	input := []byte("// @no-interpolation\n{\n  // comment\n  \"key\": \"${TEST_VAL}\"\n}")
	got, err := TransformJSONC(input)
	require.NoError(t, err)

	// Should strip comments but NOT interpolate env vars
	assert.Contains(t, string(got), `"${TEST_VAL}"`)
	assert.NotContains(t, string(got), "//")
}

func TestStripAllowedTools(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "strips allowed-tools block",
			input: `---
name: "test"
description: "a skill"
allowed-tools:
  - Read
  - Bash
---

# Content`,
			want: `---
name: "test"
description: "a skill"
---

# Content`,
		},
		{
			name: "no frontmatter",
			input: `# Just content
No frontmatter here.`,
			want: `# Just content
No frontmatter here.`,
		},
		{
			name: "frontmatter without allowed-tools",
			input: `---
name: "test"
---

# Content`,
			want: `---
name: "test"
---

# Content`,
		},
		{
			name: "allowed-tools with bracket syntax",
			input: `---
description: "test"
allowed-tools:
  [
    "Bash(git add:*)",
    "Bash(git commit:*)",
  ]
---

# Content`,
			want: `---
description: "test"
---

# Content`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripAllowedTools([]byte(tt.input))
			assert.Equal(t, tt.want, string(got))
		})
	}
}
