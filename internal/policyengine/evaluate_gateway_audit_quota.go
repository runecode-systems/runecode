package policyengine

import (
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	gatewayAuditMaxDuration       = 10 * time.Minute
	gatewayAuditEarliestValidYear = 2020
	gatewayAuditLatestValidYear   = 2100
)

func denyIfGatewayAuditBindingsInvalid(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if !isGatewayRequestExecutionOperation(payload.Operation) {
		return PolicyDecision{}, false
	}
	if payload.AuditContext == nil {
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, "missing_gateway_audit_context", nil)
	}
	startedAt, completedAt, timestampReason := gatewayAuditTimestampBounds(*payload.AuditContext)
	if timestampReason != "" {
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, timestampReason, nil)
	}
	if completedAt.Before(startedAt) {
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, "gateway_audit_completed_before_started", nil)
	}
	bindingReason, bindingDetails := gatewayAuditHashBinding(payload.PayloadHash, payload.AuditContext)
	if bindingReason != "" {
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, bindingReason, bindingDetails)
	}
	if payload.AuditContext.Outcome == "succeeded" && payload.AuditContext.ResponseHash == nil {
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, "missing_gateway_audit_response_hash_on_success", nil)
	}
	return PolicyDecision{}, false
}

func denyIfGatewayQuotaContextInvalid(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if !isGatewayRequestExecutionOperation(payload.Operation) {
		return PolicyDecision{}, false
	}
	if payload.QuotaContext == nil {
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, "missing_gateway_quota_context", nil)
	}
	quotaReason := gatewayQuotaPhaseReason(*payload.QuotaContext, payload.Operation)
	if quotaReason != "" {
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, quotaReason, nil)
	}
	meterReason := gatewayQuotaMeterReason(*payload.QuotaContext)
	if meterReason != "" {
		extra := map[string]any{"quota_profile_kind": payload.QuotaContext.QuotaProfileKind}
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, meterReason, extra)
	}
	streamReason, streamDetails := gatewayQuotaStreamReason(*payload.QuotaContext)
	if streamReason != "" {
		if streamDetails == nil {
			streamDetails = map[string]any{}
		}
		streamDetails["quota_profile_kind"] = payload.QuotaContext.QuotaProfileKind
		streamDetails["quota_enforcement_at"] = payload.QuotaContext.Phase
		return denyGatewayAuditQuotaInvariant(compiled, action, actionHash, payload, streamReason, streamDetails)
	}
	return PolicyDecision{}, false
}

func denyGatewayAuditQuotaInvariant(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload, reason string, extra map[string]any) (PolicyDecision, bool) {
	details := map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "gateway_audit_and_quota_enforcement",
		"non_approvable":    true,
		"gateway_role_kind": payload.GatewayRoleKind,
		"destination_kind":  payload.DestinationKind,
		"operation":         payload.Operation,
		"reason":            reason,
	}
	for key, value := range extra {
		details[key] = value
	}
	return denyInvariantDecision(compiled, action, actionHash, details), true
}

func gatewayAuditTimestampBounds(audit gatewayAuditContext) (time.Time, time.Time, string) {
	startedAt, err := time.Parse(time.RFC3339, audit.StartedAt)
	if err != nil {
		return time.Time{}, time.Time{}, "invalid_gateway_audit_started_at"
	}
	completedAt, err := time.Parse(time.RFC3339, audit.CompletedAt)
	if err != nil {
		return time.Time{}, time.Time{}, "invalid_gateway_audit_completed_at"
	}
	if startedAt.Year() < gatewayAuditEarliestValidYear || completedAt.Year() < gatewayAuditEarliestValidYear {
		return time.Time{}, time.Time{}, "gateway_audit_timestamp_out_of_bounds"
	}
	if startedAt.Year() > gatewayAuditLatestValidYear || completedAt.Year() > gatewayAuditLatestValidYear {
		return time.Time{}, time.Time{}, "gateway_audit_timestamp_out_of_bounds"
	}
	if completedAt.Sub(startedAt) > gatewayAuditMaxDuration {
		return time.Time{}, time.Time{}, "gateway_audit_duration_exceeds_bounds"
	}
	return startedAt, completedAt, ""
}

func gatewayAuditHashBinding(payloadHash *trustpolicy.Digest, audit *gatewayAuditContext) (string, map[string]any) {
	if payloadHash == nil || audit == nil || audit.RequestHash == nil {
		return "missing_gateway_audit_request_hash_binding", nil
	}
	payloadHashIdentity, err := payloadHash.Identity()
	if err != nil {
		return "invalid_payload_hash_identity", nil
	}
	auditRequestIdentity, err := audit.RequestHash.Identity()
	if err != nil {
		return "invalid_gateway_audit_request_hash_identity", nil
	}
	if payloadHashIdentity == auditRequestIdentity {
		return "", nil
	}
	return "gateway_audit_request_hash_not_bound_to_payload_hash", map[string]any{
		"payload_hash":         payloadHashIdentity,
		"audit_request_hash":   auditRequestIdentity,
		"audit_outcome":        audit.Outcome,
		"audit_outbound_bytes": audit.OutboundBytes,
	}
}

func gatewayQuotaPhaseReason(quota gatewayQuotaContext, operation string) string {
	if quota.Phase == "stream" && operation != "invoke_model" {
		return "stream_quota_phase_requires_model_invoke_operation"
	}
	if quota.Phase == "stream" && !quota.EnforceDuringStream {
		return "stream_quota_phase_requires_stream_enforcement"
	}
	if quota.EnforceDuringStream && quota.StreamLimitBytes == nil {
		return "stream_enforcement_requires_stream_limit_bytes"
	}
	return ""
}

func gatewayQuotaMeterReason(quota gatewayQuotaContext) string {
	meters := quota.Meters
	if quota.QuotaProfileKind == "token_metered_api" && meters.InputTokens == nil && meters.OutputTokens == nil {
		return "token_metered_quota_profile_requires_token_meters"
	}
	if quota.QuotaProfileKind == "request_entitlement" && meters.RequestUnits == nil && meters.EntitlementUnits == nil {
		return "request_entitlement_quota_profile_requires_request_or_entitlement_meter"
	}
	if quota.QuotaProfileKind == "hybrid" {
		hasTokenMeter := meters.InputTokens != nil || meters.OutputTokens != nil
		hasEntitlementMeter := meters.RequestUnits != nil || meters.EntitlementUnits != nil
		if !hasTokenMeter || !hasEntitlementMeter {
			return "hybrid_quota_profile_requires_token_and_entitlement_meters"
		}
	}
	return ""
}

func gatewayQuotaStreamReason(quota gatewayQuotaContext) (string, map[string]any) {
	if quota.Phase != "stream" {
		return "", nil
	}
	if quota.Meters.StreamedBytes == nil {
		return "stream_quota_phase_requires_streamed_bytes_meter", nil
	}
	if quota.StreamLimitBytes == nil || *quota.Meters.StreamedBytes <= *quota.StreamLimitBytes {
		return "", nil
	}
	return "streamed_bytes_exceed_stream_limit", map[string]any{
		"streamed_bytes":     *quota.Meters.StreamedBytes,
		"stream_limit_bytes": *quota.StreamLimitBytes,
	}
}
