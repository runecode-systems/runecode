package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

type shellPaletteEntriesLoadedMsg struct {
	request uint64
	entries []paletteEntry
}

type shellPaletteFilterLoadedMsg struct {
	request        uint64
	entriesVersion uint64
	needle         string
	matches        []int
}

type paletteModel struct {
	open              bool
	query             string
	appliedNeedle     string
	selectedIndex     int
	matchIndexes      []int
	entries           []paletteEntry
	normalizedEntries []string
	entriesVersion    uint64
	entriesRequest    uint64
	filterRequest     uint64
	entriesLoading    bool
	filterLoading     bool
}

func newPaletteModel(entries []paletteEntry) paletteModel {
	return paletteModel{}.UpdateEntries(entries)
}

func (m paletteModel) UpdateEntries(entries []paletteEntry) paletteModel {
	m.entries = append([]paletteEntry(nil), entries...)
	m.normalizedEntries = buildPaletteNormalizedEntries(m.entries)
	m.entriesVersion++
	m.entriesLoading = false
	m.filterLoading = false
	m.appliedNeedle = ""
	m.selectedIndex = 0
	m.matchIndexes = buildPaletteFullIndexes(m.matchIndexes[:0], len(m.entries))
	return m
}

func (m paletteModel) Open() paletteModel {
	m.open = true
	m.query = ""
	m.appliedNeedle = ""
	m.selectedIndex = 0
	m.filterLoading = false
	m.matchIndexes = buildPaletteFullIndexes(m.matchIndexes[:0], len(m.entries))
	return m
}

func (m paletteModel) Close() paletteModel {
	m.open = false
	m.entriesLoading = false
	m.filterLoading = false
	return m
}

func (m paletteModel) IsOpen() bool {
	return m.open
}

func (m paletteModel) MatchCount() int {
	return len(m.matchIndexes)
}

func (m paletteModel) MatchEntry(index int) (paletteEntry, bool) {
	if index < 0 || index >= len(m.matchIndexes) {
		return paletteEntry{}, false
	}
	entryIndex := m.matchIndexes[index]
	if entryIndex < 0 || entryIndex >= len(m.entries) {
		return paletteEntry{}, false
	}
	return m.entries[entryIndex], true
}

func (m paletteModel) SelectedEntry() (paletteEntry, bool) {
	if len(m.matchIndexes) == 0 {
		return paletteEntry{}, false
	}
	selected := m.selectedIndex
	if selected < 0 {
		selected = 0
	}
	if selected >= len(m.matchIndexes) {
		selected = len(m.matchIndexes) - 1
	}
	return m.MatchEntry(selected)
}

func (m paletteModel) BeginEntriesRefresh() (paletteModel, uint64) {
	m.entriesRequest++
	m.entriesLoading = true
	return m, m.entriesRequest
}

func (m paletteModel) ApplyEntriesRefresh(msg shellPaletteEntriesLoadedMsg) (paletteModel, bool) {
	if msg.request != m.entriesRequest {
		return m, false
	}
	m = m.UpdateEntries(msg.entries)
	return m, true
}

func (m paletteModel) BeginFilterRefresh() (paletteModel, uint64, bool) {
	needle := normalizePaletteQuery(m.query)
	if needle == "" {
		m.filterLoading = false
		m.appliedNeedle = ""
		m.matchIndexes = buildPaletteFullIndexes(m.matchIndexes[:0], len(m.entries))
		return m, 0, false
	}
	m.filterRequest++
	m.filterLoading = true
	return m, m.filterRequest, true
}

func (m paletteModel) ApplyFilterRefresh(msg shellPaletteFilterLoadedMsg) (paletteModel, bool) {
	if msg.request != m.filterRequest || msg.entriesVersion != m.entriesVersion {
		return m, false
	}
	m.filterLoading = false
	m.appliedNeedle = msg.needle
	m.matchIndexes = append(m.matchIndexes[:0], msg.matches...)
	m.clampSelectedIndex()
	return m, true
}

func (m paletteModel) AppendQuery(value string) paletteModel {
	if strings.TrimSpace(value) == "" {
		return m
	}
	m.query += value
	m.selectedIndex = 0
	return m
}

func (m paletteModel) DeleteQueryRune() paletteModel {
	if len(m.query) == 0 {
		return m
	}
	_, size := utf8.DecodeLastRuneInString(m.query)
	if size <= 0 {
		return m
	}
	m.query = m.query[:len(m.query)-size]
	m.selectedIndex = 0
	return m
}

func (m paletteModel) Update(msg tea.Msg, keys shellKeyMap) (paletteModel, paletteActionMsg, bool) {
	if !m.open {
		return m, paletteActionMsg{}, false
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, paletteActionMsg{}, false
	}
	return m.updateKey(key, keys)
}

func (m paletteModel) UpdateMouse(msg tea.MouseMsg, paletteStartY int, layoutWidth int) (paletteModel, paletteActionMsg, bool) {
	if !m.open {
		return m, paletteActionMsg{}, false
	}
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionRelease {
		return m, paletteActionMsg{}, false
	}
	index, ok := m.matchIndexAtPosition(msg.X, msg.Y, paletteStartY, layoutWidth)
	if !ok {
		return m, paletteActionMsg{}, false
	}
	m.selectedIndex = index
	return m.pickRoute()
}

func (m paletteModel) updateKey(key tea.KeyMsg, keys shellKeyMap) (paletteModel, paletteActionMsg, bool) {
	switch {
	case keys.PaletteClose.matches(key):
		return m.Close(), paletteActionMsg{}, false
	case keys.PalettePick.matches(key):
		return m.pickRoute()
	case keys.PaletteNext.matches(key):
		return m.stepSelection(1), paletteActionMsg{}, false
	case keys.PalettePrev.matches(key):
		return m.stepSelection(-1), paletteActionMsg{}, false
	case key.Type == tea.KeyBackspace || key.Type == tea.KeyDelete:
		return m.DeleteQueryRune(), paletteActionMsg{}, false
	case isTypingKey(key):
		return m.AppendQuery(key.String()), paletteActionMsg{}, false
	default:
		return m, paletteActionMsg{}, false
	}
}

func (m paletteModel) pickRoute() (paletteModel, paletteActionMsg, bool) {
	r, ok := m.SelectedEntry()
	if !ok {
		return m.Close(), paletteActionMsg{}, false
	}
	return m.Close(), r.Action, true
}

func (m paletteModel) stepSelection(delta int) paletteModel {
	if len(m.matchIndexes) == 0 {
		return m
	}
	if delta > 0 {
		m.selectedIndex = (m.selectedIndex + 1) % len(m.matchIndexes)
		return m
	}
	m.selectedIndex--
	if m.selectedIndex < 0 {
		m.selectedIndex = len(m.matchIndexes) - 1
	}
	return m
}

func (m *paletteModel) clampSelectedIndex() {
	if m.selectedIndex >= len(m.matchIndexes) {
		if len(m.matchIndexes) == 0 {
			m.selectedIndex = 0
		} else {
			m.selectedIndex = len(m.matchIndexes) - 1
		}
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
}

func buildPaletteFilterResult(request uint64, entriesVersion uint64, query string, normalizedEntries []string, priorNeedle string, priorMatches []int) shellPaletteFilterLoadedMsg {
	needle := normalizePaletteQuery(query)
	searchSpace := paletteFilterSearchSpace(needle, priorNeedle, priorMatches, len(normalizedEntries))
	return shellPaletteFilterLoadedMsg{
		request:        request,
		entriesVersion: entriesVersion,
		needle:         needle,
		matches:        filterPaletteMatchIndexes(needle, normalizedEntries, searchSpace),
	}
}

func paletteFilterSearchSpace(needle string, priorNeedle string, priorMatches []int, total int) []int {
	if priorNeedle != "" && strings.HasPrefix(needle, priorNeedle) && len(priorMatches) > 0 {
		return append([]int(nil), priorMatches...)
	}
	return buildPaletteFullIndexes(make([]int, 0, total), total)
}

func filterPaletteMatchIndexes(needle string, normalizedEntries []string, searchSpace []int) []int {
	if needle == "" {
		return append([]int(nil), searchSpace...)
	}
	matches := make([]int, 0, len(searchSpace))
	for _, i := range searchSpace {
		if i < 0 || i >= len(normalizedEntries) {
			continue
		}
		if strings.Contains(normalizedEntries[i], needle) {
			matches = append(matches, i)
		}
	}
	return matches
}

func buildPaletteFullIndexes(dst []int, count int) []int {
	dst = dst[:0]
	for i := 0; i < count; i++ {
		dst = append(dst, i)
	}
	return dst
}

func normalizePaletteQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func buildPaletteNormalizedEntries(entries []paletteEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	normalized := make([]string, len(entries))
	for i, entry := range entries {
		normalized[i] = normalizePaletteEntrySearch(entry)
	}
	return normalized
}

func normalizePaletteEntrySearch(entry paletteEntry) string {
	var b strings.Builder
	b.Grow(len(entry.Label) + len(entry.Description) + len(entry.Search) + 2)
	b.WriteString(strings.TrimSpace(entry.Label))
	b.WriteByte('\n')
	b.WriteString(strings.TrimSpace(entry.Description))
	b.WriteByte('\n')
	b.WriteString(strings.TrimSpace(entry.Search))
	return strings.ToLower(b.String())
}

func (m paletteModel) matchIndexAtPosition(x int, y int, paletteStartY int, layoutWidth int) (int, bool) {
	if len(m.matchIndexes) == 0 {
		return 0, false
	}
	startX, endX := centeredOverlayContentBounds(layoutWidth)
	if x < startX || x > endX {
		return 0, false
	}
	startY := paletteStartY + 5
	for _, candidateY := range []int{y, y - 1} {
		index := candidateY - startY
		if index >= 0 && index < len(m.matchIndexes) {
			return index, true
		}
	}
	return 0, false
}

func paletteOverlayBounds(layoutWidth int) (int, int) {
	return centeredOverlayContentBounds(layoutWidth)
}

func paletteMatchLine(entry paletteEntry, selected bool) string {
	marker := "•"
	if selected {
		marker = "▶"
	}
	line := " " + marker + " " + fmt.Sprintf("%d. %s — %s", entry.Index, entry.Label, entry.Description)
	return line
}

func paletteMatchLineBounded(entry paletteEntry, width int) string {
	label := strings.TrimSpace(entry.Label)
	description := strings.TrimSpace(entry.Description)
	kind := paletteEntryShortcut(entry)
	if width <= 0 {
		if description == "" {
			return label
		}
		return label + "  " + muted(description)
	}
	parts := []string{label}
	if description != "" {
		parts = append(parts, description)
	}
	if kind != "" {
		parts = append(parts, "["+kind+"]")
	}
	return clipDisplayText(strings.Join(parts, "  "), width)
}

func paletteEntryShortcut(entry paletteEntry) string {
	if strings.TrimSpace(entry.Action.Target.CommandID) != "" {
		return "Cmd"
	}
	if route := strings.TrimSpace(string(entry.Action.Target.RouteID)); route != "" {
		return "Route"
	}
	if strings.TrimSpace(entry.Action.Target.SessionID) != "" {
		return "Session"
	}
	if strings.TrimSpace(entry.Action.Target.RunID) != "" {
		return "Run"
	}
	if strings.TrimSpace(entry.Action.Target.ApprovalID) != "" {
		return "Approval"
	}
	if strings.TrimSpace(entry.Action.Target.Digest) != "" {
		return "Evidence"
	}
	if kind := strings.TrimSpace(entry.Action.Target.Kind); kind != "" {
		return kind
	}
	return string(entry.Action.Verb)
}

func paletteSearchText(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return muted("type to filter commands, routes, sessions, and evidence")
	}
	return query
}

func isTypingKey(msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		return r >= 32 && r != 127
	}
	return false
}
