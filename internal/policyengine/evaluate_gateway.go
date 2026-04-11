package policyengine

import "strings"

func evaluateGatewayBoundary(compiled *CompiledContext, action ActionRequest, actionHash string) (PolicyDecision, bool) {
	if decision, blocked := denyIfInvalidGatewayFamily(compiled, action, actionHash); blocked {
		return decision, true
	}
	payload, decision, blocked := decodeAndValidateGatewayPayload(compiled, action, actionHash)
	if blocked {
		return decision, true
	}
	if decision, blocked := denyIfDependencySplitViolation(compiled, action, actionHash, payload); blocked {
		return decision, true
	}
	if decision, blocked := denyIfUnknownGatewayDestination(compiled, action, actionHash, payload); blocked {
		return decision, true
	}
	if decision, blocked := denyIfGatewayRoleMismatch(compiled, action, actionHash, payload); blocked {
		return decision, true
	}
	if decision, blocked := denyIfDestinationNotAllowlisted(compiled, action, actionHash, payload); blocked {
		return decision, true
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
	if gatewayDestinationAllowedBySignedAllowlists(compiled, action, payload) {
		return PolicyDecision{}, false
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
	if compiled.Context.ActiveRoleKind == "workspace-read" || compiled.Context.ActiveRoleKind == "workspace-edit" || compiled.Context.ActiveRoleKind == "workspace-test" {
		return denyInvariantDecision(compiled, action, actionHash, map[string]any{
			"precedence":                    "invariants_first",
			"invariant":                     "network_egress_hard_boundary",
			"non_approvable":                true,
			"workspace_role_kind":           compiled.Context.ActiveRoleKind,
			"workspace_offline_only":        true,
			"required_cross_boundary_route": "artifact_io",
			"artifact_route_actions":        []string{ActionKindArtifactRead},
		}), true
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

func requiredGatewayRoleForDestination(destinationKind string) (string, bool) {
	requiredRoleForDestination := map[string]string{
		"model_endpoint":   "model-gateway",
		"auth_provider":    "auth-gateway",
		"git_remote":       "git-gateway",
		"web_origin":       "web-research",
		"package_registry": "dependency-fetch",
	}
	requiredRole, ok := requiredRoleForDestination[destinationKind]
	return requiredRole, ok
}

func gatewayDestinationAllowedBySignedAllowlists(compiled *CompiledContext, action ActionRequest, payload gatewayEgressPayload) bool {
	refs := compiled.Context.ActiveAllowlistRefs
	for _, ref := range refs {
		allowlist, ok := compiled.AllowlistsByHash[ref]
		if !ok {
			continue
		}
		for _, entry := range allowlist.Entries {
			if gatewayScopeEntryMatchesPayload(entry, payload) {
				return true
			}
		}
	}
	return false
}

func gatewayScopeEntryMatchesPayload(entry GatewayScopeRule, payload gatewayEgressPayload) bool {
	if entry.ScopeKind != "gateway_destination" {
		return false
	}
	if payload.Operation == "" {
		return false
	}
	if entry.GatewayRoleKind != "" && entry.GatewayRoleKind != payload.GatewayRoleKind {
		return false
	}
	if entry.Destination.DescriptorKind != payload.DestinationKind {
		return false
	}
	if !destinationRefMatches(entry.Destination, payload.DestinationRef) {
		return false
	}
	if !containsString(entry.PermittedOperations, payload.Operation) {
		return false
	}
	return containsString(entry.AllowedEgressDataClasses, payload.EgressDataClass)
}
