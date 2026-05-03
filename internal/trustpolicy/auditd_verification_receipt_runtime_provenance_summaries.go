package trustpolicy

import (
	"fmt"
	"slices"
	"strings"
)

func validateRuntimeSummaryReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaRuntimeSummaryV0 {
		return fmt.Errorf("%s receipts require runtime summary payload schema", receipt.AuditReceiptKind)
	}
	payload := runtimeSummaryReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode runtime summary payload: %w", err)
	}
	if err := validateRuntimeSummaryScope(payload); err != nil {
		return err
	}
	if err := validateRuntimeSummaryCounts(payload); err != nil {
		return err
	}
	if err := validateRuntimeSummaryAbsenceClaims(payload); err != nil {
		return err
	}
	return validateRuntimeSummaryBoundarySupport(payload)
}

func validateRuntimeSummaryScope(payload runtimeSummaryReceiptPayload) error {
	if strings.TrimSpace(payload.SummaryScopeKind) == "" {
		return fmt.Errorf("summary_scope_kind is required")
	}
	return nil
}

func validateRuntimeSummaryCounts(payload runtimeSummaryReceiptPayload) error {
	counts := []struct {
		name  string
		value int64
	}{
		{name: "provider_invocation_count", value: payload.ProviderInvocationCount},
		{name: "secret_lease_issue_count", value: payload.SecretLeaseIssueCount},
		{name: "secret_lease_revoke_count", value: payload.SecretLeaseRevokeCount},
		{name: "network_egress_count", value: payload.NetworkEgressCount},
		{name: "approval_consumption_count", value: payload.ApprovalConsumptionCount},
		{name: "boundary_crossing_count", value: payload.BoundaryCrossingCount},
	}
	for _, count := range counts {
		if count.value < 0 {
			return fmt.Errorf("%s must be >= 0", count.name)
		}
	}
	return nil
}

func validateRuntimeSummaryAbsenceClaims(payload runtimeSummaryReceiptPayload) error {
	checks := []struct {
		flag  bool
		count int64
		name  string
	}{
		{flag: payload.NoProviderInvocation, count: payload.ProviderInvocationCount, name: "no_provider_invocation"},
		{flag: payload.NoSecretLeaseIssued, count: payload.SecretLeaseIssueCount, name: "no_secret_lease_issued"},
		{flag: payload.NoApprovalConsumed, count: payload.ApprovalConsumptionCount, name: "no_approval_consumed"},
		{flag: payload.NoArtifactCrossedBoundary, count: payload.BoundaryCrossingCount, name: "no_artifact_crossed_boundary"},
	}
	for _, check := range checks {
		if check.flag && check.count != 0 {
			return fmt.Errorf("%s requires corresponding count=0", check.name)
		}
	}
	if payload.NoArtifactCrossedBoundary && strings.TrimSpace(payload.BoundaryRoute) == "" {
		return fmt.Errorf("boundary_route is required when no_artifact_crossed_boundary=true")
	}
	return nil
}

func validateRuntimeSummaryBoundarySupport(payload runtimeSummaryReceiptPayload) error {
	if strings.TrimSpace(payload.BoundaryCrossingSupport) == "" {
		return nil
	}
	if !slices.Contains([]string{"explicit", "limited"}, strings.TrimSpace(payload.BoundaryCrossingSupport)) {
		return fmt.Errorf("boundary_crossing_support must be explicit or limited when provided")
	}
	return nil
}

func validateDegradedPostureSummaryReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaDegradedPostureV0 {
		return fmt.Errorf("%s receipts require degraded posture summary payload schema", receipt.AuditReceiptKind)
	}
	payload := degradedPostureSummaryReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode degraded posture summary payload: %w", err)
	}
	if err := validateDegradedPostureCore(payload); err != nil {
		return err
	}
	if err := validateDegradedPostureAcknowledgment(payload); err != nil {
		return err
	}
	if err := validateDegradedPostureApprovalAndOverride(payload); err != nil {
		return err
	}
	return validateDegradedPostureOptionalDigests(payload)
}

func validateDegradedPostureCore(payload degradedPostureSummaryReceiptPayload) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "summary_scope_kind", value: payload.SummaryScopeKind},
		{name: "degradation_cause_code", value: payload.DegradationCauseCode},
		{name: "trust_claim_before", value: payload.TrustClaimBefore},
		{name: "trust_claim_after", value: payload.TrustClaimAfter},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	if payload.ChangedTrustClaim != (strings.TrimSpace(payload.TrustClaimBefore) != strings.TrimSpace(payload.TrustClaimAfter)) {
		return fmt.Errorf("changed_trust_claim must match trust_claim_before/trust_claim_after comparison")
	}
	if payload.Degraded && len(payload.DegradationReasonCodes) == 0 {
		return fmt.Errorf("degradation_reason_codes are required when degraded=true")
	}
	return nil
}

func validateDegradedPostureAcknowledgment(payload degradedPostureSummaryReceiptPayload) error {
	if payload.UserAcknowledged && strings.TrimSpace(payload.AcknowledgmentEvidence) == "" {
		return fmt.Errorf("acknowledgment_evidence is required when user_acknowledged=true")
	}
	if !payload.UserAcknowledged && strings.TrimSpace(payload.AcknowledgmentEvidence) != "" {
		return fmt.Errorf("acknowledgment_evidence requires user_acknowledged=true")
	}
	return nil
}

func validateDegradedPostureApprovalAndOverride(payload degradedPostureSummaryReceiptPayload) error {
	if payload.ApprovalConsumed && !payload.ApprovalRequired {
		return fmt.Errorf("approval_consumed=true requires approval_required=true")
	}
	if payload.ApprovalConsumed && strings.TrimSpace(payload.ApprovalConsumptionLink) == "" {
		return fmt.Errorf("approval_consumption_link is required when approval_consumed=true")
	}
	if payload.OverrideApplied && !payload.OverrideRequired {
		return fmt.Errorf("override_applied=true requires override_required=true")
	}
	if payload.OverrideApplied && strings.TrimSpace(payload.OverridePolicyDecisionRef) == "" {
		return fmt.Errorf("override_policy_decision_ref is required when override_applied=true")
	}
	return nil
}

func validateDegradedPostureOptionalDigests(payload degradedPostureSummaryReceiptPayload) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "approval_policy_decision_ref", value: payload.ApprovalPolicyDecisionRef},
		{name: "approval_consumption_link", value: payload.ApprovalConsumptionLink},
		{name: "override_policy_decision_ref", value: payload.OverridePolicyDecisionRef},
		{name: "override_action_request_hash", value: payload.OverrideActionRequestHash},
	} {
		if err := validateOptionalDigestIdentityString(field.value, field.name); err != nil {
			return err
		}
	}
	return nil
}

func validateNegativeCapabilitySummaryReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaNegativeCapabilityV0 {
		return fmt.Errorf("%s receipts require negative capability summary payload schema", receipt.AuditReceiptKind)
	}
	payload := negativeCapabilitySummaryReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode negative capability summary payload: %w", err)
	}
	if strings.TrimSpace(payload.SummaryScopeKind) == "" {
		return fmt.Errorf("summary_scope_kind is required")
	}
	if payload.NoArtifactCrossedBoundary && strings.TrimSpace(payload.BoundaryRoute) == "" {
		return fmt.Errorf("boundary_route is required when no_artifact_crossed_boundary=true")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "secret_lease_evidence_support", value: payload.SecretLeaseEvidenceSupport},
		{name: "network_egress_evidence_support", value: payload.NetworkEgressEvidenceSupport},
		{name: "approval_consumption_evidence_support", value: payload.ApprovalConsumptionEvidenceSupport},
		{name: "boundary_crossing_evidence_support", value: payload.BoundaryCrossingEvidenceSupport},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
		if !slices.Contains([]string{"explicit", "limited"}, strings.TrimSpace(field.value)) {
			return fmt.Errorf("%s must be explicit or limited", field.name)
		}
	}
	return nil
}

func validateMetaAuditActionReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaMetaAuditActionV0 {
		return fmt.Errorf("%s receipts require meta-audit action payload schema", receipt.AuditReceiptKind)
	}
	payload := metaAuditActionReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode meta-audit action payload: %w", err)
	}
	if err := validateMetaAuditActionCore(receipt.AuditReceiptKind, payload); err != nil {
		return err
	}
	if err := validateMetaAuditActionOptionalDigests(payload); err != nil {
		return err
	}
	if err := validateMetaAuditActionOperator(payload.Operator); err != nil {
		return err
	}
	if strings.TrimSpace(payload.SensitiveViewClass) != "" && strings.TrimSpace(payload.ActionCode) != auditReceiptKindSensitiveEvidenceView {
		return fmt.Errorf("sensitive_view_class is only valid for %s", auditReceiptKindSensitiveEvidenceView)
	}
	return nil
}
