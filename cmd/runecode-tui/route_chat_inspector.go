package main

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func renderSessionList(sessions []brokerapi.SessionSummary, selected int) string {
	if len(sessions) == 0 {
		return "  - no sessions"
	}
	line := ""
	for i, s := range sessions {
		marker := " "
		if i == selected {
			marker = ">"
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s %s turns=%d", marker, s.Identity.SessionID, stateBadgeWithLabel("status", s.Status), s.TurnCount)) + "\n"
	}
	return line
}

func renderSessionInspector(detail *brokerapi.SessionDetail, presentation contentPresentationMode, document *longFormDocumentState) string {
	if detail == nil {
		return "  Select a session and press enter to load transcript."
	}
	if document == nil {
		fallback := newLongFormDocumentState()
		document = &fallback
	}
	presentation = normalizePresentationMode(presentation)
	transcript := renderTranscriptTurns(detail.TranscriptTurns)
	contentKind := inspectorContentTranscript
	if presentation == presentationRaw {
		transcript = renderTranscriptRaw(detail.TranscriptTurns)
		contentKind = inspectorContentRaw
	}
	if presentation == presentationStructured {
		transcript = renderTranscriptStructured(detail.TranscriptTurns)
		contentKind = inspectorContentStructured
	}
	summary := detail.Summary
	ref := workbenchObjectRef{Kind: "session", ID: strings.TrimSpace(summary.Identity.SessionID), WorkspaceID: strings.TrimSpace(summary.Identity.WorkspaceID), SessionID: strings.TrimSpace(summary.Identity.SessionID)}
	document.SetDocument(ref, contentKind, "transcript", transcript)
	references := chatInspectorReferences(detail)
	localActions := chatInspectorLocalActions()
	return renderInspectorShell(inspectorShellSpec{
		Title:   "Session inspector",
		Summary: activeSessionSummaryLine(detail),
		Identity: fmt.Sprintf("session=%s workspace=%s", summary.Identity.SessionID,
			valueOrNA(summary.Identity.WorkspaceID)),
		Status:       fmt.Sprintf("status=%s turn_count=%d", valueOrNA(summary.Status), summary.TurnCount),
		Badges:       []string{stateBadgeWithLabel("status", summary.Status), appTheme.InspectorHint.Render("linked refs + ordered transcript")},
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		References:   references,
		LocalActions: localActions,
		CopyActions:  chatRouteCopyActions(detail),
		Document:     document,
	})
}

func chatInspectorReferences(detail *brokerapi.SessionDetail) []inspectorReference {
	if detail == nil {
		return nil
	}
	return []inspectorReference{
		{Label: "runs", Items: mapReferenceIDs(detail.LinkedRunIDs, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "run", RouteID: routeRuns, RunID: id}}
		})},
		{Label: "approvals", Items: mapReferenceIDs(detail.LinkedApprovalIDs, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "approval", RouteID: routeApprovals, ApprovalID: id}}
		})},
		{Label: "artifacts", Items: mapReferenceIDs(detail.LinkedArtifactDigests, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "artifact", RouteID: routeArtifacts, Digest: id}}
		})},
		{Label: "audit", Items: mapReferenceIDs(detail.LinkedAuditRecordDigests, func(id string) paletteActionMsg {
			return paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "audit", RouteID: routeAudit, Digest: id}}
		})},
	}
}

func chatInspectorLocalActions() []routeActionItem {
	return []routeActionItem{
		{Label: "jump:runs", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeRuns}}},
		{Label: "jump:approvals", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeApprovals}}},
		{Label: "jump:artifacts", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeArtifacts}}},
		{Label: "jump:audit", Action: paletteActionMsg{Verb: verbJump, Target: paletteTarget{Kind: "route", RouteID: routeAudit}}},
		{Label: "copy:session_id"},
	}
}

func chatInspectorReferenceActions(detail *brokerapi.SessionDetail) []routeActionItem {
	refs := chatInspectorReferences(detail)
	out := make([]routeActionItem, 0, 12)
	for _, ref := range refs {
		for _, item := range ref.Items {
			if strings.TrimSpace(item.Label) == "" {
				continue
			}
			out = append(out, routeActionItem{Label: fmt.Sprintf("%s:%s", ref.Label, item.Label), Action: item.Action})
		}
	}
	return out
}

func chatRouteCopyActions(detail *brokerapi.SessionDetail) []routeCopyAction {
	if detail == nil {
		return nil
	}
	summary := detail.Summary
	actions := []routeCopyAction{
		{ID: "session_id", Label: "session id", Text: summary.Identity.SessionID},
		{ID: "workspace_id", Label: "workspace id", Text: summary.Identity.WorkspaceID},
		{ID: "transcript_excerpt", Label: "transcript excerpt", Text: transcriptExcerpt(detail.TranscriptTurns, 6)},
	}
	refs := linkedReferencesText(detail)
	if strings.TrimSpace(refs) != "" {
		actions = append(actions, routeCopyAction{ID: "linked_references", Label: "linked references", Text: refs})
	}
	return compactCopyActions(actions)
}

func linkedReferencesText(detail *brokerapi.SessionDetail) string {
	if detail == nil {
		return ""
	}
	lines := []string{
		renderLinkedReferenceLine("runs", detail.LinkedRunIDs),
		renderLinkedReferenceLine("approvals", detail.LinkedApprovalIDs),
		renderLinkedReferenceLine("artifacts", detail.LinkedArtifactDigests),
		renderLinkedReferenceLine("audit", detail.LinkedAuditRecordDigests),
	}
	return strings.Join(lines, "\n")
}

func transcriptExcerpt(turns []brokerapi.SessionTranscriptTurn, maxLines int) string {
	rendered := strings.TrimSpace(renderTranscriptTurns(turns))
	if rendered == "" {
		return ""
	}
	if maxLines <= 0 {
		maxLines = 6
	}
	lines := strings.Split(rendered, "\n")
	if len(lines) <= maxLines {
		return rendered
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

func compactCopyActions(actions []routeCopyAction) []routeCopyAction {
	out := make([]routeCopyAction, 0, len(actions))
	for _, action := range actions {
		if strings.TrimSpace(action.Text) == "" {
			continue
		}
		out = append(out, action)
	}
	return out
}
