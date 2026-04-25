package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpUsesBaseBindingsForLeaderOverlay(t *testing.T) {
	m := newShellModel()
	help := renderHelp(m.keys, false, m.actions)
	if strings.Contains(help, "ctrl+n") || strings.Contains(help, "Open selected match") {
		t.Fatalf("expected leader/help footer not to advertise palette/session overlay bindings, got %q", help)
	}
	if !strings.Contains(help, "ctrl+p") || !strings.Contains(help, "ctrl+j") {
		t.Fatalf("expected base shell help bindings to remain visible, got %q", help)
	}
}

func TestShellViewLeaderOverlayKeepsBaseFooterHelp(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	shell := updated.(shellModel)
	v := shell.View()
	if !strings.Contains(v, "Leader Mode") {
		t.Fatalf("expected leader overlay in view, got %q", v)
	}
	if strings.Contains(v, "ctrl+n") || strings.Contains(v, "Open selected match") {
		t.Fatalf("expected footer help under leader overlay not to switch into palette/session bindings, got %q", v)
	}
}
