package main

import "strings"

func renderHelp(keys shellKeyMap, paletteOpen bool) string {
	bindings := keys.helpBindings(paletteOpen)
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		parts = append(parts, b.label()+" "+b.Description)
	}
	return "Help: " + strings.Join(parts, " | ")
}
