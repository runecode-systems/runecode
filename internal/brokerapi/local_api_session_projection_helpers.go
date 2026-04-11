package brokerapi

import (
	"fmt"
	"sort"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type sessionSummaryStats struct {
	created           time.Time
	updated           time.Time
	runs              map[string]struct{}
	artifactsByDigest map[string]struct{}
}

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

func buildSessionSummary(sessionID string, records []artifacts.ArtifactRecord, linkedApprovalCount, linkedAuditEventCount int) SessionSummary {
	stats := collectSessionSummaryStats(records)
	updatedAt := stats.updated.Format(time.RFC3339)
	return SessionSummary{
		SchemaID:      "runecode.protocol.v0.SessionSummary",
		SchemaVersion: "0.1.0",
		Identity: SessionIdentity{
			SchemaID:       "runecode.protocol.v0.SessionIdentity",
			SchemaVersion:  "0.1.0",
			SessionID:      sessionID,
			WorkspaceID:    "workspace-local",
			CreatedAt:      stats.created.Format(time.RFC3339),
			CreatedByRunID: firstSortedStringKey(stats.runs),
		},
		UpdatedAt:             updatedAt,
		Status:                "active",
		LastActivityAt:        updatedAt,
		LastActivityKind:      "run_progress",
		LastActivityPreview:   sessionSummaryPreview(records),
		TurnCount:             sessionSummaryTurnCount(records, stats.runs),
		LinkedRunCount:        len(stats.runs),
		LinkedApprovalCount:   linkedApprovalCount,
		LinkedArtifactCount:   len(stats.artifactsByDigest),
		LinkedAuditEventCount: linkedAuditEventCount,
		HasIncompleteTurn:     false,
	}
}

func collectSessionSummaryStats(records []artifacts.ArtifactRecord) sessionSummaryStats {
	created := time.Unix(0, 0).UTC()
	updated := created
	if len(records) > 0 {
		created = records[0].CreatedAt.UTC()
		updated = created
	}
	stats := sessionSummaryStats{
		created:           created,
		updated:           updated,
		runs:              map[string]struct{}{},
		artifactsByDigest: map[string]struct{}{},
	}
	for _, rec := range records {
		stats.created = minSessionTime(stats.created, rec.CreatedAt.UTC())
		stats.updated = maxSessionTime(stats.updated, rec.CreatedAt.UTC())
		if rec.RunID != "" {
			stats.runs[rec.RunID] = struct{}{}
		}
		if rec.Reference.Digest != "" {
			stats.artifactsByDigest[rec.Reference.Digest] = struct{}{}
		}
	}
	return stats
}

func minSessionTime(current, candidate time.Time) time.Time {
	if candidate.Before(current) {
		return candidate
	}
	return current
}

func maxSessionTime(current, candidate time.Time) time.Time {
	if candidate.After(current) {
		return candidate
	}
	return current
}

func sessionSummaryPreview(records []artifacts.ArtifactRecord) string {
	if len(records) == 0 {
		return ""
	}
	return "session linked to run activity"
}

func sessionSummaryTurnCount(records []artifacts.ArtifactRecord, runs map[string]struct{}) int {
	if len(runs) > 0 {
		return len(runs)
	}
	if len(records) > 0 {
		return 1
	}
	return 0
}

func sortedStringKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func firstSortedStringKey(values map[string]struct{}) string {
	keys := sortedStringKeys(values)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
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

func buildSessionSummaries(byID map[string][]artifacts.ArtifactRecord, approvalsBySession, auditBySession map[string]map[string]struct{}) []SessionSummary {
	out := make([]SessionSummary, 0, len(byID))
	for sessionID, records := range byID {
		out = append(out, buildSessionSummary(sessionID, records, len(approvalsBySession[sessionID]), len(auditBySession[sessionID])))
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
	return SessionDetail{
		SchemaID:                 "runecode.protocol.v0.SessionDetail",
		SchemaVersion:            "0.1.0",
		Summary:                  summary,
		TranscriptTurns:          buildSessionTranscriptTurns(summary.Identity.SessionID, summary, runs, approvals, artifactsByDigest, auditRecordDigests),
		LinkedRunIDs:             boundedSortedKeys(runs, 256),
		LinkedApprovalIDs:        boundedSortedKeys(approvals, 512),
		LinkedArtifactDigests:    boundedSortedKeys(artifactsByDigest, 1024),
		LinkedAuditRecordDigests: boundedSortedKeys(auditRecordDigests, 1024),
	}
}
