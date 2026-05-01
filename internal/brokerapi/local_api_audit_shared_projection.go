package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

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
	detail.Summary = auditReceiptProjectionSummary(receipt)
	if subjectRef, ok := projectedReceiptSubjectReference(receipt); ok {
		detail.LinkedReferences = append(detail.LinkedReferences, subjectRef)
	}
	if receipt.ApprovalDecision != nil {
		detail.LinkedReferences = append(detail.LinkedReferences, digestPointerLinkedReference(receipt.ApprovalDecision, "approval", "approval_decision"))
	}
	if receipt.AnchorWitnessDigest != nil {
		detail.LinkedReferences = append(detail.LinkedReferences, digestPointerLinkedReference(receipt.AnchorWitnessDigest, "artifact", "anchor_witness"))
	}
	detail.LinkedReferences = filterEmptyLinkedReferences(detail.LinkedReferences)
	return nil
}

func isExternalAnchorReceipt(receipt *trustpolicy.AuditReceiptOperationalView) bool {
	if receipt == nil {
		return false
	}
	if strings.TrimSpace(receipt.ExternalTargetKind) != "" || strings.TrimSpace(receipt.ExternalRuntimeAdapter) != "" || strings.TrimSpace(receipt.ExternalProofKind) != "" || strings.TrimSpace(receipt.ExternalProofSchema) != "" {
		return true
	}
	return receipt.ExternalTargetDescriptorDigest != nil || receipt.ExternalProofDigest != nil
}

func auditReceiptProjectionSummary(receipt *trustpolicy.AuditReceiptOperationalView) string {
	if receipt == nil {
		return "Audit receipt recorded."
	}
	receiptKind := strings.TrimSpace(receipt.AuditReceiptKind)
	if receiptKind == "anchor" {
		if isExternalAnchorReceipt(receipt) {
			return "Audit receipt (anchor) recorded [posture=external_completed]."
		}
		return "Audit receipt (anchor) recorded [posture=local_only_completed]."
	}
	return fmt.Sprintf("Audit receipt (%s) recorded.", receiptKind)
}

func projectedReceiptSubjectReference(receipt *trustpolicy.AuditReceiptOperationalView) (AuditRecordLinkedReference, bool) {
	if receipt == nil {
		return AuditRecordLinkedReference{}, false
	}
	relation := "subject"
	subjectFamily := strings.TrimSpace(receipt.SubjectFamily)
	receiptKind := strings.TrimSpace(receipt.AuditReceiptKind)
	if subjectFamily == "audit_segment_seal" || (subjectFamily == "" && receiptKind == "anchor") {
		relation = "subject_segment_seal"
	}
	ref := digestLinkedReference(receipt.SubjectDigest, "audit_record", relation)
	if strings.TrimSpace(ref.ReferenceID) == "" {
		return AuditRecordLinkedReference{}, false
	}
	return ref, true
}

func digestLinkedReference(digest trustpolicy.Digest, kind string, relation string) AuditRecordLinkedReference {
	identity, err := digest.Identity()
	if err != nil || identity == "" {
		return AuditRecordLinkedReference{}
	}
	return AuditRecordLinkedReference{ReferenceKind: kind, ReferenceID: identity, Relation: relation}
}

func digestPointerLinkedReference(digest *trustpolicy.Digest, kind string, relation string) AuditRecordLinkedReference {
	if digest == nil {
		return AuditRecordLinkedReference{}
	}
	return digestLinkedReference(*digest, kind, relation)
}

func filterEmptyLinkedReferences(in []AuditRecordLinkedReference) []AuditRecordLinkedReference {
	out := in[:0]
	for _, ref := range in {
		if strings.TrimSpace(ref.ReferenceID) == "" {
			continue
		}
		out = append(out, ref)
	}
	return out
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

func (s *Service) deriveRecordVerificationPosture(recordDigest string) ([]string, *AuditRecordVerificationPosture) {
	if s == nil || s.auditLedger == nil {
		return nil, nil
	}
	report, err := s.auditLedger.LatestVerificationReport()
	if err != nil {
		return nil, nil
	}
	postures := deriveRecordVerificationPosturesFromFindings(report.Findings)
	posture, ok := postures[recordDigest]
	if !ok {
		return []string{}, &AuditRecordVerificationPosture{Status: "ok", ReasonCodes: []string{}}
	}
	return append([]string{}, posture.ReasonCodes...), cloneVerificationPosture(posture)
}
