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

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		pct  int
		want string
	}{
		{0, "░░░░░░░░░░"},
		{10, "█░░░░░░░░░"},
		{50, "█████░░░░░"},
		{100, "██████████"},
		{105, "██████████"}, // clamp at 10
	}
	for _, tt := range tests {
		got := RenderProgressBar(tt.pct)
		assert.Equal(t, tt.want, got, "pct=%d", tt.pct)
	}
}

func TestRenderContext_RemainingPercentage(t *testing.T) {
	remaining := 55.0 // 55% remaining = 45% used
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			RemainingPercentage: &remaining,
		},
	}
	out := RenderContext(input)
	assert.Contains(t, out, "45%")
	assert.Contains(t, out, "█") // progress bar
}

func TestRenderContext_ManualCalculation(t *testing.T) {
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: &CurrentUsage{
				InputTokens: 90000, // 45%
			},
		},
	}
	out := RenderContext(input)
	assert.Contains(t, out, "45%")
	assert.Contains(t, out, "█") // progress bar
}

func TestRenderContext_NoData(t *testing.T) {
	input := &StatusJSON{}
	out := RenderContext(input)
	assert.Equal(t, "", out)
}

func TestRenderContext_Warning(t *testing.T) {
	remaining := 40.0 // 40% remaining = 60% used
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			RemainingPercentage: &remaining,
		},
	}
	out := RenderContext(input)
	assert.Contains(t, out, "60%")
	assert.Contains(t, out, "█") // progress bar
}

func TestRenderContext_Danger(t *testing.T) {
	remaining := 20.0 // 20% remaining = 80% used
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			RemainingPercentage: &remaining,
		},
	}
	out := RenderContext(input)
	assert.Contains(t, out, "💀")
	assert.Contains(t, out, "80%")
}
