package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpUsesBaseBindingsForLeaderOverlay(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	help := renderHelp(m.keys, false, m.actions)
	if strings.TrimSpace(help) != "" {
		t.Fatalf("expected footer help removed in favor of leader overlay help, got %q", help)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	v := updated.(shellModel).View()
	for _, want := range []string{"Leader Mode", "Help:", "ctrl+p opens quick jump palette", "ctrl+j opens session quick switcher"} {
		if !strings.Contains(v, want) {
			t.Fatalf("expected leader overlay help cue %q, got %q", want, v)
		}
	}
	if strings.Contains(v, "ctrl+n") || strings.Contains(v, "Open selected match") {
		t.Fatalf("expected leader overlay help not to advertise palette/session overlay bindings, got %q", v)
	}
	if strings.Contains(help, "ctrl+n") || strings.Contains(help, "Open selected match") {
		t.Fatalf("expected leader/help footer not to advertise palette/session overlay bindings, got %q", help)
	}
	if !strings.Contains(v, "tab next focus area") || !strings.Contains(v, "shift+tab previous focus area") {
		t.Fatalf("expected base shell focus bindings to move into leader root help, got %q", v)
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
	if !strings.Contains(v, "Help:") {
		t.Fatalf("expected leader overlay to carry relocated help copy, got %q", v)
	}
	if strings.Contains(v, "ctrl+n") || strings.Contains(v, "Open selected match") {
		t.Fatalf("expected footer help under leader overlay not to switch into palette/session bindings, got %q", v)
	}
}

func TestHelpFooterStaysCompactWhilePreservingDiscovery(t *testing.T) {
	m := newShellModel()
	help := renderHelp(m.keys, false, m.actions)
	if strings.TrimSpace(help) != "" {
		t.Fatalf("expected footer help removed to reclaim pane height, got %q", help)
	}
}
