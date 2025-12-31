package statusline

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegration_FullFlow(t *testing.T) {
	// Sample StatusJSON from Claude Code
	input := &StatusJSON{
		TranscriptPath: "/tmp/transcript.jsonl",
		Model:          &ModelInfo{ID: "claude-opus-4-1", DisplayName: "Opus"},
		Workspace:      &WorkspaceInfo{CurrentDir: "/home/user/project"},
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage: &CurrentUsage{
				InputTokens:              80000,
				CacheReadInputTokens:     5000,
				CacheCreationInputTokens: 5000,
			},
		},
		Cost: &CostInfo{TotalCostUSD: 0.05},
	}

	// Config with all enabled
	cfg := Config{
		Format:  "$context │ $model │ $cost",
		Context: ContextConfig{Enabled: true},
		Model:   ModelConfig{Enabled: true},
		Cost:    CostConfig{Enabled: true},
	}

	out := RenderStatusline(input, cfg)

	// Should contain all parts
	assert.Contains(t, out, "🧠")
	assert.Contains(t, out, "45%") // 90000/200000
	assert.Contains(t, out, "[Opus]")
	assert.Contains(t, out, "💰 $0.05")
	assert.Contains(t, out, "│")
}

func TestIntegration_ContextOnly(t *testing.T) {
	input := &StatusJSON{
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage:      &CurrentUsage{InputTokens: 50000},
		},
	}

	cfg := Config{
		Format:  "$context",
		Context: ContextConfig{Enabled: true},
	}

	out := RenderStatusline(input, cfg)
	assert.Contains(t, out, "🧠")
	assert.Contains(t, out, "25%")
	assert.NotContains(t, out, "│") // No separator for single item
}

func TestIntegration_BasisAllDisabled(t *testing.T) {
	input := &StatusJSON{
		Model: &ModelInfo{DisplayName: "Opus"},
		ContextWindow: &ContextWindow{
			ContextWindowSize: 200000,
			CurrentUsage:      &CurrentUsage{InputTokens: 50000},
		},
	}

	// BASIS: all disabled
	cfg := DefaultConfig()

	out := RenderStatusline(input, cfg)
	assert.Equal(t, "", out)
}

func TestIntegration_CustomCommand(t *testing.T) {
	input := &StatusJSON{
		Workspace: &WorkspaceInfo{CurrentDir: "/tmp"},
	}

	cfg := Config{
		Format: "$custom",
		Custom: CustomConfig{
			Enabled: true,
			Command: "echo hello",
			Timeout: 1000,
		},
	}

	out := RenderStatusline(input, cfg)
	assert.Equal(t, "hello", out)
}

func TestIntegration_CustomCommandTimeout(t *testing.T) {
	input := &StatusJSON{}

	cfg := Config{
		Format: "$custom",
		Custom: CustomConfig{
			Enabled: true,
			Command: "sleep 5",
			Timeout: 50, // 50ms timeout
		},
	}

	out := RenderStatusline(input, cfg)
	assert.Equal(t, "", out) // Should timeout and return empty
}
