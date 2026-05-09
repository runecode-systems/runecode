package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func selectedSessionIndex(sessions []brokerapi.SessionSummary, activeID string) int {
	if len(sessions) == 0 {
		return 0
	}
	for i, s := range sessions {
		if s.Identity.SessionID == activeID {
			return i
		}
	}
	return 0
}

func renderLinkedReferenceLine(prefix string, refs []string) string {
	if len(refs) == 0 {
		return prefix + ": none"
	}
	return fmt.Sprintf("%s: %s", prefix, strings.Join(refs, ", "))
}

func renderSessionDirectoryItems(sessions []brokerapi.SessionSummary) []string {
	items := make([]string, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, renderSessionDirectoryItem(s))
	}
	return items
}

func renderSessionDirectoryItem(summary brokerapi.SessionSummary) string {
	parts := []string{summary.Identity.SessionID, postureBadge(summary.Status)}
	if cue := shortSessionWorkCue(summary); cue != "" {
		parts = append(parts, cue)
	}
	if followUp := sessionDirectoryFollowUpCue(summary); followUp != "" {
		parts = append(parts, muted("• "+followUp))
	}
	return strings.Join(parts, " ")
}

func shortSessionWorkCue(summary brokerapi.SessionSummary) string {
	preview := truncateText(summary.LastActivityPreview, 56)
	if preview != "" {
		return fmt.Sprintf("— %s", preview)
	}
	return fmt.Sprintf("— %s work in progress", sessionHighLevelCue(summary))
}

func sessionDirectoryFollowUpCue(summary brokerapi.SessionSummary) string {
	cues := make([]string, 0, 3)
	if summary.LinkedApprovalCount > 0 {
		cues = append(cues, countNoun(summary.LinkedApprovalCount, "approval waiting", "approvals waiting"))
	}
	if summary.LinkedRunCount > 0 {
		cues = append(cues, countNoun(summary.LinkedRunCount, "linked run", "linked runs"))
	}
	if summary.LinkedArtifactCount > 0 {
		cues = append(cues, countNoun(summary.LinkedArtifactCount, "artifact", "artifacts"))
	}
	if len(cues) == 0 && summary.TurnCount > 0 {
		cues = append(cues, countNoun(summary.TurnCount, "turn so far", "turns so far"))
	}
	return strings.Join(cues, " • ")
}

func activeSessionSummaryLine(detail *brokerapi.SessionDetail) string {
	if detail == nil {
		return "No session selected."
	}
	s := detail.Summary
	exec := chatVisibleExecution(detail)
	statusParts := []string{fmt.Sprintf("Canonical session %s in workspace %s is %s.", s.Identity.SessionID, valueOrNA(s.Identity.WorkspaceID), sessionHighLevelCue(s))}
	if exec != nil {
		statusParts = append(statusParts, fmt.Sprintf("Current workflow state: %s", humanizeExecutionToken(exec.ExecutionState)))
		if wait := strings.TrimSpace(exec.WaitState); wait != "" {
			statusParts = append(statusParts, fmt.Sprintf("waiting on %s", humanizeExecutionToken(wait)))
		}
	}
	if preview := truncateText(s.LastActivityPreview, 72); preview != "" {
		statusParts = append(statusParts, fmt.Sprintf("last activity %q", preview))
	}
	return strings.Join(statusParts, " • ")
}

func renderActiveSessionSummaryBlock(detail *brokerapi.SessionDetail, sessionCount int) string {
	if detail == nil {
		return muted("No active session selected. Pick a session from the directory to resume the work loop.")
	}
	lines := []string{
		fmt.Sprintf("Active session %s · workspace %s", detail.Summary.Identity.SessionID, valueOrNA(detail.Summary.Identity.WorkspaceID)),
	}
	if cue := shortSessionWorkCue(detail.Summary); cue != "" {
		lines = append(lines, muted(strings.TrimPrefix(cue, "— ")))
	}
	meta := []string{}
	if sessionCount > 0 {
		meta = append(meta, countNoun(sessionCount, "session in directory", "sessions in directory"))
	}
	if followUp := sessionDirectoryFollowUpCue(detail.Summary); followUp != "" {
		meta = append(meta, followUp)
	}
	if len(meta) > 0 {
		lines = append(lines, muted(strings.Join(meta, " • ")))
	}
	return compactLines(lines...)
}

func composeDraftStatusLine(draft string) string {
	draft = strings.TrimSpace(draft)
	if draft == "" {
		return "Composer is active with an empty draft."
	}
	lines := strings.Count(draft, "\n") + 1
	return fmt.Sprintf("Composer has a local draft (%d chars across %d line(s)).", len([]rune(draft)), lines)
}

func renderTranscriptTurns(turns []brokerapi.SessionTranscriptTurn) string {
	orderedTurns := sortTranscriptTurns(turns)
	if len(orderedTurns) == 0 {
		return "    - no transcript turns"
	}
	var b strings.Builder
	for _, turn := range orderedTurns {
		b.WriteString(fmt.Sprintf("    - turn[%d] %s status=%s\n", turn.TurnIndex, turn.TurnID, turn.Status))
		for _, msg := range sortTranscriptMessages(turn.Messages) {
			b.WriteString(fmt.Sprintf("      • msg[%d] %s: %s\n", msg.MessageIndex, msg.Role, sanitizeUIText(msg.ContentText)))
			related := flattenRelatedLinks(msg.RelatedLinks)
			if related != "" {
				b.WriteString(fmt.Sprintf("        links: %s\n", related))
			}
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func flattenRelatedLinks(links brokerapi.SessionTranscriptLinks) string {
	parts := make([]string, 0, 4)
	if len(links.RunIDs) > 0 {
		parts = append(parts, "runs="+strings.Join(links.RunIDs, ","))
	}
	if len(links.ApprovalIDs) > 0 {
		parts = append(parts, "approvals="+strings.Join(links.ApprovalIDs, ","))
	}
	if len(links.ArtifactDigests) > 0 {
		parts = append(parts, "artifacts="+strings.Join(links.ArtifactDigests, ","))
	}
	if len(links.AuditRecordDigests) > 0 {
		parts = append(parts, "audit="+strings.Join(links.AuditRecordDigests, ","))
	}
	return strings.Join(parts, " ")
}

func renderComposer(on bool, draft string, textareaView string) string {
	if !on {
		return muted("Composer closed. Press c to continue the conversation.")
	}
	if strings.TrimSpace(textareaView) != "" {
		return compactLines("Compose draft:", textareaView)
	}
	return fmt.Sprintf("Compose draft: %q", redactSecrets(draft))
}

func renderTranscriptRaw(turns []brokerapi.SessionTranscriptTurn) string {
	orderedTurns := sortTranscriptTurns(turns)
	if len(orderedTurns) == 0 {
		return "    - no transcript turns"
	}
	var b strings.Builder
	for _, turn := range orderedTurns {
		b.WriteString(fmt.Sprintf("    turn_id=%s index=%d status=%s\n", turn.TurnID, turn.TurnIndex, turn.Status))
		for _, msg := range turn.Messages {
			b.WriteString(fmt.Sprintf("      msg_id=%s idx=%d role=%s text=%q\n", msg.MessageID, msg.MessageIndex, msg.Role, redactSecrets(msg.ContentText)))
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderTranscriptStructured(turns []brokerapi.SessionTranscriptTurn) string {
	if len(turns) == 0 {
		return "    - no transcript turns"
	}
	msgCount := 0
	for _, turn := range turns {
		msgCount += len(turn.Messages)
	}
	return fmt.Sprintf("    - turn_count=%d message_count=%d", len(turns), msgCount)
}

func sortTranscriptTurns(turns []brokerapi.SessionTranscriptTurn) []brokerapi.SessionTranscriptTurn {
	orderedTurns := append([]brokerapi.SessionTranscriptTurn(nil), turns...)
	sort.Slice(orderedTurns, func(i, j int) bool {
		if orderedTurns[i].TurnIndex == orderedTurns[j].TurnIndex {
			return orderedTurns[i].TurnID < orderedTurns[j].TurnID
		}
		return orderedTurns[i].TurnIndex < orderedTurns[j].TurnIndex
	})
	return orderedTurns
}

func sortTranscriptMessages(messages []brokerapi.SessionTranscriptMessage) []brokerapi.SessionTranscriptMessage {
	orderedMessages := append([]brokerapi.SessionTranscriptMessage(nil), messages...)
	sort.Slice(orderedMessages, func(i, j int) bool {
		if orderedMessages[i].MessageIndex == orderedMessages[j].MessageIndex {
			return orderedMessages[i].MessageID < orderedMessages[j].MessageID
		}
		return orderedMessages[i].MessageIndex < orderedMessages[j].MessageIndex
	})
	return orderedMessages
}
