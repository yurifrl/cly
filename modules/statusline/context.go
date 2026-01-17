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

// RenderProgressBar creates a 10-segment progress bar.
func RenderProgressBar(pct int) string {
	filled := pct / 10
	if filled > 10 {
		filled = 10
	}
	empty := 10 - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	return bar
}

// RenderContext outputs context window percentage with progress bar.
func RenderContext(input *StatusJSON) string {
	if input.ContextWindow == nil {
		return ""
	}

	var pct int

	// Prefer remaining_percentage from Claude Code (more accurate)
	if input.ContextWindow.RemainingPercentage != nil {
		remaining := int(*input.ContextWindow.RemainingPercentage)
		pct = 100 - remaining // Show USED percentage
	} else if input.ContextWindow.CurrentUsage != nil {
		// Fallback to manual calculation
		tokens := CalculateTokens(input.ContextWindow.CurrentUsage)
		max := input.ContextWindow.ContextWindowSize
		if max == 0 {
			max = MaxContextTokens
		}
		pct = CalculatePercentage(tokens, max)
	} else {
		return ""
	}

	bar := RenderProgressBar(pct)
	base := fmt.Sprintf("%s %d%%", bar, pct)

	switch {
	case pct >= 80:
		return style.RedStyle.Render(base) + " 💀"
	case pct >= 65:
		return "\x1b[38;5;208m" + base + "\x1b[0m" // orange
	case pct >= 50:
		return style.YellowStyle.Render(base)
	default:
		return style.GreenStyle.Render(base)
	}
}
