package main

import (
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

func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
