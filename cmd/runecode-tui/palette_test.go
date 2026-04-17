package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPaletteDeleteQueryRunePreservesUTF8(t *testing.T) {
	m := newPaletteModel([]paletteEntry{{Index: 1, Label: "jump route chat", Description: "open chat route", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeChat}}}})
	m.open = true
	m.query = "Goλ"

	m = m.deleteQueryRune()
	if m.query != "Go" {
		t.Fatalf("expected UTF-8-safe delete to keep valid string, got %q", m.query)
	}

	m = m.deleteQueryRune()
	if m.query != "G" {
		t.Fatalf("expected second delete to remove one rune, got %q", m.query)
	}
}

func TestPalettePickReturnsActionMessage(t *testing.T) {
	m := newPaletteModel([]paletteEntry{{Index: 1, Label: "back", Description: "go back", Action: paletteActionMsg{Verb: verbBack}}}).Open()
	updated, action, changed := m.Update(keyMsg("enter"), defaultShellKeyMap())
	if !changed {
		t.Fatal("expected palette pick to emit action")
	}
	if action.Verb != verbBack {
		t.Fatalf("expected back verb, got %q", action.Verb)
	}
	if updated.IsOpen() {
		t.Fatal("expected palette to close after pick")
	}
}

func TestPaletteMatchLineUsesSelectedStyling(t *testing.T) {
	line := paletteMatchLine(paletteEntry{Index: 1, Label: "open route chat", Description: "go to chat"}, true)
	if !strings.Contains(line, "▶") {
		t.Fatalf("expected selected marker in palette match line, got %q", line)
	}
}

func TestPaletteMousePickTriggersOnlyOnRelease(t *testing.T) {
	m := newPaletteModel([]paletteEntry{{Index: 1, Label: "back", Description: "go back", Action: paletteActionMsg{Verb: verbBack}}}).Open()
	updated, action, changed := m.UpdateMouse(tea.MouseMsg{X: 30, Y: 9, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}, 3, 80)
	if changed {
		t.Fatalf("expected press to only select, not emit action: %+v", action)
	}
	updated, action, changed = updated.UpdateMouse(tea.MouseMsg{X: 30, Y: 9, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}, 3, 80)
	if !changed {
		t.Fatal("expected release to emit palette action")
	}
	if action.Verb != verbBack {
		t.Fatalf("expected back verb, got %q", action.Verb)
	}
}

func TestPaletteMouseIgnoresClicksOutsideOverlayBounds(t *testing.T) {
	m := newPaletteModel([]paletteEntry{{Index: 1, Label: "back", Description: "go back", Action: paletteActionMsg{Verb: verbBack}}}).Open()
	updated, _, changed := m.UpdateMouse(tea.MouseMsg{X: 0, Y: 9, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}, 3, 80)
	if changed {
		t.Fatal("expected click outside overlay bounds to be ignored")
	}
	if !updated.IsOpen() {
		t.Fatal("expected palette to remain open after ignored click")
	}
}

func TestPaletteMouseIgnoresClicksInsideFrameOutsideContentBounds(t *testing.T) {
	m := newPaletteModel([]paletteEntry{{Index: 1, Label: "back", Description: "go back", Action: paletteActionMsg{Verb: verbBack}}}).Open()
	updated, _, changed := m.UpdateMouse(tea.MouseMsg{X: 4, Y: 9, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}, 3, 80)
	if changed {
		t.Fatal("expected click on overlay frame edge to be ignored")
	}
	if !updated.IsOpen() {
		t.Fatal("expected palette to remain open after frame-edge click")
	}
}

func TestPaletteMouseUsesRealViewportWidthForBounds(t *testing.T) {
	m := newPaletteModel([]paletteEntry{{Index: 1, Label: "back", Description: "go back", Action: paletteActionMsg{Verb: verbBack}}}).Open()
	updated, action, changed := m.UpdateMouse(tea.MouseMsg{X: 4, Y: 9, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}, 3, 42)
	if !changed {
		t.Fatal("expected narrow viewport click inside real content bounds to trigger selection")
	}
	if action.Verb != verbBack {
		t.Fatalf("expected back verb, got %q", action.Verb)
	}
	if updated.IsOpen() {
		t.Fatal("expected palette to close after successful pick")
	}
}

func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
