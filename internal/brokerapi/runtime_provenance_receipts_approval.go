package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) persistApprovalResolutionReceipts(record approvalRecord, resolutionReason string, approvedArtifact *ArtifactSummary) {
	if s == nil || s.auditLedger == nil {
		return
	}
	sealDigest, ok := s.currentRuntimeProvenanceSealDigest()
	if !ok {
		return
	}
	s.persistApprovalResolutionReceipt(sealDigest, record, resolutionReason)
	s.persistApprovalConsumptionReceipt(sealDigest, record)
	s.persistApprovalPublicationReceipt(sealDigest, record, approvedArtifact)
	s.persistApprovalOverrideReceipt(sealDigest, record)
}

func (s *Service) persistApprovalResolutionReceipt(sealDigest trustpolicy.Digest, record approvalRecord, resolutionReason string) {
	payload, err := approvalResolutionReceiptPayload(record, resolutionReason)
	if err != nil {
		return
	}
	_ = s.persistRuntimeProvenanceReceipt(
		sealDigest,
		auditReceiptKindApprovalResolution,
		trustpolicy.AuditSegmentAnchoringSubjectSeal,
		auditReceiptPayloadSchemaApprovalEvidenceV0,
		payload,
	)
}

func (s *Service) persistApprovalConsumptionReceipt(sealDigest trustpolicy.Digest, record approvalRecord) {
	payload, hasConsumption, err := approvalConsumptionReceiptPayload(record)
	if err != nil || !hasConsumption {
		return
	}
	_ = s.persistRuntimeProvenanceReceipt(
		sealDigest,
		auditReceiptKindApprovalConsumption,
		trustpolicy.AuditSegmentAnchoringSubjectSeal,
		auditReceiptPayloadSchemaApprovalEvidenceV0,
		payload,
	)
}

func (s *Service) persistApprovalPublicationReceipt(sealDigest trustpolicy.Digest, record approvalRecord, approvedArtifact *ArtifactSummary) {
	if approvedArtifact == nil {
		return
	}
	payload, err := approvalPublicationReceiptPayload(record, *approvedArtifact)
	if err != nil {
		return
	}
	_ = s.persistRuntimeProvenanceReceipt(
		sealDigest,
		auditReceiptKindArtifactPublished,
		trustpolicy.AuditSegmentAnchoringSubjectSeal,
		auditReceiptPayloadSchemaPublicationV0,
		payload,
	)
}

func (s *Service) persistApprovalOverrideReceipt(sealDigest trustpolicy.Digest, record approvalRecord) {
	payload, hasOverride, err := approvalOverrideReceiptPayload(record)
	if err != nil || !hasOverride {
		return
	}
	_ = s.persistRuntimeProvenanceReceipt(
		sealDigest,
		auditReceiptKindOverrideOrBreakGlass,
		trustpolicy.AuditSegmentAnchoringSubjectSeal,
		auditReceiptPayloadSchemaOverrideV0,
		payload,
	)
}

func approvalResolutionReceiptPayload(record approvalRecord, resolutionReason string) (map[string]any, error) {
	if strings.TrimSpace(record.Summary.ApprovalID) == "" {
		return nil, fmt.Errorf("approval_id missing")
	}
	requestDigest, err := digestFromIdentity(strings.TrimSpace(record.Summary.RequestDigest))
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"approval_id":            strings.TrimSpace(record.Summary.ApprovalID),
		"approval_status":        strings.TrimSpace(record.Summary.Status),
		"request_digest":         requestDigest,
		"recorded_from":          "approval_resolve",
		"action_kind":            strings.TrimSpace(record.Summary.BoundScope.ActionKind),
		"resolution_reason_code": strings.TrimSpace(resolutionReason),
	}
	for _, field := range []struct {
		key      string
		identity string
	}{
		{key: "decision_digest", identity: strings.TrimSpace(record.Summary.DecisionDigest)},
		{key: "scope_digest", identity: strings.TrimSpace(record.Summary.ScopeDigest)},
		{key: "artifact_set_digest", identity: strings.TrimSpace(record.Summary.ArtifactSetDigest)},
		{key: "diff_digest", identity: strings.TrimSpace(record.Summary.DiffDigest)},
		{key: "summary_preview_digest", identity: strings.TrimSpace(record.Summary.SummaryPreviewDigest)},
		{key: "consumption_link_digest", identity: strings.TrimSpace(record.Summary.ConsumptionLinkDigest)},
		{key: "policy_decision_digest", identity: strings.TrimSpace(record.Summary.PolicyDecisionHash)},
	} {
		if err := appendOptionalDigestIdentity(payload, field.key, field.identity); err != nil {
			return nil, err
		}
	}
	if runDigest := hashIdentityDigest(strings.TrimSpace(record.Summary.BoundScope.RunID)); runDigest != nil {
		payload["run_id_digest"] = *runDigest
	}
	if approver := recordDecisionApproverIdentity(record); approver != nil {
		payload["approver"] = *approver
	}
	return payload, nil
}

func approvalConsumptionReceiptPayload(record approvalRecord) (map[string]any, bool, error) {
	if strings.TrimSpace(record.Summary.Status) != "consumed" {
		return nil, false, nil
	}
	payload, err := approvalResolutionReceiptPayload(record, "approval_consumed")
	if err != nil {
		return nil, false, err
	}
	payload["approval_status"] = "consumed"
	payload["recorded_from"] = "approval_consumption"
	return payload, true, nil
}

func approvalPublicationReceiptPayload(record approvalRecord, approvedArtifact ArtifactSummary) (map[string]any, error) {
	artifactDigest, err := digestFromIdentity(strings.TrimSpace(approvedArtifact.Reference.Digest))
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"publication_kind": "promotion",
		"artifact_digest":  artifactDigest,
		"action_kind":      strings.TrimSpace(record.Summary.BoundScope.ActionKind),
	}
	for _, field := range []struct {
		key      string
		identity string
	}{
		{key: "source_artifact_digest", identity: strings.TrimSpace(record.SourceDigest)},
		{key: "approval_decision_digest", identity: strings.TrimSpace(record.Summary.DecisionDigest)},
		{key: "approval_link_digest", identity: strings.TrimSpace(record.Summary.ConsumptionLinkDigest)},
	} {
		if err := appendOptionalDigestIdentity(payload, field.key, field.identity); err != nil {
			return nil, err
		}
	}
	if runDigest := hashIdentityDigest(strings.TrimSpace(record.Summary.BoundScope.RunID)); runDigest != nil {
		payload["run_id_digest"] = *runDigest
	}
	return payload, nil
}

func approvalOverrideReceiptPayload(record approvalRecord) (map[string]any, bool, error) {
	if strings.TrimSpace(record.Summary.BoundScope.ActionKind) != "action_gate_override" {
		return nil, false, nil
	}
	payload := map[string]any{
		"override_kind":     "gate_override",
		"approval_required": true,
		"approval_consumed": strings.TrimSpace(record.Summary.Status) == "consumed",
	}
	for _, field := range []struct {
		key      string
		identity string
	}{
		{key: "policy_decision_digest", identity: strings.TrimSpace(record.Summary.PolicyDecisionHash)},
		{key: "action_request_digest", identity: strings.TrimSpace(record.Summary.ConsumedActionHash)},
		{key: "approval_link_digest", identity: strings.TrimSpace(record.Summary.ConsumptionLinkDigest)},
	} {
		if err := appendOptionalDigestIdentity(payload, field.key, field.identity); err != nil {
			return nil, false, err
		}
	}
	if runDigest := hashIdentityDigest(strings.TrimSpace(record.Summary.BoundScope.RunID)); runDigest != nil {
		payload["run_id_digest"] = *runDigest
	}
	return payload, true, nil
}

func appendOptionalDigestIdentity(payload map[string]any, key, identity string) error {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return nil
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	payload[key] = digest
	return nil
}

func recordDecisionApproverIdentity(record approvalRecord) *trustpolicy.PrincipalIdentity {
	if record.DecisionEnvelope == nil {
		return nil
	}
	decision, err := decodeApprovalDecision(*record.DecisionEnvelope)
	if err != nil {
		return nil
	}
	if strings.TrimSpace(decision.Approver.PrincipalID) == "" {
		return nil
	}
	identity := decision.Approver
	return &identity
}
