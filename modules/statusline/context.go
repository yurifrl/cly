package statusline

import (
	"fmt"

	"github.com/yurifrl/cly/pkg/style"
)

// CalculateTokens sums all token fields.
func CalculateTokens(usage *CurrentUsage) int {
	if usage == nil {
		return 0
	}
	return usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
}

// CalculatePercentage returns tokens as percentage of max.
func CalculatePercentage(tokens, max int) int {
	if max == 0 {
		return 0
	}
	return tokens * 100 / max
}

// FormatTokens formats token count as "90K".
func FormatTokens(tokens int) string {
	return fmt.Sprintf("%dK", tokens/1000)
}

// RenderContext outputs context window percentage.
func RenderContext(input *StatusJSON) string {
	if input.ContextWindow == nil || input.ContextWindow.CurrentUsage == nil {
		return ""
	}

	tokens := CalculateTokens(input.ContextWindow.CurrentUsage)
	max := input.ContextWindow.ContextWindowSize
	if max == 0 {
		max = MaxContextTokens
	}
	pct := CalculatePercentage(tokens, max)
	tokensStr := FormatTokens(tokens)
	maxStr := FormatTokens(max)

	base := fmt.Sprintf("🧠 %d%% (%s/%s)", pct, tokensStr, maxStr)

	switch {
	case pct >= 75:
		return style.RedStyle.Render(base) + " 🔴"
	case pct >= 50:
		return style.YellowStyle.Render(base) + " ⚠️"
	default:
		return style.GreenStyle.Render(base)
	}
}
