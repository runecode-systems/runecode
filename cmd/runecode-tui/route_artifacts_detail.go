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
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s class=%s bytes=%d run=%s", marker, item.Reference.Digest, item.Reference.DataClass, item.Reference.SizeBytes, item.RunID)) + "\n"
	}
	return line
}

func renderArtifactDirectoryItems(items []brokerapi.ArtifactSummary) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s %s class=%s bytes=%d %s", artifactDisplayLabel(item), shortIdentity(item.Reference.Digest), item.Reference.DataClass, item.Reference.SizeBytes, artifactRunLabel(item.RunID)))
	}
	return out
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
		fmt.Sprintf("Evidence trail: workflow result/run %s -> artifact %s -> audit records -> verification posture -> export/offline verification or anchoring where available", valueOrNA(a.RunID), artifactDisplayLabel(a)),
		fmt.Sprintf("Primary digest display: %s (copy raw digest below)", shortIdentity(a.Reference.Digest)),
		fmt.Sprintf("Typed detail mode: %s (metadata remains control-plane truth)", mode),
		fmt.Sprintf("Presentation mode: %s", presentation),
		fmt.Sprintf("Provenance receipt: %s", a.Reference.ProvenanceReceiptHash),
		"Inspectable content is supplemental evidence, not authoritative run/approval truth.",
		contentView,
	))
	return renderInspectorShell(inspectorShellSpec{
		Title:    "Artifact inspector",
		Summary:  fmt.Sprintf("artifact=%s class=%s bytes=%d", artifactDisplayLabel(a), a.Reference.DataClass, a.Reference.SizeBytes),
		Identity: fmt.Sprintf("artifact=%s digest=%s", artifactDisplayLabel(a), shortIdentity(a.Reference.Digest)),
		Status:   fmt.Sprintf("evidence_trail=run:%s -> artifact:%s -> audit -> verification posture", valueOrNA(a.RunID), shortIdentity(a.Reference.Digest)),
		Badges:   []string{stateBadgeWithLabel("class", fmt.Sprintf("%v", a.Reference.DataClass)), appTheme.InspectorHint.Render("typed metadata first")},
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
		fmt.Sprintf("Evidence trail: workflow result/run %s -> artifact %s -> audit records -> verification posture -> export/offline verification or anchoring where available", valueOrNA(a.RunID), artifactDisplayLabel(a)),
		fmt.Sprintf("Primary digest display: %s (copy raw digest below)", shortIdentity(a.Reference.Digest)),
		fmt.Sprintf("Typed detail mode: %s (metadata remains control-plane truth)", mode),
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
		message := "Select an artifact to inspect the evidence trail from workflow result to audit verification."
		if strings.TrimSpace(classFilter) != "" {
			message = fmt.Sprintf("Select an artifact in class %q to inspect the evidence trail from workflow result to audit verification.", classFilter)
		}
		return renderStateCardSpec(stateCardSpec{State: routeLoadStateWaiting, Title: "Evidence trail", Message: message, Reason: "No artifact detail is loaded yet.", NextAction: "Choose an artifact, then continue to Audit for verification posture and anchoring.", ShortcutCue: "enter", RouteCue: "Artifacts → Audit"})
	}
	a := head.Artifact
	return renderStateCardSpec(stateCardSpec{State: routeLoadStateReady, Title: "Evidence trail", Message: fmt.Sprintf("%s is ready for inspection.", artifactDisplayLabel(a)), Reason: fmt.Sprintf("This artifact belongs to run %s and carries broker metadata plus copyable raw digests.", valueOrNA(a.RunID)), NextAction: "Inspect content here, then open Audit to review verification posture, export, or anchoring actions.", ShortcutCue: "enter / m", RouteCue: "Audit"})
}

func renderArtifactEvidenceTrail(head *brokerapi.LocalArtifactHeadResponse) string {
	if head == nil {
		return "Evidence path: workflow result -> artifact -> audit record -> verification posture -> export/offline verification or anchoring where available"
	}
	a := head.Artifact
	return fmt.Sprintf("Evidence path: workflow result/run %s -> artifact %s (%s) -> Audit -> verification posture -> export/offline verification or anchoring where available", valueOrNA(a.RunID), artifactDisplayLabel(a), shortIdentity(a.Reference.Digest))
}

func artifactDisplayLabel(item brokerapi.ArtifactSummary) string {
	class := strings.TrimSpace(fmt.Sprintf("%v", item.Reference.DataClass))
	run := strings.TrimSpace(item.RunID)
	if class != "" && run != "" {
		return fmt.Sprintf("%s for %s", class, run)
	}
	if class != "" {
		return class + " artifact"
	}
	return shortIdentity(item.Reference.Digest)
}

func artifactRunLabel(runID string) string {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ""
	}
	return "run=" + runID
}
