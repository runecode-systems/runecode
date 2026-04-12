package main

import "testing"

func TestPaletteDeleteQueryRunePreservesUTF8(t *testing.T) {
	m := newPaletteModel(shellRoutes())
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
