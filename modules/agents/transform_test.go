package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
