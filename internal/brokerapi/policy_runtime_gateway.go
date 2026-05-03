package brokerapi

import (
	"encoding/json"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	gatewayPolicyDecisionSchemaID      = "runecode.protocol.v0.PolicyDecision"
	gatewayPolicyDecisionSchemaVersion = "0.3.0"
	gatewayPolicyDetailsSchemaID       = "runecode.protocol.details.policy.evaluation.v0"
	gatewayRuntimeMaxTimeoutSeconds    = 300
	gatewayRuntimeMaxResponseBytes     = 16 * 1024 * 1024
	gatewayDNSLookupTimeout            = 2 * time.Second
)

type gatewayActionPayloadRuntime struct {
	GatewayRoleKind string                      `json:"gateway_role_kind"`
	DestinationKind string                      `json:"destination_kind"`
	DestinationRef  string                      `json:"destination_ref"`
	EgressDataClass string                      `json:"egress_data_class"`
	Operation       string                      `json:"operation"`
	TimeoutSeconds  *int                        `json:"timeout_seconds,omitempty"`
	PayloadHash     *trustpolicy.Digest         `json:"payload_hash,omitempty"`
	AuditContext    *gatewayAuditContextPayload `json:"audit_context,omitempty"`
	GitRequest      map[string]any              `json:"git_request,omitempty"`
	GitRuntimeProof *gitRuntimeProofPayload     `json:"git_runtime_proof,omitempty"`
	QuotaContext    *gatewayQuotaContextPayload `json:"quota_context,omitempty"`
}

func (r policyRuntime) enforceGatewayRuntime(runID string, compiled *policyengine.CompiledContext, action policyengine.ActionRequest, decision policyengine.PolicyDecision) policyengine.PolicyDecision {
	if !gatewayRuntimeEnforcementEligible(r, action, decision) {
		return decision
	}

	payload, err := decodeGatewayRuntimePayload(action.ActionPayload)
	if err != nil {
		return runtimeGatewayDenyDecision(compiled, decision, payload, "runtime_gateway_payload_decode_failed", map[string]any{"error": err.Error()})
	}

	entry, match, found, reason := findMatchingGatewayAllowlistEntry(compiled, payload)
	if !found {
		if reason == "" {
			reason = "runtime_gateway_destination_not_allowlisted"
		}
		r.service.persistProviderInvocationReceipt(runID, string(policyengine.DecisionDeny), reason, payload, gatewayAllowlistMatch{})
		return runtimeGatewayDenyDecision(compiled, decision, payload, reason, nil)
	}

	if reason, details, denied := r.service.gatewayRuntime.runtimeEnforcementDenyReason(runID, entry, payload); denied {
		r.service.persistProviderInvocationReceipt(runID, string(policyengine.DecisionDeny), reason, payload, match)
		return runtimeGatewayDenyDecision(compiled, decision, payload, reason, details)
	}
	if reason, details, denied := runtimeGitOutboundVerificationReason(payload); denied {
		r.service.gatewayRuntime.releaseQuotaUsage(runID, payload)
		r.service.persistProviderInvocationReceipt(runID, string(policyengine.DecisionDeny), reason, payload, match)
		return runtimeGatewayDenyDecision(compiled, decision, payload, reason, details)
	}

	if err := r.service.gatewayRuntime.emitGatewayAuditEvent(runID, decision, payload, match); err != nil {
		r.service.gatewayRuntime.releaseQuotaUsage(runID, payload)
		r.service.persistProviderInvocationReceipt(runID, string(policyengine.DecisionDeny), "runtime_gateway_audit_emit_failed", payload, match)
		return runtimeGatewayDenyDecision(compiled, decision, payload, "runtime_gateway_audit_emit_failed", map[string]any{"error": err.Error()})
	}
	r.service.persistProviderInvocationReceipt(runID, string(decision.DecisionOutcome), "", payload, match)
	r.service.gatewayRuntime.releaseQuotaUsage(runID, payload)
	return decision
}

func gatewayRuntimeEnforcementEligible(r policyRuntime, action policyengine.ActionRequest, decision policyengine.PolicyDecision) bool {
	if r.service == nil || r.service.gatewayRuntime == nil {
		return false
	}
	if decision.DecisionOutcome != policyengine.DecisionAllow {
		return false
	}
	return action.ActionKind == policyengine.ActionKindGatewayEgress || action.ActionKind == policyengine.ActionKindDependencyFetch
}

func decodeGatewayRuntimePayload(raw map[string]any) (gatewayActionPayloadRuntime, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return gatewayActionPayloadRuntime{}, err
	}
	var payload gatewayActionPayloadRuntime
	if err := json.Unmarshal(b, &payload); err != nil {
		return gatewayActionPayloadRuntime{}, err
	}
	return payload, nil
}

func findMatchingGatewayAllowlistEntry(compiled *policyengine.CompiledContext, payload gatewayActionPayloadRuntime) (policyengine.GatewayScopeRule, gatewayAllowlistMatch, bool, string) {
	for _, ref := range compiled.Context.ActiveAllowlistRefs {
		allowlist, ok := compiled.AllowlistsByHash[ref]
		if !ok {
			continue
		}
		for _, entry := range allowlist.Entries {
			if gatewayAllowlistEntryMatchesRuntimePayload(entry, payload) {
				return entry, gatewayAllowlistMatch{AllowlistRef: ref, EntryID: entry.EntryID}, true, ""
			}
		}
	}
	return policyengine.GatewayScopeRule{}, gatewayAllowlistMatch{}, false, "runtime_gateway_destination_not_allowlisted"
}

func gatewayAllowlistEntryMatchesRuntimePayload(entry policyengine.GatewayScopeRule, payload gatewayActionPayloadRuntime) bool {
	if entry.ScopeKind != "gateway_destination" {
		return false
	}
	if entry.GatewayRoleKind != "" && entry.GatewayRoleKind != payload.GatewayRoleKind {
		return false
	}
	if entry.Destination.DescriptorKind != payload.DestinationKind {
		return false
	}
	if !runtimeDestinationRefMatches(entry.Destination, payload.DestinationRef) {
		return false
	}
	return containsStringValue(entry.PermittedOperations, payload.Operation)
}

func runtimeGatewayHardeningReason(entry policyengine.GatewayScopeRule, payload gatewayActionPayloadRuntime) string {
	if reason := runtimeDestinationHardeningReason(entry.Destination, entry.RedirectPosture); reason != "" {
		return reason
	}
	if isGatewayRequestExecutionOperation(payload.Operation) {
		return runtimeRequestExecutionHardeningReason(entry, payload)
	}
	return ""
}

func runtimeDestinationHardeningReason(destination policyengine.DestinationDescriptor, redirectPosture string) string {
	if !destination.TLSRequired {
		return "runtime_gateway_tls_required"
	}
	if destination.PrivateRangeBlocking != "enforced" {
		return "runtime_gateway_private_range_blocking_required"
	}
	if destination.DNSRebindingProtection != "enforced" {
		return "runtime_gateway_dns_rebinding_protection_required"
	}
	if redirectPosture != "deny" && redirectPosture != "allowlist_only" {
		return "runtime_gateway_unknown_redirect_posture"
	}
	return ""
}

func runtimeRequestExecutionHardeningReason(entry policyengine.GatewayScopeRule, payload gatewayActionPayloadRuntime) string {
	if entry.MaxTimeoutSeconds == nil || *entry.MaxTimeoutSeconds < 1 || *entry.MaxTimeoutSeconds > gatewayRuntimeMaxTimeoutSeconds {
		return "runtime_gateway_allowlist_timeout_invalid"
	}
	if payload.TimeoutSeconds == nil || *payload.TimeoutSeconds < 1 || *payload.TimeoutSeconds > gatewayRuntimeMaxTimeoutSeconds {
		return "runtime_gateway_timeout_invalid"
	}
	if *payload.TimeoutSeconds > *entry.MaxTimeoutSeconds {
		return "runtime_gateway_timeout_exceeds_allowlist"
	}
	if entry.MaxResponseBytes == nil || *entry.MaxResponseBytes < 1 || *entry.MaxResponseBytes > gatewayRuntimeMaxResponseBytes {
		return "runtime_gateway_response_size_limit_invalid"
	}
	return ""
}

func runtimeGatewayRoleSeparationReason(payload gatewayActionPayloadRuntime) string {
	if payload.GatewayRoleKind != "model-gateway" {
		return ""
	}
	if payload.DestinationKind == "auth_provider" {
		return "runtime_gateway_model_role_auth_provider_forbidden"
	}
	if payload.Operation == "exchange_auth_code" || payload.Operation == "refresh_auth_token" {
		return "runtime_gateway_model_role_auth_operation_forbidden"
	}
	return ""
}

func runtimeGatewayRuntimeDenyReason(entry policyengine.GatewayScopeRule, payload gatewayActionPayloadRuntime) (string, map[string]any, bool) {
	if reason := runtimeGatewayHardeningReason(entry, payload); reason != "" {
		return reason, nil, true
	}
	if reason := runtimeGatewayRoleSeparationReason(payload); reason != "" {
		return reason, nil, true
	}
	return "", nil, false
}

func runtimeGatewayDenyDecision(compiled *policyengine.CompiledContext, baseline policyengine.PolicyDecision, payload gatewayActionPayloadRuntime, reason string, extra map[string]any) policyengine.PolicyDecision {
	details := map[string]any{
		"precedence":        "runtime_gateway_execution",
		"invariant":         "network_egress_hard_boundary",
		"non_approvable":    true,
		"gateway_role_kind": payload.GatewayRoleKind,
		"destination_kind":  payload.DestinationKind,
		"destination_ref":   payload.DestinationRef,
		"operation":         payload.Operation,
		"reason":            reason,
	}
	if payload.QuotaContext != nil {
		details["quota_profile_kind"] = payload.QuotaContext.QuotaProfileKind
		details["quota_enforcement_at"] = payload.QuotaContext.Phase
	}
	for key, value := range extra {
		details[key] = value
	}
	return policyengine.PolicyDecision{
		SchemaID:               gatewayPolicyDecisionSchemaID,
		SchemaVersion:          gatewayPolicyDecisionSchemaVersion,
		DecisionOutcome:        policyengine.DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           compiled.ManifestHash,
		PolicyInputHashes:      append([]string{}, compiled.PolicyInputHashes...),
		ActionRequestHash:      baseline.ActionRequestHash,
		RelevantArtifactHashes: append([]string{}, baseline.RelevantArtifactHashes...),
		DetailsSchemaID:        gatewayPolicyDetailsSchemaID,
		Details:                details,
	}
}

func isGatewayRequestExecutionOperation(operation string) bool {
	switch operation {
	case "invoke_model", "fetch_dependency", "exchange_auth_code", "refresh_auth_token":
		return true
	default:
		return false
	}
}
