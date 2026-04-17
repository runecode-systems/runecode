package main

import (
	"fmt"
	"strings"
)

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
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, tableHeader(title))
	for i, item := range items {
		marker := " "
		if i == selected {
			marker = ">"
		}
		lines = append(lines, selectedLine(i == selected, fmt.Sprintf(" %s %s", marker, item)))
	}
	return compactLines(lines...)
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
