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

func renderTranscriptTurns(turns []brokerapi.SessionTranscriptTurn) string {
	orderedTurns := sortTranscriptTurns(turns)
	if len(orderedTurns) == 0 {
		return "    - no transcript turns"
	}
	var b strings.Builder
	for _, turn := range orderedTurns {
		b.WriteString(fmt.Sprintf("    - turn[%d] %s status=%s\n", turn.TurnIndex, turn.TurnID, turn.Status))
		for _, msg := range sortTranscriptMessages(turn.Messages) {
			b.WriteString(fmt.Sprintf("      • msg[%d] %s: %s\n", msg.MessageIndex, msg.Role, redactSecrets(msg.ContentText)))
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

func renderComposer(on bool, draft string) string {
	if !on {
		return "Composer: press c to compose and send to active session"
	}
	return fmt.Sprintf("Compose draft: %q", redactSecrets(draft))
}

func composerState(on bool) string {
	if on {
		return "active"
	}
	return "idle"
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
