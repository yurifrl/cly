package agentsession

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func newTestPicker(sessions Sessions, allowYolo bool) pickerModel {
	entries := sortedEntries(sessions, SortDateDesc)
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		items = append(items, pickerItem{entry: e})
	}
	l := list.New(items, simpleDelegate{}, 80, 14)
	return pickerModel{list: l, sessions: sessions, order: SortDateDesc, allowYolo: allowYolo}
}

func TestPickerYoloToggleWhenAllowed(t *testing.T) {
	sessions := Sessions{
		"claude:proj": Entry{ID: "id-1", Name: "proj", Provider: "claude", Path: "/tmp"},
	}
	m := newTestPicker(sessions, true)
	assert.False(t, m.yolo)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m = updated.(pickerModel)
	assert.True(t, m.yolo)

	updated, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m = updated.(pickerModel)
	assert.False(t, m.yolo)
}

func TestPickerYoloIgnoredWhenNotAllowed(t *testing.T) {
	sessions := Sessions{
		"pi:proj": Entry{ID: "id-1", Name: "proj", Provider: "pi", Path: "/tmp"},
	}
	m := newTestPicker(sessions, false)
	assert.False(t, m.yolo)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m = updated.(pickerModel)
	assert.False(t, m.yolo)
}

func TestPickerHeadlineIncludesProvider(t *testing.T) {
	item := pickerItem{entry: Entry{ID: "abc-123", Name: "proj", Provider: "pi"}}
	headline := item.headline()

	assert.True(t, strings.Contains(headline, "proj"))
	assert.True(t, strings.Contains(headline, "pi"))
	assert.True(t, strings.Contains(headline, "abc-123"))
}

func TestProviderTagDefaultsToClaude(t *testing.T) {
	tag := providerTag("")
	assert.True(t, strings.Contains(tag, "claude"))
}
