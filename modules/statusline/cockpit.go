package statusline

import (
	"fmt"
	"strings"
)

// Cockpit layout, ported from statusline.sh/community/cockpit-4shq (top line
// only; the rate-limit "usage" line is intentionally omitted).

const (
	cockpitBarWidth = 30
	cockpitFilled   = "￭"
	cockpitEmpty    = "･"
	ansiReset       = "\x1b[0m"
)

func sgr(code, text string) string {
	if text == "" {
		return ""
	}
	return "\x1b[" + code + "m" + text + ansiReset
}

// cockpitBar renders a fixed-width bar for a 0-100 percentage.
func cockpitBar(pct float64, width int) string {
	p := int(pct)
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	n := (p*width + 50) / 100
	return strings.Repeat(cockpitFilled, n) + strings.Repeat(cockpitEmpty, width-n)
}

// compactTokens formats 94400 as "94.4k", 200000 as "200k", 1.5M as "1.5M".
func compactTokens(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1000000:
		whole, dec := n/1000, (n%1000)/100
		if dec == 0 {
			return fmt.Sprintf("%dk", whole)
		}
		return fmt.Sprintf("%d.%dk", whole, dec)
	default:
		whole, dec := n/1000000, (n%1000000)/100000
		if dec == 0 {
			return fmt.Sprintf("%dM", whole)
		}
		return fmt.Sprintf("%d.%dM", whole, dec)
	}
}

// RenderCockpit outputs: " model  | effort   tok used/total (pct) bar   $cost".
func RenderCockpit(input *StatusJSON) string {
	const badge = "1;38;2;255;255;255;48;2;43;47;58"
	const blue = "1;38;2;143;184;218"
	const dim = "2;38;2;143;184;218"

	var b strings.Builder

	model := ""
	if input.Model != nil {
		model = input.Model.DisplayName
	}
	b.WriteString(sgr(badge, " "+model+" "))
	b.WriteString(" | ")
	if input.Thinking != nil && input.Thinking.Enabled && input.Effort != nil {
		b.WriteString(input.Effort.Level)
	}
	b.WriteString("  ")

	used, total, pct := 0, 0, 0.0
	if cw := input.ContextWindow; cw != nil {
		used, total = cw.TotalInputTokens, cw.ContextWindowSize
		if cw.UsedPercentage != nil {
			pct = *cw.UsedPercentage
		}
	}
	b.WriteString(sgr(dim, "tok "))
	b.WriteString(sgr(blue, fmt.Sprintf("%s/%s (%d%%) ", compactTokens(used), compactTokens(total), int(pct))))
	b.WriteString(cockpitBar(pct, cockpitBarWidth))
	b.WriteString("  ")

	cost := 0.0
	if input.Cost != nil {
		cost = input.Cost.TotalCostUSD
	}
	b.WriteString(fmt.Sprintf("$%.2f", cost))
	return b.String()
}
