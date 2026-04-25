package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type shellHelpKeys []key.Binding

func (k shellHelpKeys) ShortHelp() []key.Binding  { return []key.Binding(k) }
func (k shellHelpKeys) FullHelp() [][]key.Binding { return [][]key.Binding{[]key.Binding(k)} }

func renderHelp(keys shellKeyMap, paletteOpen bool, actions shellActionGraph) string {
	bindings := keys.helpBindings(paletteOpen)
	bubbleBindings := make([]key.Binding, 0, len(bindings))
	for _, b := range bindings {
		bubbleBindings = append(bubbleBindings, key.NewBinding(key.WithKeys(b.Keys...), key.WithHelp(b.label(), b.Description)))
	}
	h := help.New()
	h.ShowAll = false
	view := "Help: " + h.View(shellHelpKeys(bubbleBindings))
	if entries := actions.helpEntries(6); len(entries) > 0 {
		view += " | Actions: " + strings.Join(entries, " · ")
	}
	return view
}
