package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type keyBinding struct {
	Keys        []string
	Description string
}

func (k keyBinding) matches(msg tea.KeyMsg) bool {
	pressed := msg.String()
	for _, key := range k.Keys {
		if pressed == key {
			return true
		}
	}
	return false
}

func (k keyBinding) label() string {
	return strings.Join(k.Keys, "/")
}

type shellKeyMap struct {
	Quit           keyBinding
	CycleFocusNext keyBinding
	CycleFocusPrev keyBinding
	RouteNext      keyBinding
	RoutePrev      keyBinding
	RouteOpen      keyBinding
	QuickJump      keyBinding
	OpenPalette    keyBinding
	PaletteClose   keyBinding
	PalettePick    keyBinding
	PaletteNext    keyBinding
	PalettePrev    keyBinding
	ScrollDown     keyBinding
	ScrollUp       keyBinding
}

func defaultShellKeyMap() shellKeyMap {
	return shellKeyMap{
		Quit:           keyBinding{Keys: []string{"q", "ctrl+c"}, Description: "Quit"},
		CycleFocusNext: keyBinding{Keys: []string{"tab"}, Description: "Next focus area"},
		CycleFocusPrev: keyBinding{Keys: []string{"shift+tab"}, Description: "Previous focus area"},
		RouteNext:      keyBinding{Keys: []string{"l", "right"}, Description: "Move nav selection right"},
		RoutePrev:      keyBinding{Keys: []string{"h", "left"}, Description: "Move nav selection left"},
		RouteOpen:      keyBinding{Keys: []string{"enter"}, Description: "Open selected route"},
		QuickJump:      keyBinding{Keys: []string{"1-7"}, Description: "Open route by number"},
		OpenPalette:    keyBinding{Keys: []string{":", "ctrl+p"}, Description: "Open quick jump palette"},
		PaletteClose:   keyBinding{Keys: []string{"esc"}, Description: "Close palette"},
		PalettePick:    keyBinding{Keys: []string{"enter"}, Description: "Open selected match"},
		PaletteNext:    keyBinding{Keys: []string{"down", "ctrl+n"}, Description: "Next palette match"},
		PalettePrev:    keyBinding{Keys: []string{"up", "ctrl+p"}, Description: "Previous palette match"},
		ScrollDown:     keyBinding{Keys: []string{"j", "down", "pgdown"}, Description: "Scroll content down"},
		ScrollUp:       keyBinding{Keys: []string{"k", "up", "pgup"}, Description: "Scroll content up"},
	}
}

func (k shellKeyMap) helpBindings(paletteOpen bool) []keyBinding {
	base := []keyBinding{
		k.CycleFocusNext,
		k.CycleFocusPrev,
		k.OpenPalette,
		k.QuickJump,
		k.ScrollUp,
		k.ScrollDown,
		k.Quit,
	}
	if !paletteOpen {
		base = append(base, k.RoutePrev, k.RouteNext, k.RouteOpen)
		return base
	}
	return append(base, k.PalettePrev, k.PaletteNext, k.PalettePick, k.PaletteClose)
}
