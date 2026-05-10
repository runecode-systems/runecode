package main

import "github.com/charmbracelet/bubbles/key"

type shellHelpKeys []key.Binding

func (k shellHelpKeys) ShortHelp() []key.Binding  { return []key.Binding(k) }
func (k shellHelpKeys) FullHelp() [][]key.Binding { return [][]key.Binding{[]key.Binding(k)} }

func renderHelp(keys shellKeyMap, paletteOpen bool, actions shellActionGraph) string {
	_ = keys
	_ = paletteOpen
	_ = actions
	return ""
}
