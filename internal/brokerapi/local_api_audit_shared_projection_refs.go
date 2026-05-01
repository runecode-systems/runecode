package brokerapi

import (
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func verificationReasonRefs(reasons []string) []AuditRecordLinkedReference {
	refs := make([]AuditRecordLinkedReference, 0, len(reasons))
	for _, code := range reasons {
		refs = append(refs, AuditRecordLinkedReference{ReferenceKind: "verification_reason", ReferenceID: code, Relation: "posture_reason"})
	}
	return refs
}

func projectEventRefs(event *trustpolicy.AuditEventOperationalPayload) []AuditRecordLinkedReference {
	if event == nil {
		return []AuditRecordLinkedReference{}
	}
	refs := make([]AuditRecordLinkedReference, 0, 8)
	refs = appendMappedAuditRefs(refs, event.SubjectRef)
	refs = appendMappedAuditRefSlice(refs, event.CauseRefs)
	refs = appendMappedAuditRefSlice(refs, event.RelatedRefs)
	refs = appendMappedAuditRefSlice(refs, event.SignerEvidenceRefs)
	return refs
}

func appendMappedAuditRefSlice(refs []AuditRecordLinkedReference, items []trustpolicy.AuditTypedReference) []AuditRecordLinkedReference {
	for _, item := range items {
		refs = appendMappedAuditRefs(refs, &item)
	}
	return refs
}

func appendMappedAuditRefs(refs []AuditRecordLinkedReference, ref *trustpolicy.AuditTypedReference) []AuditRecordLinkedReference {
	if next, ok := mapTypedReference(ref); ok {
		return append(refs, next)
	}
	return refs
}

func mapTypedReference(ref *trustpolicy.AuditTypedReference) (AuditRecordLinkedReference, bool) {
	if ref == nil {
		return AuditRecordLinkedReference{}, false
	}
	referenceID, err := ref.Digest.Identity()
	if err != nil {
		return AuditRecordLinkedReference{}, false
	}
	referenceKind, ok := auditReferenceKind(strings.TrimSpace(ref.ObjectFamily))
	if !ok {
		return AuditRecordLinkedReference{}, false
	}
	return AuditRecordLinkedReference{ReferenceKind: referenceKind, ReferenceID: referenceID, Relation: strings.TrimSpace(ref.RefRole)}, true
}

func auditReferenceKind(objectFamily string) (string, bool) {
	switch objectFamily {
	case "approval_request", "approval_decision":
		return "approval", true
	case "artifact":
		return "artifact", true
	case "audit_event", "audit_receipt", "audit_segment_seal":
		return "audit_record", true
	default:
		return "", false
	}
}

func projectAuditScope(scope map[string]string) *AuditRecordScope {
	if len(scope) == 0 {
		return nil
	}
	out := &AuditRecordScope{WorkspaceID: strings.TrimSpace(scope["workspace_id"]), RunID: strings.TrimSpace(scope["run_id"]), StageID: strings.TrimSpace(scope["stage_id"]), StepID: strings.TrimSpace(scope["step_id"])}
	if out.WorkspaceID == "" && out.RunID == "" && out.StageID == "" && out.StepID == "" {
		return nil
	}
	return out
}

func projectAuditCorrelation(correlation map[string]string) *AuditRecordCorrelation {
	if len(correlation) == 0 {
		return nil
	}
	out := &AuditRecordCorrelation{SessionID: strings.TrimSpace(correlation["session_id"]), OperationID: strings.TrimSpace(correlation["operation_id"]), ParentOperationID: strings.TrimSpace(correlation["parent_operation_id"])}
	if out.SessionID == "" && out.OperationID == "" && out.ParentOperationID == "" {
		return nil
	}
	return out
}

func dedupeAuditRecordReferences(in []AuditRecordLinkedReference) []AuditRecordLinkedReference {
	if len(in) == 0 {
		return []AuditRecordLinkedReference{}
	}
	seen := map[string]AuditRecordLinkedReference{}
	for _, next := range in {
		candidate, ok := normalizedAuditRecordReference(next)
		if !ok {
			continue
		}
		seen[auditReferenceKey(candidate)] = candidate
	}
	out := make([]AuditRecordLinkedReference, 0, len(seen))
	for _, next := range seen {
		out = append(out, next)
	}
	sort.Slice(out, func(i, j int) bool {
		return auditReferenceKey(out[i]) < auditReferenceKey(out[j])
	})
	return out
}

func normalizedAuditRecordReference(next AuditRecordLinkedReference) (AuditRecordLinkedReference, bool) {
	candidate := AuditRecordLinkedReference{ReferenceKind: strings.TrimSpace(next.ReferenceKind), ReferenceID: strings.TrimSpace(next.ReferenceID), Relation: strings.TrimSpace(next.Relation), Label: strings.TrimSpace(next.Label)}
	if candidate.ReferenceKind == "" || candidate.ReferenceID == "" {
		return AuditRecordLinkedReference{}, false
	}
	return candidate, true
}

func auditReferenceKey(ref AuditRecordLinkedReference) string {
	return ref.ReferenceKind + "|" + ref.ReferenceID + "|" + ref.Relation + "|" + ref.Label
}
