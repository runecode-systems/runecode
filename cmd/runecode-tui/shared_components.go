package main

import (
	"fmt"
	"strings"
)

type boundedListRow struct {
	Text       string
	Selectable bool
}

type boundedListSpec struct {
	Rows          []boundedListRow
	Selected      int
	Width         int
	Height        int
	GapMarker     string
	Empty         string
	PreserveGaps  bool
	ApplySelected bool
}

type inspectorContentKind string

const (
	inspectorContentTranscript inspectorContentKind = "transcript"
	inspectorContentDiff       inspectorContentKind = "diff"
	inspectorContentLog        inspectorContentKind = "log"
	inspectorContentMarkdown   inspectorContentKind = "markdown"
	inspectorContentRaw        inspectorContentKind = "raw"
	inspectorContentStructured inspectorContentKind = "structured"
)

type inspectorReference struct {
	Label string
	Items []inspectorReferenceItem
}

type inspectorReferenceItem struct {
	Label  string
	Action paletteActionMsg
}

type inspectorShellSpec struct {
	Title          string
	Summary        string
	Identity       string
	Status         string
	Badges         []string
	References     []inspectorReference
	LocalActions   []routeActionItem
	ModeTabs       []string
	ActiveMode     string
	ContentKind    inspectorContentKind
	ContentLabel   string
	Content        string
	ViewportWidth  int
	ViewportHeight int
	Document       *longFormDocumentState
	CopyActions    []routeCopyAction
}

func renderDirectory(title string, items []string, selected int) string {
	if len(items) == 0 {
		return renderStateCard(routeLoadStateEmpty, title, "no items")
	}
	rows := make([]boundedListRow, 0, len(items))
	for i, item := range items {
		marker := " "
		if i == selected {
			marker = ">"
		}
		rows = append(rows, boundedListRow{Text: fmt.Sprintf(" %s %s", marker, item), Selectable: true})
	}
	return compactLines(
		tableHeader(title),
		renderBoundedList(boundedListSpec{Rows: rows, Selected: selected, ApplySelected: true, PreserveGaps: true}),
	)
}

func renderBoundedList(spec boundedListSpec) string {
	rows := normalizeBoundedListRows(spec.Rows, spec.PreserveGaps)
	if len(rows) == 0 {
		empty := strings.TrimSpace(spec.Empty)
		if empty == "" {
			empty = "no items"
		}
		return clipBoundedListText(empty, spec.Width)
	}

	selectedRow := boundedListSelectedRow(rows, spec.Selected)
	windowStart, windowEnd, showTopGap, showBottomGap := boundedListWindow(len(rows), selectedRow, spec.Height)
	gap := strings.TrimSpace(spec.GapMarker)
	if gap == "" {
		gap = "..."
	}

	lines := make([]string, 0, (windowEnd-windowStart)+2)
	if showTopGap {
		lines = append(lines, clipBoundedListText(gap, spec.Width))
	}
	for rowIdx := windowStart; rowIdx < windowEnd; rowIdx++ {
		line := clipBoundedListText(rows[rowIdx].Text, spec.Width)
		if spec.ApplySelected {
			line = selectedLine(rowIdx == selectedRow && rows[rowIdx].Selectable, line)
		}
		lines = append(lines, line)
	}
	if showBottomGap {
		lines = append(lines, clipBoundedListText(gap, spec.Width))
	}
	return strings.Join(lines, "\n")
}

func normalizeBoundedListRows(rows []boundedListRow, preserveGaps bool) []boundedListRow {
	if preserveGaps {
		return append([]boundedListRow(nil), rows...)
	}
	normalized := make([]boundedListRow, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Text) == "" {
			continue
		}
		normalized = append(normalized, row)
	}
	return normalized
}

func boundedListSelectedRow(rows []boundedListRow, selected int) int {
	selectable := make([]int, 0, len(rows))
	for i, row := range rows {
		if row.Selectable {
			selectable = append(selectable, i)
		}
	}
	if len(selectable) == 0 {
		return -1
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(selectable) {
		selected = len(selectable) - 1
	}
	return selectable[selected]
}

func boundedListWindow(totalRows int, selectedRow int, height int) (start int, end int, showTopGap bool, showBottomGap bool) {
	if totalRows <= 0 {
		return 0, 0, false, false
	}
	if height <= 0 || totalRows <= height {
		return 0, totalRows, false, false
	}
	selectedRow = clampBoundedListSelectedRow(selectedRow, totalRows)
	if height == 1 {
		return selectedRow, selectedRow + 1, false, false
	}
	showTopGap, showBottomGap = boundedListGapFlags(totalRows, selectedRow, height)
	if height == 2 {
		return boundedListTwoRowWindow(selectedRow, showTopGap, showBottomGap)
	}
	dataSlots := boundedListDataSlots(height, showTopGap, showBottomGap)
	start = boundedListWindowStart(totalRows, selectedRow, dataSlots)
	end = start + dataSlots
	if end > totalRows {
		end = totalRows
	}
	showTopGap = start > 0
	showBottomGap = end < totalRows
	return start, end, showTopGap, showBottomGap
}

func clampBoundedListSelectedRow(selectedRow int, totalRows int) int {
	if selectedRow < 0 || selectedRow >= totalRows {
		return 0
	}
	return selectedRow
}

func boundedListGapFlags(totalRows int, selectedRow int, height int) (bool, bool) {
	nearTopThreshold := (height - 2) / 2
	nearBottomThreshold := totalRows - 1 - nearTopThreshold
	return selectedRow > nearTopThreshold, selectedRow < nearBottomThreshold
}

func boundedListTwoRowWindow(selectedRow int, showTopGap bool, showBottomGap bool) (int, int, bool, bool) {
	if showTopGap && !showBottomGap {
		return selectedRow, selectedRow + 1, true, false
	}
	return selectedRow, selectedRow + 1, false, true
}

func boundedListDataSlots(height int, showTopGap bool, showBottomGap bool) int {
	dataSlots := height
	if showTopGap {
		dataSlots--
	}
	if showBottomGap {
		dataSlots--
	}
	if dataSlots < 1 {
		return 1
	}
	return dataSlots
}

func boundedListWindowStart(totalRows int, selectedRow int, dataSlots int) int {
	start := selectedRow - (dataSlots / 2)
	if start < 0 {
		start = 0
	}
	if start+dataSlots > totalRows {
		start = totalRows - dataSlots
	}
	if start < 0 {
		return 0
	}
	return start
}

func clipBoundedListText(text string, width int) string {
	if width <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func renderInspectorHeader(title string, badges ...string) string {
	line := strings.TrimSpace(title)
	if line == "" {
		line = "Inspector"
	} else if !strings.HasPrefix(strings.ToLower(line), "inspector") {
		line = "Inspector " + line
	}
	if len(badges) == 0 {
		return tableHeader(line)
	}
	return tableHeader(line) + " " + strings.Join(badges, " ")
}

func renderModeSwitchTabs(tabs []string, active string) string {
	if len(tabs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		if tab == active {
			parts = append(parts, "["+strings.ToUpper(tab)+"]")
			continue
		}
		parts = append(parts, tab)
	}
	return "Modes: " + strings.Join(parts, " | ")
}

func renderInspectorShell(spec inspectorShellSpec) string {
	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = "Inspector"
	}
	lines := []string{renderInspectorHeader(title, spec.Badges...)}
	lines = append(lines, renderInspectorOverview(spec)...)
	lines = append(lines, renderInspectorReferences(spec.References)...)
	lines = append(lines, renderInspectorActions(spec.LocalActions, spec.CopyActions)...)
	if tabs := renderModeSwitchTabs(spec.ModeTabs, spec.ActiveMode); strings.TrimSpace(tabs) != "" {
		lines = append(lines, tabs)
	}
	lines = append(lines, renderInspectorDetailViewport(spec)...)
	return compactLines(lines...)
}

func renderInspectorOverview(spec inspectorShellSpec) []string {
	lines := []string{tableHeader("Overview")}
	if summary := strings.TrimSpace(spec.Summary); summary != "" {
		lines = append(lines, "Summary: "+summary)
	}
	if identity := strings.TrimSpace(spec.Identity); identity != "" {
		lines = append(lines, "Identity: "+identity)
	}
	if status := strings.TrimSpace(spec.Status); status != "" {
		lines = append(lines, "Status: "+status)
	}
	if len(spec.ModeTabs) > 0 {
		lines = append(lines, fmt.Sprintf("Summary → detail: %s mode", strings.ToUpper(strings.TrimSpace(spec.ActiveMode))))
	}
	return lines
}

func renderInspectorReferences(refs []inspectorReference) []string {
	if len(refs) == 0 {
		return nil
	}
	lines := []string{tableHeader("Linked references")}
	for _, ref := range refs {
		items := make([]string, 0, len(ref.Items))
		for _, item := range ref.Items {
			label := strings.TrimSpace(item.Label)
			if label == "" {
				continue
			}
			items = append(items, label)
		}
		lines = append(lines, renderLinkedReferenceLine("Linked "+defaultInspectorReferenceLabel(ref.Label), items))
	}
	return lines
}

func defaultInspectorReferenceLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "linked"
	}
	return label
}

func mapReferenceIDs(ids []string, build func(string) paletteActionMsg) []inspectorReferenceItem {
	items := make([]inspectorReferenceItem, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		items = append(items, inspectorReferenceItem{Label: id, Action: build(id)})
	}
	return items
}

func renderInspectorActions(local []routeActionItem, copyActions []routeCopyAction) []string {
	if len(local) == 0 && len(copyActions) == 0 {
		return nil
	}
	lines := []string{tableHeader("Actions")}
	if len(local) > 0 {
		labels := make([]string, 0, len(local))
		for _, action := range local {
			label := strings.TrimSpace(action.Label)
			if label == "" {
				continue
			}
			labels = append(labels, label)
		}
		if len(labels) > 0 {
			lines = append(lines, "Local actions: "+strings.Join(labels, " | "))
		}
	}
	if labels := inspectorCopyActionLabels(copyActions); len(labels) > 0 {
		lines = append(lines, "Copy actions: "+strings.Join(labels, " | "))
	}
	return lines
}

func inspectorCopyActionLabels(copyActions []routeCopyAction) []string {
	labels := make([]string, 0, len(copyActions))
	for _, action := range copyActions {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			label = strings.TrimSpace(action.ID)
		}
		if label == "" {
			continue
		}
		labels = append(labels, label)
	}
	return labels
}

func renderInspectorDetailViewport(spec inspectorShellSpec) []string {
	if spec.Document != nil {
		return []string{tableHeader("Detail viewport"), "Long-form " + spec.Document.contentLabel() + ":", spec.Document.Render()}
	}
	contentLabel := strings.TrimSpace(spec.ContentLabel)
	if contentLabel == "" {
		contentLabel = "content"
	}
	renderedContent := renderInspectorLongForm(spec.ContentKind, spec.Content, spec.ViewportWidth, spec.ViewportHeight)
	return []string{tableHeader("Detail viewport"), "Long-form " + contentLabel + ":", renderedContent}
}

func renderStateCard(state routeLoadState, title, message string) string {
	label := string(state)
	if label == "" {
		label = string(routeLoadStateReady)
	}
	if strings.TrimSpace(message) == "" {
		message = "n/a"
	}
	return compactLines(
		tableHeader(strings.ToUpper(label))+" "+title,
		message,
	)
}
