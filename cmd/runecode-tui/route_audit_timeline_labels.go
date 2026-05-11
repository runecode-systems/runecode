package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func auditRecordDisplayLabel(entry brokerapi.AuditTimelineViewEntry) string {
	digest, _ := entry.RecordDigest.Identity()
	if strings.TrimSpace(entry.Summary) != "" {
		return strings.TrimSpace(entry.Summary)
	}
	if label := auditEventLabel(entry.EventType); label != "Audit record" {
		return label
	}
	return shortIdentity(digest)
}

func auditRecordDigestLabel(entry brokerapi.AuditTimelineViewEntry) string {
	digest, _ := entry.RecordDigest.Identity()
	return shortIdentity(digest)
}

func auditRecordDigestDetail(record brokerapi.AuditRecordDetail) string {
	digest, _ := record.RecordDigest.Identity()
	return shortIdentity(digest)
}

func auditTimelinePostureLabel(entry brokerapi.AuditTimelineViewEntry) string {
	if entry.VerificationPosture == nil {
		return "verification unavailable"
	}
	return auditVerificationCue(entry.VerificationPosture.Status, entry.VerificationPosture.ReasonCodes)
}

func auditTimelineDirectoryItem(entry brokerapi.AuditTimelineViewEntry) string {
	label := auditRecordDisplayLabel(entry)
	cue := auditTimelinePostureLabel(entry)
	refs := auditReferenceCue(entry.LinkedReferences)
	digest := auditRecordDigestLabel(entry)
	parts := []string{label, digest, cue}
	if refs != "" {
		parts = append(parts, refs)
	}
	return strings.Join(parts, " • ")
}

func auditReferenceCue(refs []brokerapi.AuditRecordLinkedReference) string {
	if len(refs) == 0 {
		return ""
	}
	kinds := make([]string, 0, len(refs))
	seen := map[string]struct{}{}
	for _, ref := range refs {
		kind := strings.TrimSpace(ref.ReferenceKind)
		if kind == "" {
			kind = "linked item"
		}
		if _, ok := seen[kind]; ok {
			continue
		}
		seen[kind] = struct{}{}
		kinds = append(kinds, kind)
		if len(kinds) == 2 {
			break
		}
	}
	label := fmt.Sprintf("%d linked item", len(refs))
	if len(refs) != 1 {
		label += "s"
	}
	if len(kinds) == 0 {
		return label
	}
	return fmt.Sprintf("%s: %s", label, strings.Join(kinds, ", "))
}
