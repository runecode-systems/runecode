package brokerapi

import (
	"fmt"
	"sort"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func buildSessionTranscriptTurns(sessionID string, summary SessionSummary, runs, approvals, artifactsByDigest, auditRecordDigests map[string]struct{}) []SessionTranscriptTurn {
	turnLimit := cappedSessionTurnCount(summary.TurnCount)
	if turnLimit == 0 {
		return []SessionTranscriptTurn{}
	}
	links := SessionTranscriptLinks{
		SchemaID:           "runecode.protocol.v0.SessionTranscriptLinks",
		SchemaVersion:      "0.1.0",
		RunIDs:             boundedSortedKeys(runs, 256),
		ApprovalIDs:        boundedSortedKeys(approvals, 512),
		ArtifactDigests:    boundedSortedKeys(artifactsByDigest, 1024),
		AuditRecordDigests: boundedSortedKeys(auditRecordDigests, 1024),
	}
	turns := make([]SessionTranscriptTurn, 0, turnLimit)
	for i := 0; i < turnLimit; i++ {
		turns = append(turns, buildProjectedSessionTranscriptTurn(sessionID, i+1, summary, links))
	}
	return turns
}

func buildSessionTranscriptTurnsFromDurable(turns []artifacts.SessionTranscriptTurnDurableState) []SessionTranscriptTurn {
	if len(turns) == 0 {
		return []SessionTranscriptTurn{}
	}
	out := make([]SessionTranscriptTurn, 0, len(turns))
	for _, turn := range turns {
		out = append(out, buildSessionTranscriptTurnFromDurable(turn))
	}
	return out
}

func buildSessionTranscriptTurnFromDurable(turn artifacts.SessionTranscriptTurnDurableState) SessionTranscriptTurn {
	completedAt := ""
	if turn.CompletedAt != nil {
		completedAt = turn.CompletedAt.UTC().Format(time.RFC3339)
	}
	return SessionTranscriptTurn{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptTurn",
		SchemaVersion: "0.1.0",
		TurnID:        turn.TurnID,
		SessionID:     turn.SessionID,
		TurnIndex:     turn.TurnIndex,
		StartedAt:     turn.StartedAt.UTC().Format(time.RFC3339),
		CompletedAt:   completedAt,
		Status:        turn.Status,
		Messages:      buildSessionTranscriptMessagesFromDurable(turn.Messages),
	}
}

func buildSessionTranscriptMessagesFromDurable(messages []artifacts.SessionTranscriptMessageDurableState) []SessionTranscriptMessage {
	out := make([]SessionTranscriptMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, buildSessionTranscriptMessageFromDurable(message))
	}
	return out
}

func buildSessionTranscriptMessageFromDurable(message artifacts.SessionTranscriptMessageDurableState) SessionTranscriptMessage {
	return SessionTranscriptMessage{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptMessage",
		SchemaVersion: "0.1.0",
		MessageID:     message.MessageID,
		TurnID:        message.TurnID,
		SessionID:     message.SessionID,
		MessageIndex:  message.MessageIndex,
		Role:          message.Role,
		CreatedAt:     message.CreatedAt.UTC().Format(time.RFC3339),
		ContentText:   message.ContentText,
		RelatedLinks:  boundedSessionTranscriptLinks(message.RelatedLinks),
	}
}

func boundedSessionTranscriptLinks(links artifacts.SessionTranscriptLinksDurableState) SessionTranscriptLinks {
	return SessionTranscriptLinks{
		SchemaID:           "runecode.protocol.v0.SessionTranscriptLinks",
		SchemaVersion:      "0.1.0",
		RunIDs:             boundedStrings(append([]string{}, links.RunIDs...), 256),
		ApprovalIDs:        boundedStrings(append([]string{}, links.ApprovalIDs...), 512),
		ArtifactDigests:    boundedStrings(append([]string{}, links.ArtifactDigests...), 1024),
		AuditRecordDigests: boundedStrings(append([]string{}, links.AuditRecordDigests...), 1024),
	}
}

func buildProjectedSessionTranscriptTurn(sessionID string, turnIndex int, summary SessionSummary, links SessionTranscriptLinks) SessionTranscriptTurn {
	turnID := fmt.Sprintf("%s.turn.%06d", sessionID, turnIndex)
	messageID := fmt.Sprintf("%s.msg.%06d", turnID, 1)
	return SessionTranscriptTurn{
		SchemaID:      "runecode.protocol.v0.SessionTranscriptTurn",
		SchemaVersion: "0.1.0",
		TurnID:        turnID,
		SessionID:     sessionID,
		TurnIndex:     turnIndex,
		StartedAt:     summary.UpdatedAt,
		CompletedAt:   summary.UpdatedAt,
		Status:        "completed",
		Messages: []SessionTranscriptMessage{{
			SchemaID:      "runecode.protocol.v0.SessionTranscriptMessage",
			SchemaVersion: "0.1.0",
			MessageID:     messageID,
			TurnID:        turnID,
			SessionID:     sessionID,
			MessageIndex:  1,
			Role:          "system",
			CreatedAt:     summary.UpdatedAt,
			ContentText:   summary.LastActivityPreview,
			RelatedLinks:  links,
		}},
	}
}

func cappedSessionTurnCount(turnCount int) int {
	if turnCount <= 0 {
		return 0
	}
	if turnCount > 2048 {
		return 2048
	}
	return turnCount
}

func buildSessionSummary(state artifacts.SessionDurableState, linkedRunCount, linkedApprovalCount, linkedArtifactCount, linkedAuditEventCount int) SessionSummary {
	updatedAt := state.UpdatedAt.UTC().Format(time.RFC3339)
	return SessionSummary{
		SchemaID:      "runecode.protocol.v0.SessionSummary",
		SchemaVersion: "0.1.0",
		Identity: SessionIdentity{
			SchemaID:       "runecode.protocol.v0.SessionIdentity",
			SchemaVersion:  "0.1.0",
			SessionID:      state.SessionID,
			WorkspaceID:    state.WorkspaceID,
			CreatedAt:      state.CreatedAt.UTC().Format(time.RFC3339),
			CreatedByRunID: state.CreatedByRunID,
		},
		UpdatedAt:             updatedAt,
		Status:                state.Status,
		WorkPosture:           state.WorkPosture,
		WorkPostureReasonCode: state.WorkPostureReason,
		LastActivityAt:        state.LastActivityAt.UTC().Format(time.RFC3339),
		LastActivityKind:      state.LastActivityKind,
		LastActivityPreview:   state.LastActivityPreview,
		TurnCount:             state.TurnCount,
		LinkedRunCount:        linkedRunCount,
		LinkedApprovalCount:   linkedApprovalCount,
		LinkedArtifactCount:   linkedArtifactCount,
		LinkedAuditEventCount: linkedAuditEventCount,
		HasIncompleteTurn:     state.HasIncompleteTurn,
	}
}

func sortedStringKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func boundedStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func boundedSortedKeys(values map[string]struct{}, limit int) []string {
	return boundedStrings(sortedStringKeys(values), limit)
}

func buildSessionSummaries(states []artifacts.SessionDurableState, runsBySession, approvalsBySession, artifactsBySession, auditBySession map[string]map[string]struct{}) []SessionSummary {
	out := make([]SessionSummary, 0, len(states))
	for _, state := range states {
		out = append(out, buildSessionSummary(state, len(runsBySession[state.SessionID]), len(approvalsBySession[state.SessionID]), len(artifactsBySession[state.SessionID]), len(auditBySession[state.SessionID])))
	}
	return out
}

func sortSessionSummaries(items []SessionSummary, order string) {
	sort.Slice(items, func(i, j int) bool {
		if order == "updated_at_asc" {
			if items[i].UpdatedAt == items[j].UpdatedAt {
				return items[i].Identity.SessionID < items[j].Identity.SessionID
			}
			return items[i].UpdatedAt < items[j].UpdatedAt
		}
		if items[i].UpdatedAt == items[j].UpdatedAt {
			return items[i].Identity.SessionID < items[j].Identity.SessionID
		}
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
}

func buildSessionDetail(summary SessionSummary, runs, approvals, artifactsByDigest, auditRecordDigests map[string]struct{}) SessionDetail {
	return buildSessionDetailFromState(summary, nil, runs, approvals, artifactsByDigest, auditRecordDigests)
}

func buildSessionDetailFromState(summary SessionSummary, transcriptTurns []artifacts.SessionTranscriptTurnDurableState, runs, approvals, artifactsByDigest, auditRecordDigests map[string]struct{}) SessionDetail {
	projectedTurns := buildSessionTranscriptTurnsFromDurable(transcriptTurns)
	if len(projectedTurns) == 0 {
		projectedTurns = buildSessionTranscriptTurns(summary.Identity.SessionID, summary, runs, approvals, artifactsByDigest, auditRecordDigests)
	}
	return newSessionDetail(summary, projectedTurns, runs, approvals, artifactsByDigest, auditRecordDigests)
}
