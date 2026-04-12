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
	matches       []routeDefinition
	routes        []routeDefinition
}

func newPaletteModel(routes []routeDefinition) paletteModel {
	m := paletteModel{routes: routes}
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

func (m paletteModel) SelectedRoute() (routeDefinition, bool) {
	if len(m.matches) == 0 {
		return routeDefinition{}, false
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
	if m.selectedIndex >= len(m.matches) {
		m.selectedIndex = len(m.matches) - 1
	}
	return m.matches[m.selectedIndex], true
}

func (m paletteModel) Update(msg tea.Msg, keys shellKeyMap) (paletteModel, routeSwitchMsg, bool) {
	if !m.open {
		return m, routeSwitchMsg{}, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, routeSwitchMsg{}, false
	}
	return m.updateKey(key, keys)
}

func (m paletteModel) UpdateMouse(msg tea.MouseMsg, paletteStartY int, layoutWidth int) (paletteModel, routeSwitchMsg, bool) {
	if !m.open {
		return m, routeSwitchMsg{}, false
	}
	_ = layoutWidth
	if msg.Button != tea.MouseButtonLeft {
		return m, routeSwitchMsg{}, false
	}
	if msg.Action != tea.MouseActionPress && msg.Action != tea.MouseActionRelease {
		return m, routeSwitchMsg{}, false
	}
	index, ok := m.matchIndexAtPosition(msg.Y, paletteStartY, layoutWidth)
	if !ok {
		return m, routeSwitchMsg{}, false
	}
	m.selectedIndex = index
	return m.pickRoute()
}

func (m paletteModel) updateKey(key tea.KeyMsg, keys shellKeyMap) (paletteModel, routeSwitchMsg, bool) {
	switch {
	case keys.PaletteClose.matches(key):
		return m.Close(), routeSwitchMsg{}, false
	case keys.PalettePick.matches(key):
		return m.pickRoute()
	case keys.PaletteNext.matches(key):
		return m.stepSelection(1), routeSwitchMsg{}, false
	case keys.PalettePrev.matches(key):
		return m.stepSelection(-1), routeSwitchMsg{}, false
	case key.Type == tea.KeyBackspace || key.Type == tea.KeyDelete:
		return m.deleteQueryRune(), routeSwitchMsg{}, false
	case isTypingKey(key):
		m.query += key.String()
		m.rebuildMatches()
		return m, routeSwitchMsg{}, false
	default:
		return m, routeSwitchMsg{}, false
	}
}

func (m paletteModel) pickRoute() (paletteModel, routeSwitchMsg, bool) {
	r, ok := m.SelectedRoute()
	if !ok {
		return m.Close(), routeSwitchMsg{}, false
	}
	return m.Close(), routeSwitchMsg{RouteID: r.ID}, true
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
	m.matches = m.matches[:0]
	if needle == "" {
		m.matches = append(m.matches, m.routes...)
	} else {
		for _, r := range m.routes {
			if strings.Contains(strings.ToLower(r.Label), needle) || strings.Contains(strings.ToLower(r.Description), needle) {
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

func (m paletteModel) matchIndexAtPosition(y int, paletteStartY int, layoutWidth int) (int, bool) {
	if len(m.matches) == 0 {
		return 0, false
	}
	_ = layoutWidth
	startY := paletteStartY + 3
	for _, candidateY := range []int{y, y - 1} {
		index := candidateY - startY
		if index >= 0 && index < len(m.matches) {
			return index, true
		}
	}
	return 0, false
}

func paletteMatchLine(route routeDefinition, selected bool) string {
	marker := " "
	if selected {
		marker = ">"
	}
	return " " + marker + " " + fmt.Sprintf("%d. %s — %s", route.Index, route.Label, route.Description)
}

func isTypingKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		return r >= 32 && r != 127
	}
	return false
}
