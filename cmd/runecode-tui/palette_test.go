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

	m = m.DeleteQueryRune()
	if m.query != "Go" {
		t.Fatalf("expected UTF-8-safe delete to keep valid string, got %q", m.query)
	}

	m = m.DeleteQueryRune()
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
	updated, _, changed := m.UpdateMouse(tea.MouseMsg{X: 1, Y: 9, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}, 3, 80)
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

func TestPaletteFilteringMatchesAcrossCachedNormalizedFields(t *testing.T) {
	entries := []paletteEntry{
		{Index: 1, Label: "jump route chat", Description: "open workspace home", Search: "session-alpha ws-home"},
		{Index: 2, Label: "inspect session beta", Description: "follow-up needed", Search: "workspace-2 run-2"},
	}
	m := newPaletteModel(entries).Open()
	if got := len(m.normalizedEntries); got != len(entries) {
		t.Fatalf("expected %d cached normalized entries, got %d", len(entries), got)
	}

	m.query = "WORKSPACE-2"
	updated, request, ok := m.BeginFilterRefresh()
	if !ok {
		t.Fatal("expected filter refresh request for workspace query")
	}
	m, _ = updated.ApplyFilterRefresh(buildPaletteFilterResult(request, updated.entriesVersion, updated.query, updated.normalizedEntries, updated.appliedNeedle, updated.matchIndexes))
	if m.MatchCount() != 1 {
		t.Fatalf("expected one workspace match, got %d", m.MatchCount())
	}
	matched, ok := m.MatchEntry(0)
	if !ok || matched.Index != 2 {
		t.Fatalf("expected search text match for second entry, got %+v ok=%v", matched, ok)
	}

	m.query = "FOLLOW-UP"
	updated, request, ok = m.BeginFilterRefresh()
	if !ok {
		t.Fatal("expected filter refresh request for description query")
	}
	m, _ = updated.ApplyFilterRefresh(buildPaletteFilterResult(request, updated.entriesVersion, updated.query, updated.normalizedEntries, updated.appliedNeedle, updated.matchIndexes))
	matched, ok = m.MatchEntry(0)
	if m.MatchCount() != 1 || !ok || matched.Index != 2 {
		t.Fatalf("expected description match for second entry, got %+v count=%d ok=%v", matched, m.MatchCount(), ok)
	}
}

func TestPaletteIncrementalFilteringNarrowsFromPreviousMatches(t *testing.T) {
	entries := []paletteEntry{
		{Index: 1, Label: "copy current identity", Description: "copy route identity", Search: "copy identity route"},
		{Index: 2, Label: "copy next route action", Description: "copy route action", Search: "copy next action"},
		{Index: 3, Label: "open runs", Description: "jump to runs", Search: "open runs"},
	}
	m := newPaletteModel(entries).Open()
	m = m.AppendQuery("c")
	updated, request, ok := m.BeginFilterRefresh()
	if !ok {
		t.Fatal("expected initial filter refresh request")
	}
	m, _ = updated.ApplyFilterRefresh(buildPaletteFilterResult(request, updated.entriesVersion, updated.query, updated.normalizedEntries, updated.appliedNeedle, updated.matchIndexes))
	if m.MatchCount() != 2 {
		t.Fatalf("expected broad initial copy matches, got %d", m.MatchCount())
	}
	m = m.AppendQuery("o")
	updated, request, ok = m.BeginFilterRefresh()
	if !ok {
		t.Fatal("expected second filter refresh request")
	}
	m, _ = updated.ApplyFilterRefresh(buildPaletteFilterResult(request, updated.entriesVersion, updated.query, updated.normalizedEntries, updated.appliedNeedle, updated.matchIndexes))
	if m.MatchCount() != 2 {
		t.Fatalf("expected incremental filter to retain two copy matches, got %d", m.MatchCount())
	}
	m = m.AppendQuery("p")
	updated, request, ok = m.BeginFilterRefresh()
	if !ok {
		t.Fatal("expected third filter refresh request")
	}
	m, _ = updated.ApplyFilterRefresh(buildPaletteFilterResult(request, updated.entriesVersion, updated.query, updated.normalizedEntries, updated.appliedNeedle, updated.matchIndexes))
	if m.MatchCount() != 2 {
		t.Fatalf("expected copy matches after third rune, got %d", m.MatchCount())
	}
}

func TestPaletteFilterRefreshRejectsStaleResultAfterQueryClears(t *testing.T) {
	entries := []paletteEntry{
		{Index: 1, Label: "open chat", Search: "chat"},
		{Index: 2, Label: "open runs", Search: "runs"},
	}
	m := newPaletteModel(entries).Open().AppendQuery("runs")
	filtering, request, ok := m.BeginFilterRefresh()
	if !ok {
		t.Fatal("expected non-empty query to request async filtering")
	}
	delayed := buildPaletteFilterResult(request, filtering.entriesVersion, filtering.query, filtering.normalizedEntries, filtering.appliedNeedle, filtering.matchIndexes)

	cleared := filtering
	for cleared.query != "" {
		cleared = cleared.DeleteQueryRune()
	}
	cleared, _, ok = cleared.BeginFilterRefresh()
	if ok {
		t.Fatal("expected empty query to apply full match set synchronously")
	}
	if cleared.MatchCount() != len(entries) {
		t.Fatalf("expected full match set after clearing query, got %d", cleared.MatchCount())
	}

	updated, applied := cleared.ApplyFilterRefresh(delayed)
	if applied {
		t.Fatal("expected stale delayed filter result to be rejected")
	}
	if updated.MatchCount() != len(entries) {
		t.Fatalf("expected stale filter result not to change matches, got %d", updated.MatchCount())
	}
}
