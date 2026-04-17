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

func renderSessionInspector(detail *brokerapi.SessionDetail, presentation contentPresentationMode) string {
	if detail == nil {
		return "  Select a session and press enter to load transcript."
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
	return renderInspectorShell(inspectorShellSpec{
		Title:   "Session inspector",
		Summary: activeSessionSummaryLine(detail),
		Identity: fmt.Sprintf("session=%s workspace=%s", summary.Identity.SessionID,
			valueOrNA(summary.Identity.WorkspaceID)),
		Status:     fmt.Sprintf("status=%s turn_count=%d", valueOrNA(summary.Status), summary.TurnCount),
		Badges:     []string{stateBadgeWithLabel("status", summary.Status), appTheme.InspectorHint.Render("linked refs + ordered transcript")},
		ModeTabs:   []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode: string(presentation),
		References: []inspectorReference{
			{Label: "runs", Items: detail.LinkedRunIDs},
			{Label: "approvals", Items: detail.LinkedApprovalIDs},
			{Label: "artifacts", Items: detail.LinkedArtifactDigests},
			{Label: "audit", Items: detail.LinkedAuditRecordDigests},
		},
		LocalActions:   []string{"jump:runs", "jump:approvals", "jump:artifacts", "jump:audit", "copy:session_id"},
		CopyActions:    chatRouteCopyActions(detail),
		ContentKind:    contentKind,
		ContentLabel:   "transcript",
		Content:        transcript,
		ViewportWidth:  96,
		ViewportHeight: 12,
	})
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
