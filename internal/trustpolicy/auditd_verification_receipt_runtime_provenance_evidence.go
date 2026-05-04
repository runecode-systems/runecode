package trustpolicy

import (
	"fmt"
	"strings"
)

func validateApprovalEvidenceReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaApprovalEvidenceV0 {
		return fmt.Errorf("%s receipts require approval evidence payload schema", receipt.AuditReceiptKind)
	}
	payload := approvalEvidenceReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode approval evidence payload: %w", err)
	}
	if strings.TrimSpace(payload.ApprovalID) == "" {
		return fmt.Errorf("approval_id is required")
	}
	if strings.TrimSpace(payload.ApprovalStatus) == "" {
		return fmt.Errorf("approval_status is required")
	}
	if _, err := payload.RequestDigest.Identity(); err != nil {
		return fmt.Errorf("request_digest: %w", err)
	}
	if err := validateApprovalEvidenceOptionalDigests(payload); err != nil {
		return err
	}
	if err := validatePrincipalIdentityOptional(payload.Approver, "approver"); err != nil {
		return err
	}
	if receipt.AuditReceiptKind == auditReceiptKindApprovalConsumption && payload.ConsumptionLinkDigest == nil {
		return fmt.Errorf("consumption_link_digest is required for %s", auditReceiptKindApprovalConsumption)
	}
	return nil
}

func validateApprovalEvidenceOptionalDigests(payload approvalEvidenceReceiptPayload) error {
	for _, field := range []struct {
		name   string
		digest *Digest
	}{
		{name: "decision_digest", digest: payload.DecisionDigest},
		{name: "scope_digest", digest: payload.ScopeDigest},
		{name: "artifact_set_digest", digest: payload.ArtifactSetDigest},
		{name: "diff_digest", digest: payload.DiffDigest},
		{name: "summary_preview_digest", digest: payload.SummaryPreviewDigest},
		{name: "consumption_link_digest", digest: payload.ConsumptionLinkDigest},
		{name: "policy_decision_digest", digest: payload.PolicyDecisionDigest},
		{name: "run_id_digest", digest: payload.RunIDDigest},
	} {
		if err := validateOptionalReceiptDigestField(field.digest, field.name); err != nil {
			return err
		}
	}
	return nil
}

func validatePublicationEvidenceReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaPublicationV0 {
		return fmt.Errorf("%s receipts require publication evidence payload schema", receipt.AuditReceiptKind)
	}
	payload := publicationEvidenceReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode publication evidence payload: %w", err)
	}
	if strings.TrimSpace(payload.PublicationKind) == "" {
		return fmt.Errorf("publication_kind is required")
	}
	if _, err := payload.ArtifactDigest.Identity(); err != nil {
		return fmt.Errorf("artifact_digest: %w", err)
	}
	for _, field := range []struct {
		name   string
		digest *Digest
	}{
		{name: "source_artifact_digest", digest: payload.SourceArtifactDigest},
		{name: "approval_decision_digest", digest: payload.ApprovalDecisionDigest},
		{name: "approval_link_digest", digest: payload.ApprovalLinkDigest},
		{name: "run_id_digest", digest: payload.RunIDDigest},
	} {
		if err := validateOptionalReceiptDigestField(field.digest, field.name); err != nil {
			return err
		}
	}
	return nil
}

func validateOverrideEvidenceReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaOverrideV0 {
		return fmt.Errorf("%s receipts require override evidence payload schema", receipt.AuditReceiptKind)
	}
	payload := overrideEvidenceReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode override evidence payload: %w", err)
	}
	if strings.TrimSpace(payload.OverrideKind) == "" {
		return fmt.Errorf("override_kind is required")
	}
	if payload.ApprovalConsumed && !payload.ApprovalRequired {
		return fmt.Errorf("approval_consumed=true requires approval_required=true")
	}
	for _, field := range []struct {
		name   string
		digest *Digest
	}{
		{name: "policy_decision_digest", digest: payload.PolicyDecisionDigest},
		{name: "action_request_digest", digest: payload.ActionRequestDigest},
		{name: "approval_link_digest", digest: payload.ApprovalLinkDigest},
		{name: "run_id_digest", digest: payload.RunIDDigest},
	} {
		if err := validateOptionalReceiptDigestField(field.digest, field.name); err != nil {
			return err
		}
	}
	return nil
}
