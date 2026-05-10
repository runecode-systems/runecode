package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpIncludesActionMetadataFromUnifiedDefinitions(t *testing.T) {
	m := newShellModel()
	m.width = 150
	m.height = 40
	cmd := m.commands.commands["shell.focus_main"]
	cmd.HelpText = "focus main pane — custom help text"
	cmd.LeaderPath = []string{"w", "m"}
	cmd.LeaderGroup = "Workbench"
	m.commands.Register(cmd)
	m.actions = newShellActionGraph(m.routes, m.commands)

	help := renderHelp(m.keys, false, m.actions)
	if !strings.Contains(help, "ctrl+p commands") {
		t.Fatalf("expected compact footer help, got %q", help)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	v := updated.(shellModel).View()
	if !strings.Contains(v, "Move focus to main content") || strings.Contains(v, "custom help text") {
		t.Fatalf("expected leader overlay to use compact action descriptions without long help text, got %q", v)
	}
}
