package claudesession

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func newTestPicker(sessions Sessions) pickerModel {
	entries := sortedEntries(sessions, SortDateDesc)
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		items = append(items, pickerItem{entry: e})
	}
	l := list.New(items, simpleDelegate{}, 80, 14)
	return pickerModel{list: l, sessions: sessions, order: SortDateDesc}
}

func TestPickerYoloToggle(t *testing.T) {
	sessions := Sessions{
		"proj": Entry{ID: "id-1", Name: "proj", Path: "/tmp"},
	}
	m := newTestPicker(sessions)
	assert.False(t, m.yolo)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(pickerModel)
	assert.True(t, m.yolo)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = updated.(pickerModel)
	assert.False(t, m.yolo)
}

func TestPickerYoloDefault(t *testing.T) {
	sessions := Sessions{
		"proj": Entry{ID: "id-1", Name: "proj", Path: "/tmp"},
	}
	m := newTestPicker(sessions)
	assert.False(t, m.yolo)
}
