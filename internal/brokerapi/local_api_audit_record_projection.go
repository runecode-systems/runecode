package brokerapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) projectAuditRecordDetail(recordIdentity string, envelope trustpolicy.SignedObjectEnvelope) (AuditRecordDetail, error) {
	view, viewDigest, detail, err := baseProjectedAuditRecordDetail(recordIdentity, envelope)
	if err != nil {
		return AuditRecordDetail{}, err
	}
	if err := projectAuditRecordFamilyDetail(&detail, envelope, view); err != nil {
		return AuditRecordDetail{}, err
	}
	reasons, posture := s.deriveRecordVerificationPosture(viewDigest)
	if posture != nil {
		detail.VerificationPosture = posture
		detail.LinkedReferences = append(detail.LinkedReferences, verificationReasonRefs(reasons)...)
	}
	detail.LinkedReferences = dedupeAuditRecordReferences(detail.LinkedReferences)
	return detail, nil
}

func baseProjectedAuditRecordDetail(recordIdentity string, envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.AuditOperationalView, string, AuditRecordDetail, error) {
	view, err := trustpolicy.BuildDefaultOperationalAuditView(envelope)
	if err != nil {
		return trustpolicy.AuditOperationalView{}, "", AuditRecordDetail{}, err
	}
	viewDigest, err := view.RecordDigest.Identity()
	if err != nil {
		return trustpolicy.AuditOperationalView{}, "", AuditRecordDetail{}, err
	}
	if viewDigest != recordIdentity {
		return trustpolicy.AuditOperationalView{}, "", AuditRecordDetail{}, fmt.Errorf("record_digest projection mismatch")
	}
	detail := AuditRecordDetail{SchemaID: "runecode.protocol.v0.AuditRecordDetail", SchemaVersion: "0.1.0", RecordDigest: view.RecordDigest, LinkedReferences: []AuditRecordLinkedReference{}}
	return view, viewDigest, detail, nil
}

func projectAuditRecordFamilyDetail(detail *AuditRecordDetail, envelope trustpolicy.SignedObjectEnvelope, view trustpolicy.AuditOperationalView) error {
	switch envelope.PayloadSchemaID {
	case trustpolicy.AuditEventSchemaID:
		return projectAuditEventRecordDetail(detail, view.Event)
	case trustpolicy.AuditReceiptSchemaID:
		return projectAuditReceiptRecordDetail(detail, view.Receipt)
	case trustpolicy.AuditSegmentSealSchemaID:
		return projectAuditSegmentSealRecordDetail(detail, envelope.Payload)
	default:
		return fmt.Errorf("unsupported audit record family for payload_schema_id %q", envelope.PayloadSchemaID)
	}
}

func projectAuditEventRecordDetail(detail *AuditRecordDetail, event *trustpolicy.AuditEventOperationalPayload) error {
	if event == nil {
		return fmt.Errorf("audit event projection missing event payload")
	}
	detail.RecordFamily = "audit_event"
	detail.OccurredAt = event.OccurredAt
	detail.EventType = strings.TrimSpace(event.AuditEventType)
	detail.Summary = fmt.Sprintf("Audit event %s recorded.", detail.EventType)
	detail.LinkedReferences = append(detail.LinkedReferences, projectEventRefs(event)...)
	detail.Scope = projectAuditScope(event.Scope)
	detail.Correlation = projectAuditCorrelation(event.Correlation)
	return nil
}

func projectAuditReceiptRecordDetail(detail *AuditRecordDetail, receipt *trustpolicy.AuditReceiptOperationalView) error {
	if receipt == nil {
		return fmt.Errorf("audit receipt projection missing receipt payload")
	}
	detail.RecordFamily = "audit_receipt"
	detail.OccurredAt = receipt.RecordedAt
	detail.Summary = fmt.Sprintf("Audit receipt (%s) recorded.", strings.TrimSpace(receipt.AuditReceiptKind))
	if subject, err := receipt.SubjectDigest.Identity(); err == nil && subject != "" {
		detail.LinkedReferences = append(detail.LinkedReferences, AuditRecordLinkedReference{ReferenceKind: "audit_record", ReferenceID: subject, Relation: "subject"})
	}
	return nil
}

func projectAuditSegmentSealRecordDetail(detail *AuditRecordDetail, payload json.RawMessage) error {
	seal := trustpolicy.AuditSegmentSealPayload{}
	if err := json.Unmarshal(payload, &seal); err != nil {
		return fmt.Errorf("decode audit segment seal payload: %w", err)
	}
	if err := trustpolicy.ValidateAuditSegmentSealPayload(seal); err != nil {
		return fmt.Errorf("validate audit segment seal payload: %w", err)
	}
	detail.RecordFamily = "audit_segment_seal"
	detail.OccurredAt = seal.SealedAt
	detail.Summary = fmt.Sprintf("Audit segment seal recorded for %s.", strings.TrimSpace(seal.SegmentID))
	if firstID, err := seal.FirstRecordDigest.Identity(); err == nil {
		detail.LinkedReferences = append(detail.LinkedReferences, AuditRecordLinkedReference{ReferenceKind: "audit_record", ReferenceID: firstID, Relation: "first_record"})
	}
	if lastID, err := seal.LastRecordDigest.Identity(); err == nil {
		detail.LinkedReferences = append(detail.LinkedReferences, AuditRecordLinkedReference{ReferenceKind: "audit_record", ReferenceID: lastID, Relation: "last_record"})
	}
	return nil
}

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

func (s *Service) deriveRecordVerificationPosture(recordDigest string) ([]string, *AuditRecordVerificationPosture) {
	surface, err := s.LatestAuditVerificationSurface(1000)
	if err != nil {
		return nil, nil
	}
	reasons := map[string]struct{}{}
	status := "ok"
	for _, finding := range surface.Report.Findings {
		if !findingAppliesToRecord(finding, recordDigest) {
			continue
		}
		reasons[finding.Code] = struct{}{}
		if finding.Severity == trustpolicy.AuditVerificationSeverityError {
			status = "failed"
			continue
		}
		if status != "failed" {
			status = "degraded"
		}
	}
	if len(reasons) == 0 {
		return nil, nil
	}
	reasonCodes := make([]string, 0, len(reasons))
	for code := range reasons {
		reasonCodes = append(reasonCodes, code)
	}
	sort.Strings(reasonCodes)
	return reasonCodes, &AuditRecordVerificationPosture{Status: status, ReasonCodes: reasonCodes}
}

func findingAppliesToRecord(finding trustpolicy.AuditVerificationFinding, recordDigest string) bool {
	if finding.SubjectRecordDigest != nil {
		if id, err := finding.SubjectRecordDigest.Identity(); err == nil && id == recordDigest {
			return true
		}
	}
	for _, related := range finding.RelatedRecordDigests {
		if id, err := related.Identity(); err == nil && id == recordDigest {
			return true
		}
	}
	return false
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
