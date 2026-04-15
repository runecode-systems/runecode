package brokerapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) projectAuditTimelineEntries(views []trustpolicy.AuditOperationalView, postures map[string]AuditRecordVerificationPosture) []AuditTimelineViewEntry {
	out := make([]AuditTimelineViewEntry, 0, len(views))
	for _, view := range views {
		entry, ok := projectAuditTimelineEntry(view, postures)
		if ok {
			out = append(out, entry)
		}
	}
	return out
}

func projectAuditTimelineEntry(view trustpolicy.AuditOperationalView, postures map[string]AuditRecordVerificationPosture) (AuditTimelineViewEntry, bool) {
	recordDigest, err := view.RecordDigest.Identity()
	if err != nil || recordDigest == "" {
		return AuditTimelineViewEntry{}, false
	}
	entry := AuditTimelineViewEntry{RecordDigest: view.RecordDigest}
	projectTimelineEventSection(&entry, view.Event)
	projectTimelineReceiptSection(&entry, view.Receipt)
	if posture := timelineVerificationPosture(recordDigest, postures); posture != nil {
		entry.VerificationPosture = posture
		entry.LinkedReferences = append(entry.LinkedReferences, verificationReasonRefs(posture.ReasonCodes)...)
	}
	entry.LinkedReferences = dedupeAuditRecordReferences(entry.LinkedReferences)
	if strings.TrimSpace(entry.Summary) == "" {
		entry.Summary = "Audit record projected for timeline."
	}
	return entry, true
}

func projectTimelineEventSection(entry *AuditTimelineViewEntry, event *trustpolicy.AuditEventOperationalPayload) {
	if entry == nil || event == nil {
		return
	}
	entry.EventType = strings.TrimSpace(event.AuditEventType)
	entry.Summary = fmt.Sprintf("Audit event %s recorded.", entry.EventType)
	entry.LinkedReferences = append(entry.LinkedReferences, projectEventRefs(event)...)
}

func projectTimelineReceiptSection(entry *AuditTimelineViewEntry, receipt *trustpolicy.AuditReceiptOperationalView) {
	if entry == nil || receipt == nil {
		return
	}
	entry.Summary = fmt.Sprintf("Audit receipt (%s) recorded.", strings.TrimSpace(receipt.AuditReceiptKind))
	if subject, ok := digestIdentity(receipt.SubjectDigest); ok {
		entry.LinkedReferences = append(entry.LinkedReferences, AuditRecordLinkedReference{ReferenceKind: "audit_record", ReferenceID: subject, Relation: "subject"})
	}
	if receipt.ApprovalDecision != nil {
		if approvalID, ok := digestIdentity(*receipt.ApprovalDecision); ok {
			entry.LinkedReferences = append(entry.LinkedReferences, AuditRecordLinkedReference{ReferenceKind: "approval", ReferenceID: approvalID, Relation: "approval_decision"})
		}
	}
	if receipt.AnchorWitnessDigest != nil {
		if witnessID, ok := digestIdentity(*receipt.AnchorWitnessDigest); ok {
			entry.LinkedReferences = append(entry.LinkedReferences, AuditRecordLinkedReference{ReferenceKind: "artifact", ReferenceID: witnessID, Relation: "anchor_witness"})
		}
	}
}

func digestIdentity(d trustpolicy.Digest) (string, bool) {
	identity, err := d.Identity()
	if err != nil || identity == "" {
		return "", false
	}
	return identity, true
}

func deriveRecordVerificationPosturesFromFindings(findings []trustpolicy.AuditVerificationFinding) map[string]AuditRecordVerificationPosture {
	if len(findings) == 0 {
		return map[string]AuditRecordVerificationPosture{}
	}
	reasonsByRecord, statusByRecord := buildFindingStateByRecord(findings)
	return finalizeRecordPostures(reasonsByRecord, statusByRecord)
}

func buildFindingStateByRecord(findings []trustpolicy.AuditVerificationFinding) (map[string]map[string]struct{}, map[string]string) {
	reasonsByRecord := map[string]map[string]struct{}{}
	statusByRecord := map[string]string{}
	for _, finding := range findings {
		code := strings.TrimSpace(finding.Code)
		if code == "" {
			continue
		}
		for _, record := range findingDigests(finding) {
			reasons := reasonsByRecord[record]
			if reasons == nil {
				reasons = map[string]struct{}{}
				reasonsByRecord[record] = reasons
			}
			reasons[code] = struct{}{}
			statusByRecord[record] = mergeVerificationStatus(statusByRecord[record], finding.Severity)
		}
	}
	return reasonsByRecord, statusByRecord
}

func finalizeRecordPostures(reasonsByRecord map[string]map[string]struct{}, statusByRecord map[string]string) map[string]AuditRecordVerificationPosture {
	out := make(map[string]AuditRecordVerificationPosture, len(reasonsByRecord))
	for record, reasons := range reasonsByRecord {
		reasonCodes := sortedReasonCodes(reasons)
		status := statusByRecord[record]
		if status == "" {
			status = "ok"
		}
		out[record] = AuditRecordVerificationPosture{Status: status, ReasonCodes: reasonCodes}
	}
	return out
}

func sortedReasonCodes(reasons map[string]struct{}) []string {
	reasonCodes := make([]string, 0, len(reasons))
	for code := range reasons {
		reasonCodes = append(reasonCodes, code)
	}
	sort.Strings(reasonCodes)
	return reasonCodes
}

func findingDigests(finding trustpolicy.AuditVerificationFinding) []string {
	identities := make([]string, 0, 1+len(finding.RelatedRecordDigests))
	if finding.SubjectRecordDigest != nil {
		if id, err := finding.SubjectRecordDigest.Identity(); err == nil && id != "" {
			identities = append(identities, id)
		}
	}
	for _, digest := range finding.RelatedRecordDigests {
		if id, err := digest.Identity(); err == nil && id != "" {
			identities = append(identities, id)
		}
	}
	return identities
}

func timelineVerificationPosture(recordDigest string, postures map[string]AuditRecordVerificationPosture) *AuditRecordVerificationPosture {
	if posture, ok := postures[recordDigest]; ok {
		return cloneVerificationPosture(posture)
	}
	return &AuditRecordVerificationPosture{Status: "ok", ReasonCodes: []string{}}
}

func cloneVerificationPosture(posture AuditRecordVerificationPosture) *AuditRecordVerificationPosture {
	return &AuditRecordVerificationPosture{Status: posture.Status, ReasonCodes: append([]string{}, posture.ReasonCodes...)}
}

func mergeVerificationStatus(current string, severity string) string {
	if severity == trustpolicy.AuditVerificationSeverityError {
		return "failed"
	}
	if current == "failed" {
		return current
	}
	if severity == trustpolicy.AuditVerificationSeverityWarning {
		return "degraded"
	}
	if severity == trustpolicy.AuditVerificationSeverityInfo {
		if current == "" {
			return "ok"
		}
		return current
	}
	if current == "" {
		return "degraded"
	}
	return current
}
