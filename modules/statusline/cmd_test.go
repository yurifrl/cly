package statusline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderModel(t *testing.T) {
	input := &StatusJSON{
		Model: &ModelInfo{DisplayName: "Opus"},
	}
	out := RenderModel(input)
	assert.Equal(t, "[Opus]", out)
}

func TestRenderModel_NoData(t *testing.T) {
	input := &StatusJSON{}
	out := RenderModel(input)
	assert.Equal(t, "", out)
}

func TestRenderCost(t *testing.T) {
	input := &StatusJSON{
		Cost: &CostInfo{TotalCostUSD: 0.02},
	}
	out := RenderCost(input)
	assert.Equal(t, "💰 $0.02", out)
}

func TestRenderCost_NoData(t *testing.T) {
	input := &StatusJSON{}
	out := RenderCost(input)
	assert.Equal(t, "", out)
}

func TestParseFormat(t *testing.T) {
	format := "$context │ $model │ $custom"
	parts := ParseFormat(format)
	assert.Equal(t, []string{"context", "model", "custom"}, parts)
}

func TestParseFormat_Simple(t *testing.T) {
	format := "$context$model"
	parts := ParseFormat(format)
	assert.Equal(t, []string{"context", "model"}, parts)
}

func TestRenderStatusline(t *testing.T) {
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage:      &CurrentUsage{InputTokens: 90000},
		},
		Model: &ModelInfo{DisplayName: "Opus"},
	}
	cfg := Config{
		Format:  "$context │ $model",
		Context: ContextConfig{Enabled: true},
		Model:   ModelConfig{Enabled: true},
	}
	out := RenderStatusline(input, cfg)
	assert.Contains(t, out, "█") // progress bar
	assert.Contains(t, out, "[Opus]")
	assert.Contains(t, out, "│")
}

func TestRenderStatusline_DisabledSkipped(t *testing.T) {
	input := &StatusJSON{
		Model: &ModelInfo{DisplayName: "Opus"},
	}
	cfg := Config{
		Format: "$context │ $model",
		Model:  ModelConfig{Enabled: true},
		// context disabled
	}
	out := RenderStatusline(input, cfg)
	assert.NotContains(t, out, "█") // no progress bar
	assert.Contains(t, out, "[Opus]")
	// Should not have leading separator
	assert.NotEqual(t, "│", string(out[0]))
}

func TestRenderStatusline_AllDisabled(t *testing.T) {
	input := &StatusJSON{}
	cfg := DefaultConfig()
	out := RenderStatusline(input, cfg)
	assert.Equal(t, "", out)
}
