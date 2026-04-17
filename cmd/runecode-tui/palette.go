package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

type paletteModel struct {
	open          bool
	query         string
	selectedIndex int
	matches       []paletteEntry
	entries       []paletteEntry
}

func newPaletteModel(entries []paletteEntry) paletteModel {
	m := paletteModel{entries: entries}
	m.rebuildMatches()
	return m
}

func (m paletteModel) UpdateEntries(entries []paletteEntry) paletteModel {
	m.entries = append([]paletteEntry(nil), entries...)
	m.rebuildMatches()
	return m
}

func (m paletteModel) Open() paletteModel {
	m.open = true
	m.query = ""
	m.selectedIndex = 0
	m.rebuildMatches()
	return m
}

func (m paletteModel) Close() paletteModel {
	m.open = false
	return m
}

func (m paletteModel) IsOpen() bool {
	return m.open
}

func (m paletteModel) SelectedEntry() (paletteEntry, bool) {
	if len(m.matches) == 0 {
		return paletteEntry{}, false
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	if m.selectedIndex >= len(m.matches) {
		m.selectedIndex = len(m.matches) - 1
	}
	return m.matches[m.selectedIndex], true
}

func (m paletteModel) Update(msg tea.Msg, keys shellKeyMap) (paletteModel, paletteActionMsg, bool) {
	if !m.open {
		return m, paletteActionMsg{}, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, paletteActionMsg{}, false
	}
	return m.updateKey(key, keys)
}

func (m paletteModel) UpdateMouse(msg tea.MouseMsg, paletteStartY int, layoutWidth int) (paletteModel, paletteActionMsg, bool) {
	if !m.open {
		return m, paletteActionMsg{}, false
	}
	if msg.Button != tea.MouseButtonLeft {
		return m, paletteActionMsg{}, false
	}
	if msg.Action != tea.MouseActionRelease {
		return m, paletteActionMsg{}, false
	}
	index, ok := m.matchIndexAtPosition(msg.X, msg.Y, paletteStartY, layoutWidth)
	if !ok {
		return m, paletteActionMsg{}, false
	}
	m.selectedIndex = index
	return m.pickRoute()
}

func (m paletteModel) updateKey(key tea.KeyMsg, keys shellKeyMap) (paletteModel, paletteActionMsg, bool) {
	switch {
	case keys.PaletteClose.matches(key):
		return m.Close(), paletteActionMsg{}, false
	case keys.PalettePick.matches(key):
		return m.pickRoute()
	case keys.PaletteNext.matches(key):
		return m.stepSelection(1), paletteActionMsg{}, false
	case keys.PalettePrev.matches(key):
		return m.stepSelection(-1), paletteActionMsg{}, false
	case key.Type == tea.KeyBackspace || key.Type == tea.KeyDelete:
		return m.deleteQueryRune(), paletteActionMsg{}, false
	case isTypingKey(key):
		m.query += key.String()
		m.rebuildMatches()
		return m, paletteActionMsg{}, false
	default:
		return m, paletteActionMsg{}, false
	}
}

func (m paletteModel) pickRoute() (paletteModel, paletteActionMsg, bool) {
	r, ok := m.SelectedEntry()
	if !ok {
		return m.Close(), paletteActionMsg{}, false
	}
	return m.Close(), r.Action, true
}

func (m paletteModel) stepSelection(delta int) paletteModel {
	if len(m.matches) == 0 {
		return m
	}
	if delta > 0 {
		m.selectedIndex = (m.selectedIndex + 1) % len(m.matches)
		return m
	}
	m.selectedIndex--
	if m.selectedIndex < 0 {
		m.selectedIndex = len(m.matches) - 1
	}
	return m
}

func (m paletteModel) deleteQueryRune() paletteModel {
	if len(m.query) == 0 {
		return m
	}
	_, size := utf8.DecodeLastRuneInString(m.query)
	if size <= 0 {
		return m
	}
	m.query = m.query[:len(m.query)-size]
	m.rebuildMatches()
	return m
}

func (m *paletteModel) rebuildMatches() {
	needle := strings.ToLower(strings.TrimSpace(m.query))
	m.matches = m.matches[:0:0]
	if needle == "" {
		m.matches = append(m.matches, m.entries...)
	} else {
		for _, r := range m.entries {
			if strings.Contains(strings.ToLower(r.Label), needle) || strings.Contains(strings.ToLower(r.Description), needle) || strings.Contains(strings.ToLower(r.Search), needle) {
				m.matches = append(m.matches, r)
			}
		}
	}
	if m.selectedIndex >= len(m.matches) {
		if len(m.matches) == 0 {
			m.selectedIndex = 0
		} else {
			m.selectedIndex = len(m.matches) - 1
		}
	}
}

func (m paletteModel) matchIndexAtPosition(x int, y int, paletteStartY int, layoutWidth int) (int, bool) {
	if len(m.matches) == 0 {
		return 0, false
	}
	startX, endX := centeredOverlayContentBounds(layoutWidth)
	if x < startX || x > endX {
		return 0, false
	}
	startY := paletteStartY + 5
	for _, candidateY := range []int{y, y - 1} {
		index := candidateY - startY
		if index >= 0 && index < len(m.matches) {
			return index, true
		}
	}
	return 0, false
}

func paletteOverlayBounds(layoutWidth int) (int, int) {
	return centeredOverlayContentBounds(layoutWidth)
}

func paletteMatchLine(entry paletteEntry, selected bool) string {
	marker := "•"
	if selected {
		marker = "▶"
	}
	line := " " + marker + " " + fmt.Sprintf("%d. %s — %s", entry.Index, entry.Label, entry.Description)
	return line
}

func isTypingKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		return r >= 32 && r != 127
	}
	return false
}
