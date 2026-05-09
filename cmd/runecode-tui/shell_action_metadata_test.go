package main

import (
	"strings"
	"testing"
)

func TestHelpIncludesActionMetadataFromUnifiedDefinitions(t *testing.T) {
	m := newShellModel()
	cmd := m.commands.commands["shell.focus_main"]
	cmd.HelpText = "focus main pane — custom help text"
	m.commands.Register(cmd)
	m.actions = newShellActionGraph(m.routes, m.commands)

	help := renderHelp(m.keys, false, m.actions)
	if !strings.Contains(help, "focus main pane") || strings.Contains(help, "custom help text") {
		t.Fatalf("expected help to include action-metadata help text, got %q", help)
	}
}
