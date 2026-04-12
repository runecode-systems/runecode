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
	clickX := chatBox.StartX
	updated, _ := m.Update(tea.MouseMsg{X: clickX, Y: navLineY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
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
