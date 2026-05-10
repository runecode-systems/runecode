package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

const artifactPreviewLineLimit = 10
const artifactCopyPreviewLineLimit = 8

func renderArtifactList(items []brokerapi.ArtifactSummary, selected int) string {
	if len(items) == 0 {
		return "  - no artifacts"
	}
	line := ""
	for i, item := range items {
		marker := " "
		if i == selected {
			marker = ">"
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s", marker, renderArtifactDirectoryRow(item))) + "\n"
	}
	return line
}

func renderArtifactDirectoryItems(items []brokerapi.ArtifactSummary) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, renderArtifactDirectoryRow(item))
	}
	return out
}

func renderArtifactDirectoryRow(item brokerapi.ArtifactSummary) string {
	parts := []string{artifactDisplayLabel(item), shortIdentity(item.Reference.Digest)}
	meta := []string{}
	if preview := artifactPreviewIdentity(item.Reference.SizeBytes, item.Reference.ContentType); preview != "" {
		meta = append(meta, preview)
	}
	if len(meta) > 0 {
		parts = append(parts, "• "+strings.Join(meta, " • "))
	}
	return strings.Join(parts, " ")
}

func renderArtifactInspector(head *brokerapi.LocalArtifactHeadResponse, mode artifactDetailMode, presentation contentPresentationMode, content, contentErr string, document *longFormDocumentState) string {
	if head == nil {
		return "  Select an artifact and press enter to load detail."
	}
	if document == nil {
		fallback := newLongFormDocumentState()
		document = &fallback
	}
	a := head.Artifact
	mode = normalizeArtifactMode(mode)
	presentation = normalizePresentationMode(presentation)
	contentView := renderArtifactContent(mode, presentation, content, contentErr)
	kind := artifactContentKind(mode, a.Reference.ContentType, presentation)
	document.SetDocument(workbenchObjectRef{Kind: "artifact", ID: strings.TrimSpace(a.Reference.Digest)}, kind, fmt.Sprintf("%s content", mode), compactLines(
		fmt.Sprintf("Evidence label: %s", artifactDisplayLabel(a)),
		fmt.Sprintf("Data class: %s", a.Reference.DataClass),
		fmt.Sprintf("Evidence trail: run %s -> artifact %s -> Audit for final checks and receipts", valueOrNA(a.RunID), artifactDisplayLabel(a)),
		fmt.Sprintf("Primary digest display: %s (copy raw digest below)", shortIdentity(a.Reference.Digest)),
		fmt.Sprintf("Detail mode: %s (typed metadata stays authoritative)", mode),
		fmt.Sprintf("Presentation mode: %s", presentation),
		fmt.Sprintf("Provenance receipt: %s", a.Reference.ProvenanceReceiptHash),
		"Inspectable content is supplemental evidence, not authoritative run/approval truth.",
		contentView,
	))
	return renderInspectorShell(inspectorShellSpec{
		Title:    "Artifact inspector",
		Summary:  fmt.Sprintf("%s ready to inspect.", artifactDisplayLabel(a)),
		Identity: fmt.Sprintf("Artifact %s • digest %s", artifactDisplayLabel(a), shortIdentity(a.Reference.Digest)),
		Status:   fmt.Sprintf("From run %s • open Audit next for final checks or receipts", valueOrNA(a.RunID)),
		Badges:   []string{stateBadgeWithLabel("class", fmt.Sprintf("%v", a.Reference.DataClass)), appTheme.InspectorHint.Render("evidence details")},
		References: []inspectorReference{{Label: "run", Items: mapReferenceIDs([]string{a.RunID}, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: id}}
		})}, {Label: "artifact", Items: mapReferenceIDs([]string{a.Reference.Digest}, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: id}}
		})}},
		LocalActions: artifactInspectorLocalActions(),
		CopyActions:  artifactRouteCopyActions(head, content),
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		Document:     document,
	})
}

func artifactInspectorLocalActions() []routeActionItem {
	return []routeActionItem{
		{Label: "jump:runs", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeRuns}}},
		{Label: "jump:audit", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}},
		{Label: "copy:digest"},
		{Label: "copy:provenance_receipt"},
		{Label: "copy:artifact_preview"},
	}
}

func artifactInspectorReferenceActions(head *brokerapi.LocalArtifactHeadResponse) []routeActionItem {
	if head == nil {
		return nil
	}
	items := mapReferenceIDs([]string{head.Artifact.RunID}, func(id string) paletteActionMsg {
		return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: id}}
	})
	items = append(items, mapReferenceIDs([]string{head.Artifact.Reference.Digest}, func(id string) paletteActionMsg {
		return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: id}}
	})...)
	out := make([]routeActionItem, 0, len(items))
	for _, item := range items {
		prefix := "run:"
		if strings.Contains(item.Label, "sha256:") {
			prefix = "artifact:"
		}
		out = append(out, routeActionItem{Label: prefix + item.Label, Action: item.Action})
	}
	return out
}

func artifactRouteCopyActions(head *brokerapi.LocalArtifactHeadResponse, content string) []routeCopyAction {
	if head == nil {
		return nil
	}
	ref := head.Artifact.Reference
	preview := strings.TrimSpace(content)
	preview = boundedArtifactPreview(preview, artifactCopyPreviewLineLimit)
	return compactCopyActions([]routeCopyAction{
		{ID: "digest", Label: "artifact digest", Text: ref.Digest},
		{ID: "provenance_receipt", Label: "provenance receipt", Text: ref.ProvenanceReceiptHash},
		{ID: "artifact_preview", Label: "artifact preview", Text: preview},
	})
}

func artifactContentKind(mode artifactDetailMode, contentType string, presentation contentPresentationMode) inspectorContentKind {
	if presentation == presentationRaw {
		return inspectorContentRaw
	}
	if presentation == presentationStructured {
		return inspectorContentStructured
	}
	lowerType := strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(lowerType, "markdown") {
		return inspectorContentMarkdown
	}
	switch mode {
	case artifactModeDiff:
		return inspectorContentDiff
	case artifactModeLog:
		return inspectorContentLog
	default:
		return inspectorContentRaw
	}
}

func preferredArtifactMode(dataClass any) artifactDetailMode {
	value := strings.ToLower(fmt.Sprintf("%v", dataClass))
	switch {
	case strings.Contains(value, "diff"):
		return artifactModeDiff
	case strings.Contains(value, "log"):
		return artifactModeLog
	default:
		return artifactModeResult
	}
}

func normalizeArtifactMode(mode artifactDetailMode) artifactDetailMode {
	switch mode {
	case artifactModeDiff, artifactModeLog, artifactModeResult:
		return mode
	default:
		return artifactModeResult
	}
}

func nextArtifactMode(current artifactDetailMode) artifactDetailMode {
	switch normalizeArtifactMode(current) {
	case artifactModeDiff:
		return artifactModeLog
	case artifactModeLog:
		return artifactModeResult
	default:
		return artifactModeDiff
	}
}

func renderArtifactContent(mode artifactDetailMode, presentation contentPresentationMode, content, contentErr string) string {
	presentation = normalizePresentationMode(presentation)
	if contentErr != "" {
		return fmt.Sprintf("  %s content unavailable: %s", mode, contentErr)
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Sprintf("  %s content unavailable for current artifact.", mode)
	}
	if presentation == presentationRaw {
		return fmt.Sprintf("  %s raw (secrets redacted):\n%s", mode, redactSecrets(content))
	}
	lines := strings.Split(content, "\n")
	if presentation == presentationStructured {
		first := strings.TrimSpace(lines[0])
		last := strings.TrimSpace(lines[len(lines)-1])
		first = redactSecrets(first)
		last = redactSecrets(last)
		if len(lines) > 12 {
			return fmt.Sprintf("  %s structured:\n  - lines=%d\n  - non_empty=%d\n  - preview_first=%q\n  - preview_last=%q", mode, len(lines), countNonEmptyLines(lines), first, last)
		}
		return fmt.Sprintf("  %s structured:\n  - lines=%d\n  - non_empty=%d\n  - preview=%q", mode, len(lines), countNonEmptyLines(lines), redactSecrets(strings.TrimSpace(content)))
	}
	if len(lines) > artifactPreviewLineLimit {
		return fmt.Sprintf("  %s preview (secrets redacted):\n%s\n  ... (%d more lines)", mode, redactSecrets(strings.Join(lines[:artifactPreviewLineLimit], "\n")), len(lines)-artifactPreviewLineLimit)
	}
	return fmt.Sprintf("  %s preview (secrets redacted):\n%s", mode, redactSecrets(content))
}

func boundedArtifactPreview(content string, maxLines int) string {
	preview := strings.TrimSpace(content)
	if preview == "" {
		return ""
	}
	lines := strings.Split(preview, "\n")
	if len(lines) > maxLines {
		preview = strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
	}
	return redactSecrets(preview)
}

func countNonEmptyLines(lines []string) int {
	total := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			total++
		}
	}
	return total
}

func (m *artifactsRouteModel) syncDetailDocument() {
	if m.active == nil {
		m.detailDoc.SetDocument(workbenchObjectRef{Kind: "artifact", ID: "none"}, inspectorContentRaw, "artifact content", "")
		return
	}
	a := m.active.Artifact
	mode := normalizeArtifactMode(m.mode)
	presentation := normalizePresentationMode(m.presentation)
	contentView := renderArtifactContent(mode, presentation, m.content, m.contentErr)
	kind := artifactContentKind(mode, a.Reference.ContentType, presentation)
	content := compactLines(
		fmt.Sprintf("Evidence label: %s", artifactDisplayLabel(a)),
		fmt.Sprintf("Data class: %s", a.Reference.DataClass),
		fmt.Sprintf("Evidence trail: run %s -> artifact %s -> Audit for final checks and receipts", valueOrNA(a.RunID), artifactDisplayLabel(a)),
		fmt.Sprintf("Primary digest display: %s (copy raw digest below)", shortIdentity(a.Reference.Digest)),
		fmt.Sprintf("Detail mode: %s (typed metadata stays authoritative)", mode),
		fmt.Sprintf("Presentation mode: %s", presentation),
		fmt.Sprintf("Provenance receipt: %s", a.Reference.ProvenanceReceiptHash),
		"Inspectable content is supplemental evidence, not authoritative run/approval truth.",
		contentView,
	)
	ref := workbenchObjectRef{Kind: "artifact", ID: strings.TrimSpace(a.Reference.Digest)}
	m.detailDoc.SetDocument(ref, kind, fmt.Sprintf("%s content", mode), content)
}

func renderArtifactOverviewCard(head *brokerapi.LocalArtifactHeadResponse, classFilter string) string {
	if head == nil {
		message := "Select evidence to review what the run produced before you continue to Audit."
		if strings.TrimSpace(classFilter) != "" {
			message = fmt.Sprintf("Select %s evidence to review what the run produced before you continue to Audit.", artifactClassLabel(classFilter))
		}
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateWaiting, Title: "Evidence workspace", Message: message, Reason: "No evidence is open yet.", NextAction: "Choose an artifact, then continue to Audit for final checks or receipts.", ShortcutCue: "enter", RouteCue: "Artifacts → Audit"})
	}
	a := head.Artifact
	return renderStateCardSpec(stateCardSpec{State: routeLoadStateReady, Title: "Evidence workspace", Message: fmt.Sprintf("%s is ready to inspect.", artifactDisplayLabel(a)), Reason: fmt.Sprintf("%s from run %s with %s ready in the inspector.", artifactClassLabel(a.Reference.DataClass), valueOrNA(a.RunID), artifactPreviewIdentity(a.Reference.SizeBytes, a.Reference.ContentType)), NextAction: "Review the evidence here, then open Audit to confirm final checks, export, or receipts.", ShortcutCue: "enter / m", RouteCue: "Audit"})
}

func renderArtifactEvidenceTrail(head *brokerapi.LocalArtifactHeadResponse) string {
	if head == nil {
		return "Evidence path: run result -> artifact evidence -> Audit review and receipts."
	}
	a := head.Artifact
	return fmt.Sprintf("Evidence path: run %s -> %s -> Audit review and receipts.", valueOrNA(a.RunID), artifactDisplayLabel(a))
}

func artifactDisplayLabel(item brokerapi.ArtifactSummary) string {
	class := artifactClassLabel(item.Reference.DataClass)
	run := strings.TrimSpace(item.RunID)
	if class != "" && run != "" {
		return fmt.Sprintf("%s for %s", class, run)
	}
	if class != "" {
		return class + " artifact"
	}
	return shortIdentity(item.Reference.Digest)
}

func renderArtifactClassFilterLine(classFilter string) string {
	classFilter = strings.TrimSpace(classFilter)
	if classFilter == "" {
		return "Filter: all artifact classes"
	}
	return "Filter: " + artifactClassLabel(classFilter)
}

func artifactClassLabel(dataClass any) string {
	value := strings.TrimSpace(fmt.Sprintf("%v", dataClass))
	if value == "" {
		return ""
	}
	return humanizeExecutionToken(value)
}

func artifactSourceRunLabel(runID string) string {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ""
	}
	return "source run " + runID
}

func artifactPreviewIdentity(sizeBytes int64, contentType string) string {
	parts := []string{}
	if sizeBytes > 0 {
		parts = append(parts, fmt.Sprintf("%d bytes", sizeBytes))
	}
	if label := artifactContentTypeLabel(contentType); label != "" {
		parts = append(parts, label)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ") + " preview"
}

func artifactContentTypeLabel(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return ""
	}
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	switch contentType {
	case "text/plain":
		return "plain text"
	case "text/markdown":
		return "markdown"
	case "application/json":
		return "json"
	default:
		return contentType
	}
}
