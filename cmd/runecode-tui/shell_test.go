package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestShellQuickJumpSetsRouteAndFocus(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.currentID != routeChat {
		t.Fatalf("expected route %q, got %q", routeChat, shell.currentID)
	}
	if shell.focus != focusContent {
		t.Fatalf("expected focusContent, got %v", shell.focus)
	}
}

func TestShellMouseClickNavOpensRoute(t *testing.T) {
	m := newShellModel()
	m.width = 120
	_, boxes := m.nav.Render(true)
	if len(boxes) < 2 {
		t.Fatalf("expected nav boxes, got %d", len(boxes))
	}
	chatBox := boxes[1]
	startY, _ := m.navYRange()
	clickX := len(navLinePrefix) + chatBox.StartX
	updated, _ := m.Update(tea.MouseMsg{X: clickX, Y: startY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.currentID != routeChat {
		t.Fatalf("expected route %q, got %q", routeChat, shell.currentID)
	}
	if shell.focus != focusContent {
		t.Fatalf("expected focusContent, got %v", shell.focus)
	}
}

func TestShellMouseReleaseNavOpensRoute(t *testing.T) {
	m := newShellModel()
	m.width = 120
	_, boxes := m.nav.Render(true)
	if len(boxes) < 2 {
		t.Fatalf("expected nav boxes, got %d", len(boxes))
	}
	chatBox := boxes[1]
	startY, _ := m.navYRange()
	clickX := len(navLinePrefix) + chatBox.StartX
	updated, _ := m.Update(tea.MouseMsg{X: clickX, Y: startY, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.currentID != routeChat {
		t.Fatalf("expected route %q, got %q", routeChat, shell.currentID)
	}
}

func TestShellMouseClickPaletteRowPicksRoute(t *testing.T) {
	m := newShellModel()
	m.width = 120
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if !shell.palette.IsOpen() {
		t.Fatal("expected palette open")
	}
	paletteStartY := shell.paletteStartY()
	firstMatchY := paletteStartY + 3

	updated, cmd := shell.Update(tea.MouseMsg{X: 3, Y: firstMatchY + 1, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell, ok = updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if cmd == nil {
		t.Fatal("expected route-switch command from palette click")
	}
	updated, cmd = shell.Update(cmd())
	shell, ok = updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if cmd == nil {
		t.Fatal("expected route activation command")
	}
	updated, _ = shell.Update(cmd())
	shell, ok = updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.currentID != routeChat {
		t.Fatalf("expected route %q from palette click, got %q", routeChat, shell.currentID)
	}
	if shell.palette.IsOpen() {
		t.Fatal("expected palette to close after mouse pick")
	}
}

func TestShellMouseNavWorksWhenHeaderWraps(t *testing.T) {
	m := newShellModel()
	m.width = 24
	_, boxes := m.nav.Render(false)
	if len(boxes) < 2 {
		t.Fatalf("expected nav boxes, got %d", len(boxes))
	}
	startY, _ := m.navYRange()
	chatBox := boxes[1]
	clickX := len(navLinePrefix) + chatBox.StartX
	updated, _ := m.Update(tea.MouseMsg{X: clickX, Y: startY, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.currentID != routeChat {
		t.Fatalf("expected route %q, got %q", routeChat, shell.currentID)
	}
}

func TestShellKeyboardAndWheelScrollEquivalent(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.scroll != 1 {
		t.Fatalf("expected scroll 1 after j, got %d", shell.scroll)
	}
	updated, _ = shell.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	shell, ok = updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.scroll != 2 {
		t.Fatalf("expected scroll 2 after wheel down, got %d", shell.scroll)
	}
}

func TestHelpRenderedFromRealKeyBindings(t *testing.T) {
	help := renderHelp(defaultShellKeyMap(), false)
	if !strings.Contains(help, "q/ctrl+c Quit") {
		t.Fatalf("expected quit binding in help, got %q", help)
	}
	if !strings.Contains(help, "tab Next focus area") {
		t.Fatalf("expected focus binding in help, got %q", help)
	}
}

func TestShellUpdatesWindowSizeWhilePaletteOpen(t *testing.T) {
	m := newShellModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if !shell.palette.IsOpen() {
		t.Fatal("expected palette to be open")
	}

	updated, _ = shell.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	shell, ok = updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.width != 120 || shell.height != 40 {
		t.Fatalf("expected window size 120x40, got %dx%d", shell.width, shell.height)
	}
	if !shell.palette.IsOpen() {
		t.Fatal("expected palette to remain open after resize")
	}
}

func TestShellShiftTabCyclesFocusBackward(t *testing.T) {
	m := newShellModel()
	m.focus = focusNav

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	shell, ok := updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.focus != focusContent {
		t.Fatalf("expected reverse focus from nav to content, got %v", shell.focus)
	}

	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	shell, ok = updated.(shellModel)
	if !ok {
		t.Fatalf("expected shellModel, got %T", updated)
	}
	if shell.focus != focusNav {
		t.Fatalf("expected reverse focus from content to nav, got %v", shell.focus)
	}
}
