package brokerapi

import (
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func uniqueSortedDigests(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if !isSHA256Digest(trimmed) {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func isSHA256Digest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[len("sha256:"):] {
		if (c < 'a' || c > 'f') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func approvalOverlayDigests(records map[string]approvalRecord) map[string]struct{} {
	overlayDigests := map[string]struct{}{}
	for _, record := range records {
		if isSHA256Digest(record.Summary.RequestDigest) {
			overlayDigests[record.Summary.RequestDigest] = struct{}{}
		}
		if isSHA256Digest(record.Summary.DecisionDigest) {
			overlayDigests[record.Summary.DecisionDigest] = struct{}{}
		}
	}
	return overlayDigests
}

func approvalOverlaySourceDigests(records map[string]approvalRecord) map[string]struct{} {
	overlaySources := map[string]struct{}{}
	for _, record := range records {
		if isSHA256Digest(record.SourceDigest) {
			overlaySources[record.SourceDigest] = struct{}{}
		}
	}
	return overlaySources
}

func pruneDerivedApprovalOverlays(derived map[string]approvalRecord, overlayDigests map[string]struct{}, overlaySources map[string]struct{}) {
	for id, record := range derived {
		if _, ok := overlaySources[record.SourceDigest]; ok {
			delete(derived, id)
			continue
		}
		if _, ok := overlayDigests[record.Summary.RequestDigest]; ok {
			delete(derived, id)
			continue
		}
		if _, ok := overlayDigests[record.Summary.DecisionDigest]; ok {
			delete(derived, id)
		}
	}
}

func classifyApprovalArtifacts(records []artifacts.ArtifactRecord) (map[string]artifacts.ArtifactRecord, map[string]struct{}, []artifacts.ArtifactRecord) {
	unapprovedByDigest := map[string]artifacts.ArtifactRecord{}
	consumedUnapproved := map[string]struct{}{}
	approvedRecords := make([]artifacts.ArtifactRecord, 0)
	for _, record := range records {
		switch record.Reference.DataClass {
		case artifacts.DataClassUnapprovedFileExcerpts:
			if record.Reference.Digest != "" {
				unapprovedByDigest[record.Reference.Digest] = record
			}
		case artifacts.DataClassApprovedFileExcerpts:
			approvedRecords = append(approvedRecords, record)
			if record.ApprovalOfDigest != "" {
				consumedUnapproved[record.ApprovalOfDigest] = struct{}{}
			}
		}
	}
	return unapprovedByDigest, consumedUnapproved, approvedRecords
}

func addResolvedApprovalRecords(target map[string]approvalRecord, approvedRecords []artifacts.ArtifactRecord, unapprovedByDigest map[string]artifacts.ArtifactRecord) {
	for _, approved := range approvedRecords {
		source, hasSource := unapprovedByDigest[approved.ApprovalOfDigest]
		resolved := inferredResolvedApprovalRecord(approved, source, hasSource)
		if resolved.Summary.ApprovalID != "" {
			target[resolved.Summary.ApprovalID] = resolved
		}
	}
}

func addPendingApprovalRecords(target map[string]approvalRecord, unapprovedByDigest map[string]artifacts.ArtifactRecord, consumedUnapproved map[string]struct{}, now time.Time) {
	for digest, record := range unapprovedByDigest {
		if _, ok := consumedUnapproved[digest]; ok {
			continue
		}
		pending := inferredPendingApprovalRecord(record, now)
		target[pending.Summary.ApprovalID] = pending
	}
}

func resolvedApprovalTimes(record artifacts.ArtifactRecord, source artifacts.ArtifactRecord, hasSource bool) (string, string) {
	requestTime := record.CreatedAt
	if hasSource && !source.CreatedAt.IsZero() {
		requestTime = source.CreatedAt
	}
	if requestTime.IsZero() {
		requestTime = time.Now().UTC()
	}
	decidedTime := record.CreatedAt
	if record.PromotionApprovedAt != nil {
		decidedTime = record.PromotionApprovedAt.UTC()
	}
	if decidedTime.IsZero() {
		decidedTime = requestTime
	}
	return requestTime.UTC().Format(time.RFC3339), decidedTime.UTC().Format(time.RFC3339)
}

func resolvedApprovalScope(record artifacts.ArtifactRecord) ApprovalBoundScope {
	return ApprovalBoundScope{
		SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
		SchemaVersion: "0.1.0",
		WorkspaceID:   workspaceIDForRun(record.RunID),
		RunID:         record.RunID,
		StageID:       stageIDForRun(record.RunID),
		StepID:        record.StepID,
		ActionKind:    "excerpt_promotion",
	}
}
