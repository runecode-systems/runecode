package policyengine

func denyIfModelInvokePayloadHashUnbound(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if !isGatewayRequestExecutionOperation(payload.Operation) {
		return PolicyDecision{}, false
	}
	bindingReason, details := modelRequestBindingDetails(action, payload)
	if bindingReason != "" {
		return denyModelRequestBinding(compiled, action, actionHash, payload, bindingReason, details)
	}
	return PolicyDecision{}, false
}

func modelRequestBindingDetails(action ActionRequest, payload gatewayEgressPayload) (string, map[string]any) {
	if payload.PayloadHash == nil {
		if payload.Operation == "invoke_model" {
			return "missing_payload_hash_for_canonical_llm_request_binding", nil
		}
		return "missing_payload_hash_for_gateway_request_binding", nil
	}
	requiredHashes := actionRelevantArtifactHashes(action)
	if len(requiredHashes) == 0 {
		if payload.Operation == "invoke_model" {
			return "missing_canonical_llm_request_hash_binding", nil
		}
		return "missing_canonical_gateway_request_hash_binding", nil
	}
	payloadHashIdentity, err := payload.PayloadHash.Identity()
	if err != nil {
		return "invalid_payload_hash_identity", nil
	}
	if containsString(requiredHashes, payloadHashIdentity) {
		return "", nil
	}
	reason := "payload_hash_not_bound_to_canonical_gateway_request_hash"
	if payload.Operation == "invoke_model" {
		reason = "payload_hash_not_bound_to_canonical_llm_request_hash"
	}
	return reason, map[string]any{
		"payload_hash":              payloadHashIdentity,
		"required_canonical_hashes": requiredHashes,
	}
}

func denyModelRequestBinding(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload, reason string, extra map[string]any) (PolicyDecision, bool) {
	details := map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "typed_model_request_binding",
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
