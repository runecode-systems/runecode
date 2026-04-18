package policyengine

import (
	"strings"
)

const (
	gatewayMaxTimeoutSecondsHardLimit   = 300
	gatewayMaxResponseBytesHardLimit    = 16 * 1024 * 1024
	gatewayRedirectPostureDeny          = "deny"
	gatewayRedirectPostureAllowlistOnly = "allowlist_only"
)

func evaluateGatewayBoundary(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	payload, decision, blocked := validateGatewayBoundaryInput(compiled, action, actionHash)
	if blocked {
		return decision, true
	}
	return evaluateGatewayBoundaryInvariants(compiled, action, actionHash, payload)
}

func validateGatewayBoundaryInput(compiled *CompiledContext, action ActionRequest, actionHash string) (gatewayEgressPayload, PolicyDecision, bool) {
	if decision, blocked := denyIfInvalidGatewayFamily(compiled, action, actionHash); blocked {
		return gatewayEgressPayload{}, decision, true
	}
	payload, decision, blocked := decodeAndValidateGatewayPayload(compiled, action, actionHash)
	if blocked {
		return gatewayEgressPayload{}, decision, true
	}
	return payload, PolicyDecision{}, false
}

func evaluateGatewayBoundaryInvariants(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	checks := []func(*CompiledContext, ActionRequest, string, gatewayEgressPayload) (PolicyDecision, bool){
		denyIfDependencySplitViolation,
		denyIfModelGatewayUsesAuthProviderFlow,
		denyIfModelGatewayUsesAuthOperations,
		denyIfModelInvokePayloadHashUnbound,
		denyIfGitRemoteMutationPayloadHashUnbound,
		denyIfDisallowedGatewayEgressDataClass,
		denyIfGatewayAuditBindingsInvalid,
		denyIfGatewayQuotaContextInvalid,
		denyIfUnknownGatewayDestination,
		denyIfGatewayRoleMismatch,
		denyIfDestinationNotAllowlisted,
	}
	for _, check := range checks {
		if decision, blocked := check(compiled, action, actionHash, payload); blocked {
			return decision, true
		}
	}
	return PolicyDecision{}, false
}

func denyIfUnknownGatewayDestination(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	_, known := requiredGatewayRoleForDestination(payload.DestinationKind)
	if known {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "invariants_first",
		"invariant":        "network_egress_hard_boundary",
		"non_approvable":   true,
		"destination_kind": payload.DestinationKind,
		"reason":           "unknown_destination_kind_fail_closed",
	}), true
}

func denyIfGatewayRoleMismatch(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	requiredRole, _ := requiredGatewayRoleForDestination(payload.DestinationKind)
	if payload.GatewayRoleKind == requiredRole {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":            "invariants_first",
		"invariant":             "network_egress_hard_boundary",
		"non_approvable":        true,
		"destination_kind":      payload.DestinationKind,
		"required_gateway_role": requiredRole,
		"gateway_role_kind":     payload.GatewayRoleKind,
	}), true
}

func denyIfDestinationNotAllowlisted(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	allowed, denyDetails := gatewayDestinationAllowedBySignedAllowlists(compiled, payload)
	if allowed {
		return PolicyDecision{}, false
	}
	if denyDetails != nil {
		return denyInvariantDecision(compiled, action, actionHash, denyDetails), true
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":       "allowlist_active_manifest_set",
		"invariant":        "network_egress_hard_boundary",
		"non_approvable":   true,
		"destination_kind": payload.DestinationKind,
		"destination_ref":  payload.DestinationRef,
		"reason":           "destination_not_allowlisted",
	}), true
}

func denyIfInvalidGatewayFamily(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if compiled.Context.ActiveRoleFamily != "gateway" {
		details := map[string]any{
			"precedence":             "invariants_first",
			"invariant":              "network_egress_hard_boundary",
			"non_approvable":         true,
			"action_kind":            action.ActionKind,
			"required_role_family":   "gateway",
			"active_role_family":     compiled.Context.ActiveRoleFamily,
			"workspace_offline_only": true,
		}
		if compiled.Context.ActiveRoleFamily == "workspace" {
			details["required_cross_boundary_route"] = "artifact_io"
			details["artifact_route_actions"] = []string{ActionKindArtifactRead}
		}
		return denyInvariantDecision(compiled, action, actionHash, details), true
	}
	return PolicyDecision{}, false
}

func decodeAndValidateGatewayPayload(compiled *CompiledContext, action ActionRequest, actionHash string) (gatewayEgressPayload, PolicyDecision, bool) {
	payload := gatewayEgressPayload{}
	if err := decodeActionPayload(action.ActionPayload, &payload); err != nil {
		return gatewayEgressPayload{}, denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":        "invariants_first",
			"invariant":         "deny_by_default_network",
			"non_approvable":    true,
			"payload_parse_err": err.Error(),
		}), true
	}
	if payload.GatewayRoleKind != compiled.Context.ActiveRoleKind {
		return gatewayEgressPayload{}, denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":                "invariants_first",
			"invariant":                 "no_escalation_in_place",
			"non_approvable":            true,
			"payload_gateway_role_kind": payload.GatewayRoleKind,
			"active_context_role_kind":  compiled.Context.ActiveRoleKind,
		}), true
	}
	if strings.TrimSpace(payload.Operation) == "" {
		return gatewayEgressPayload{}, denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":     "invariants_first",
			"invariant":      "network_egress_hard_boundary",
			"non_approvable": true,
			"reason":         "missing_gateway_operation",
		}), true
	}
	return payload, PolicyDecision{}, false
}

func denyIfDependencySplitViolation(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if action.ActionKind == ActionKindDependencyFetch && payload.DestinationKind != "package_registry" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":       "invariants_first",
			"invariant":        "dependency_behavior_split",
			"non_approvable":   true,
			"action_kind":      ActionKindDependencyFetch,
			"destination_kind": payload.DestinationKind,
		}), true
	}
	if action.ActionKind == ActionKindGatewayEgress && payload.DestinationKind == "package_registry" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":           "invariants_first",
			"invariant":            "dependency_behavior_split",
			"non_approvable":       true,
			"action_kind":          ActionKindGatewayEgress,
			"required_action_kind": ActionKindDependencyFetch,
		}), true
	}
	return PolicyDecision{}, false
}

func denyIfModelGatewayUsesAuthProviderFlow(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if payload.GatewayRoleKind != "model-gateway" {
		return PolicyDecision{}, false
	}
	if payload.DestinationKind != "auth_provider" {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "gateway_role_separation",
		"non_approvable":    true,
		"gateway_role_kind": payload.GatewayRoleKind,
		"destination_kind":  payload.DestinationKind,
		"reason":            "model_gateway_cannot_perform_auth_provider_exchange_or_refresh",
	}), true
}

func denyIfModelGatewayUsesAuthOperations(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if payload.GatewayRoleKind != "model-gateway" {
		return PolicyDecision{}, false
	}
	if payload.Operation != "exchange_auth_code" && payload.Operation != "refresh_auth_token" {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "gateway_role_separation",
		"non_approvable":    true,
		"gateway_role_kind": payload.GatewayRoleKind,
		"operation":         payload.Operation,
		"reason":            "model_gateway_cannot_perform_auth_exchange_or_refresh_operations",
	}), true
}

func denyIfDisallowedGatewayEgressDataClass(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if payload.GatewayRoleKind != "model-gateway" {
		return PolicyDecision{}, false
	}
	if payload.EgressDataClass != "unapproved_file_excerpts" {
		return PolicyDecision{}, false
	}
	return denyInvariantDecision(compiled, action, actionHash, map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "network_egress_hard_boundary",
		"non_approvable":    true,
		"gateway_role_kind": payload.GatewayRoleKind,
		"destination_kind":  payload.DestinationKind,
		"operation":         payload.Operation,
		"egress_data_class": payload.EgressDataClass,
		"reason":            "disallowed_egress_data_class",
	}), true
}
