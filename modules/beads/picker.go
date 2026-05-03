package beads

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// quickSelectTTL is how long a typed prefix stays "live" before the next
// key press starts a fresh search.
const quickSelectTTL = 800 * time.Millisecond

// picker is a small horizontal pill selector with:
//   - arrow-key / ctrl+n/p navigation (handled by caller)
//   - letter quick-select with 800ms auto-expiring buffer
//   - preservation of selection across option-list refreshes (by name)
type picker struct {
	tag     string // used to scope quickSelectExpireMsg across multiple pickers
	options []string
	idx     int
	qsBuf   string
	qsGen   int
}

// quickSelectExpireMsg carries both the tag and generation counter so a
// tick intended for picker A never clears picker B's buffer.
type quickSelectExpireMsg struct {
	tag string
	gen int
}

func (p *picker) next() {
	if n := len(p.options); n > 0 {
		p.idx = (p.idx + 1) % n
	}
	p.qsBuf = ""
}

func (p *picker) prev() {
	if n := len(p.options); n > 0 {
		p.idx = (p.idx - 1 + n) % n
	}
	p.qsBuf = ""
}

// quickSelect appends a character to the buffer and moves the index to the
// first prefix match (case-insensitive). Returns a tea.Cmd that expires the
// buffer after quickSelectTTL unless another keystroke bumps the generation.
func (p *picker) quickSelect(ch string) tea.Cmd {
	p.qsBuf += strings.ToLower(ch)
	if i, ok := firstPrefixMatch(p.options, p.qsBuf); ok {
		p.idx = i
	}
	p.qsGen++
	tag := p.tag
	gen := p.qsGen
	return tea.Tick(quickSelectTTL, func(time.Time) tea.Msg {
		return quickSelectExpireMsg{tag: tag, gen: gen}
	})
}

// backspaceQS pops the last char from the quick-select buffer.
func (p *picker) backspaceQS() {
	if p.qsBuf == "" {
		return
	}
	p.qsBuf = p.qsBuf[:len(p.qsBuf)-1]
	if p.qsBuf != "" {
		if i, ok := firstPrefixMatch(p.options, p.qsBuf); ok {
			p.idx = i
		}
	}
}

// onExpire handles a quickSelectExpireMsg; returns true if it consumed it.
func (p *picker) onExpire(msg quickSelectExpireMsg) bool {
	if msg.tag != p.tag {
		return false
	}
	if msg.gen == p.qsGen {
		p.qsBuf = ""
	}
	return true
}

// setOptions swaps the option list while preserving the current selection
// by name where possible.
func (p *picker) setOptions(opts []string) {
	current := p.selected()
	p.options = opts
	p.idx = 0
	for i, o := range opts {
		if o == current {
			p.idx = i
			break
		}
	}
}

func (p *picker) selected() string {
	if p.idx < 0 || p.idx >= len(p.options) {
		return ""
	}
	return p.options[p.idx]
}

func (p *picker) setSelected(name string) {
	for i, o := range p.options {
		if o == name {
			p.idx = i
			return
		}
	}
}

// view renders the picker as horizontal pills. active=true uses the bold
// active pill style; otherwise the current selection is still highlighted
// but with a muted style so you can see the choice when not focused.
func (p *picker) view(active bool) string {
	parts := make([]string, 0, len(p.options))
	for i, o := range p.options {
		switch {
		case i == p.idx && active:
			parts = append(parts, pillActiveStyle.Render(o))
		case i == p.idx:
			parts = append(parts, pillSelectedDimStyle.Render(o))
		default:
			parts = append(parts, pillStyle.Render(o))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// isQuickSelectKey reports whether s is a single a-z/A-Z/0-9 character.
func isQuickSelectKey(s string) bool {
	if len(s) != 1 {
		return false
	}
	c := s[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// firstPrefixMatch returns the index of the first option whose name starts
// with prefix (case-insensitive). ok=false when nothing matches.
func firstPrefixMatch(options []string, prefix string) (int, bool) {
	p := strings.ToLower(prefix)
	for i, o := range options {
		if strings.HasPrefix(strings.ToLower(o), p) {
			return i, true
		}
	}
	return 0, false
}
