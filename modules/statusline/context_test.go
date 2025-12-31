package statusline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateTokens(t *testing.T) {
	usage := &CurrentUsage{
		InputTokens:              5000,
		CacheReadInputTokens:     2000,
		CacheCreationInputTokens: 1000,
	}
	assert.Equal(t, 8000, CalculateTokens(usage))
}

func TestCalculateTokens_Nil(t *testing.T) {
	assert.Equal(t, 0, CalculateTokens(nil))
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		tokens int
		max    int
		want   int
	}{
		{0, 200000, 0},
		{100000, 200000, 50},
		{200000, 200000, 100},
		{150000, 200000, 75},
	}
	for _, tt := range tests {
		got := CalculatePercentage(tt.tokens, tt.max)
		assert.Equal(t, tt.want, got, "tokens=%d, max=%d", tt.tokens, tt.max)
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		tokens int
		want   string
	}{
		{0, "0K"},
		{500, "0K"},
		{1000, "1K"},
		{90000, "90K"},
		{200000, "200K"},
	}
	for _, tt := range tests {
		got := FormatTokens(tt.tokens)
		assert.Equal(t, tt.want, got, "tokens=%d", tt.tokens)
	}
}

func TestRenderContext(t *testing.T) {
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: &CurrentUsage{
				InputTokens: 90000,
			},
		},
	}
	out := RenderContext(input)
	assert.Contains(t, out, "🧠")
	assert.Contains(t, out, "45%")
	assert.Contains(t, out, "90K/200K")
}

func TestRenderContext_NoData(t *testing.T) {
	input := &StatusJSON{}
	out := RenderContext(input)
	assert.Equal(t, "", out)
}

func TestRenderContext_Warning(t *testing.T) {
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: &CurrentUsage{
				InputTokens: 120000, // 60%
			},
		},
	}
	out := RenderContext(input)
	assert.Contains(t, out, "⚠️")
}

func TestRenderContext_Danger(t *testing.T) {
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: &CurrentUsage{
				InputTokens: 160000, // 80%
			},
		},
	}
	out := RenderContext(input)
	assert.Contains(t, out, "🔴")
}
