package brokerapi

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type sessionExecutionLinkProjection struct {
	runIDs             []string
	approvalIDs        []string
	artifactDigests    []string
	auditRecordDigests []string
}

func sessionExecutionLinksFromSessionState(session artifacts.SessionDurableState) sessionExecutionLinkProjection {
	collector := newSessionExecutionLinkCollector(session)
	collector.collectTranscriptLinks(session.TranscriptTurns)
	return collector.projection()
}

type sessionExecutionLinkCollector struct {
	runs      []string
	approvals map[string]struct{}
	artifacts map[string]struct{}
	audit     map[string]struct{}
}

func newSessionExecutionLinkCollector(session artifacts.SessionDurableState) sessionExecutionLinkCollector {
	runs := append([]string{}, session.LinkedRunIDs...)
	if created := strings.TrimSpace(session.CreatedByRunID); created != "" {
		runs = append(runs, created)
	}
	return sessionExecutionLinkCollector{
		runs:      runs,
		approvals: map[string]struct{}{},
		artifacts: map[string]struct{}{},
		audit:     map[string]struct{}{},
	}
}

func (c *sessionExecutionLinkCollector) collectTranscriptLinks(turns []artifacts.SessionTranscriptTurnDurableState) {
	for _, turn := range turns {
		for _, message := range turn.Messages {
			collectTrimmedStrings(c.approvals, message.RelatedLinks.ApprovalIDs)
			collectTrimmedStrings(c.artifacts, message.RelatedLinks.ArtifactDigests)
			collectTrimmedStrings(c.audit, message.RelatedLinks.AuditRecordDigests)
		}
	}
}

func (c sessionExecutionLinkCollector) projection() sessionExecutionLinkProjection {
	return sessionExecutionLinkProjection{
		runIDs:             nonEmptyStringsUniqueSorted(c.runs),
		approvalIDs:        mapKeysSorted(c.approvals),
		artifactDigests:    mapKeysSorted(c.artifacts),
		auditRecordDigests: mapKeysSorted(c.audit),
	}
}

func collectTrimmedStrings(dst map[string]struct{}, values []string) {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			dst[trimmed] = struct{}{}
		}
	}
}

func mapKeysSorted(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return nonEmptyStringsUniqueSorted(out)
}

func nonEmptyStringsUniqueSorted(values []string) []string {
	set := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, seen := set[trimmed]; seen {
			continue
		}
		set[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) > 1 {
		sort.Strings(out)
	}
	return out
}
