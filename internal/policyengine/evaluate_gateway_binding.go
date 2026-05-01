package policyengine

import "encoding/json"

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
		return missingGatewayPayloadHashReason(payload.Operation), nil
	}
	if payload.Operation == "fetch_dependency" {
		return dependencyRequestBindingDetails(payload)
	}
	return gatewayRequestBindingDetails(action, payload)
}

func missingGatewayPayloadHashReason(operation string) string {
	if operation == "invoke_model" {
		return "missing_payload_hash_for_canonical_llm_request_binding"
	}
	if operation == "fetch_dependency" {
		return "missing_payload_hash_for_canonical_dependency_request_binding"
	}
	return "missing_payload_hash_for_gateway_request_binding"
}

func dependencyRequestBindingDetails(payload gatewayEgressPayload) (string, map[string]any) {
	if payload.DependencyRequest == nil {
		return "missing_canonical_dependency_request_binding", nil
	}
	payloadHashIdentity, err := payload.PayloadHash.Identity()
	if err != nil {
		return "invalid_payload_hash_identity", nil
	}
	requestHashIdentity, requestKind, err := canonicalDependencyTypedRequestHash(payload.DependencyRequest)
	if err != nil {
		return "invalid_canonical_dependency_request_hash", nil
	}
	if payloadHashIdentity == requestHashIdentity {
		return "", nil
	}
	return "payload_hash_not_bound_to_canonical_dependency_request_hash", map[string]any{
		"payload_hash":                      payloadHashIdentity,
		"canonical_dependency_request_hash": requestHashIdentity,
		"dependency_request_kind":           requestKind,
	}
}

func gatewayRequestBindingDetails(action ActionRequest, payload gatewayEgressPayload) (string, map[string]any) {
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
	invariant := "typed_gateway_request_binding"
	if payload.Operation == "invoke_model" {
		invariant = "typed_model_request_binding"
	}
	if payload.Operation == "fetch_dependency" {
		invariant = "typed_dependency_request_binding"
	}
	details := map[string]any{
		"precedence":        "invariants_first",
		"invariant":         invariant,
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

func denyIfGitRemoteMutationPayloadHashUnbound(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload) (PolicyDecision, bool) {
	if !isGatewayRemoteMutationOperation(payload.Operation) {
		return PolicyDecision{}, false
	}
	if payload.Operation == "external_anchor_submit" {
		bindingReason, details := externalAnchorMutationBindingDetails(payload)
		if bindingReason == "" {
			return PolicyDecision{}, false
		}
		return denyExternalAnchorMutationBinding(compiled, action, actionHash, payload, bindingReason, details)
	}
	bindingReason, details := gitRemoteMutationBindingDetails(payload)
	if bindingReason == "" {
		return PolicyDecision{}, false
	}
	return denyGitRemoteMutationBinding(compiled, action, actionHash, payload, bindingReason, details)
}

func externalAnchorMutationBindingDetails(payload gatewayEgressPayload) (string, map[string]any) {
	if payload.PayloadHash == nil {
		return "missing_payload_hash_for_canonical_external_anchor_request_binding", nil
	}
	if payload.ExternalAnchorRequest == nil {
		return "missing_canonical_external_anchor_request_binding", nil
	}
	payloadHashIdentity, err := payload.PayloadHash.Identity()
	if err != nil {
		return "invalid_payload_hash_identity", nil
	}
	requestHashIdentity, requestKind, err := canonicalExternalAnchorTypedRequestHash(payload.ExternalAnchorRequest)
	if err != nil {
		return "invalid_canonical_external_anchor_request_hash", nil
	}
	if payloadHashIdentity == requestHashIdentity {
		return "", nil
	}
	return "payload_hash_not_bound_to_canonical_external_anchor_request_hash", map[string]any{
		"payload_hash":                           payloadHashIdentity,
		"canonical_external_anchor_request_hash": requestHashIdentity,
		"external_anchor_request_kind":           requestKind,
	}
}

func canonicalExternalAnchorTypedRequestHash(request map[string]any) (string, string, error) {
	requestKind, _ := request["request_kind"].(string)
	b, err := json.Marshal(request)
	if err != nil {
		return "", "", err
	}
	hash, err := CanonicalHashBytes(b)
	if err != nil {
		return "", "", err
	}
	return hash, requestKind, nil
}

func gitRemoteMutationBindingDetails(payload gatewayEgressPayload) (string, map[string]any) {
	if payload.PayloadHash == nil {
		return "missing_payload_hash_for_canonical_git_request_binding", nil
	}
	if payload.GitRequest == nil {
		return "missing_canonical_git_request_binding", nil
	}
	payloadHashIdentity, err := payload.PayloadHash.Identity()
	if err != nil {
		return "invalid_payload_hash_identity", nil
	}
	requestHashIdentity, requestKind, err := canonicalGitTypedRequestHash(payload.GitRequest)
	if err != nil {
		return "invalid_canonical_git_request_hash", nil
	}
	if payloadHashIdentity == requestHashIdentity {
		return "", nil
	}
	return "payload_hash_not_bound_to_canonical_git_request_hash", map[string]any{
		"payload_hash":               payloadHashIdentity,
		"canonical_git_request_hash": requestHashIdentity,
		"git_request_kind":           requestKind,
	}
}

func canonicalGitTypedRequestHash(request map[string]any) (string, string, error) {
	requestKind, _ := request["request_kind"].(string)
	b, err := json.Marshal(request)
	if err != nil {
		return "", "", err
	}
	hash, err := CanonicalHashBytes(b)
	if err != nil {
		return "", "", err
	}
	return hash, requestKind, nil
}

func canonicalDependencyTypedRequestHash(request map[string]any) (string, string, error) {
	requestKind, _ := request["request_kind"].(string)
	b, err := json.Marshal(request)
	if err != nil {
		return "", "", err
	}
	hash, err := CanonicalHashBytes(b)
	if err != nil {
		return "", "", err
	}
	return hash, requestKind, nil
}

func denyGitRemoteMutationBinding(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload, reason string, extra map[string]any) (PolicyDecision, bool) {
	details := map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "typed_git_request_binding",
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

func denyExternalAnchorMutationBinding(compiled *CompiledContext, action ActionRequest, actionHash string, payload gatewayEgressPayload, reason string, extra map[string]any) (PolicyDecision, bool) {
	details := map[string]any{
		"precedence":        "invariants_first",
		"invariant":         "typed_external_anchor_request_binding",
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
