package policyengine

import "strings"

func gatewayDestinationAllowedBySignedAllowlists(compiled *CompiledContext, payload gatewayEgressPayload) (bool, map[string]any) {
	state := gatewayAllowlistMatchState{}
	for _, ref := range compiled.Context.ActiveAllowlistRefs {
		allowlist, ok := compiled.AllowlistsByHash[ref]
		if !ok {
			continue
		}
		if evaluateGatewayAllowlistEntries(allowlist.Entries, payload, &state) {
			return true, nil
		}
	}
	if !state.hadMatch {
		return false, nil
	}
	if state.firstDataClassViolation != nil {
		return false, state.firstDataClassViolation
	}
	if state.firstHardeningViolation != nil {
		return false, state.firstHardeningViolation
	}
	return false, nil
}

type gatewayAllowlistMatchState struct {
	hadMatch                bool
	firstHardeningViolation map[string]any
	firstDataClassViolation map[string]any
}

func evaluateGatewayAllowlistEntries(entries []GatewayScopeRule, payload gatewayEgressPayload, state *gatewayAllowlistMatchState) bool {
	for _, entry := range entries {
		allowed, matched := gatewayEntryAllowsPayload(entry, payload)
		if !matched {
			continue
		}
		state.hadMatch = true
		if allowed {
			return true
		}
		captureGatewayEntryViolation(state, entry, payload)
	}
	return false
}

func gatewayEntryAllowsPayload(entry GatewayScopeRule, payload gatewayEgressPayload) (bool, bool) {
	if !gatewayScopeEntryCoreMatchesPayload(entry, payload) {
		return false, false
	}
	if !containsString(entry.AllowedEgressDataClasses, payload.EgressDataClass) {
		return false, true
	}
	if gatewayScopeEntryHardeningViolation(entry, payload) != nil {
		return false, true
	}
	return true, true
}

func captureGatewayEntryViolation(state *gatewayAllowlistMatchState, entry GatewayScopeRule, payload gatewayEgressPayload) {
	if state.firstDataClassViolation == nil && !containsString(entry.AllowedEgressDataClasses, payload.EgressDataClass) {
		state.firstDataClassViolation = gatewayDataClassViolationDetails(payload, entry.AllowedEgressDataClasses)
	}
	if state.firstHardeningViolation == nil {
		state.firstHardeningViolation = gatewayScopeEntryHardeningViolation(entry, payload)
	}
}

func gatewayScopeEntryCoreMatchesPayload(entry GatewayScopeRule, payload gatewayEgressPayload) bool {
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
	if payload.Operation == "external_anchor_submit" && !externalAnchorTargetDescriptorDigestAllowlisted(entry, payload) {
		return false
	}
	if !destinationRefMatches(entry.Destination, payload.DestinationRef) {
		return false
	}
	if !containsString(entry.PermittedOperations, payload.Operation) {
		return false
	}
	return true
}

func externalAnchorTargetDescriptorDigestAllowlisted(entry GatewayScopeRule, payload gatewayEgressPayload) bool {
	requiredIdentity, ok := externalAnchorTargetDescriptorDigestFromPayload(payload)
	if !ok {
		return false
	}
	for _, digest := range entry.ExternalAnchorTargetDescriptorDigests {
		identity, err := digest.Identity()
		if err != nil {
			continue
		}
		if strings.TrimSpace(identity) == strings.TrimSpace(requiredIdentity) {
			return true
		}
	}
	return false
}

func externalAnchorTargetDescriptorDigestFromPayload(payload gatewayEgressPayload) (string, bool) {
	if payload.ExternalAnchorRequest == nil {
		return "", false
	}
	request := externalAnchorSubmitRequest{}
	if err := decodeActionPayload(payload.ExternalAnchorRequest, &request); err != nil {
		return "", false
	}
	identity, err := request.TargetDescriptorDigest.Identity()
	if err != nil || strings.TrimSpace(identity) == "" {
		return "", false
	}
	return identity, true
}

func gatewayDataClassViolationDetails(payload gatewayEgressPayload, allowedDataClasses []string) map[string]any {
	details := map[string]any{
		"precedence":                  "allowlist_active_manifest_set",
		"invariant":                   "network_egress_hard_boundary",
		"non_approvable":              true,
		"destination_kind":            payload.DestinationKind,
		"destination_ref":             payload.DestinationRef,
		"operation":                   payload.Operation,
		"egress_data_class":           payload.EgressDataClass,
		"allowed_egress_data_classes": append([]string{}, allowedDataClasses...),
		"reason":                      "egress_data_class_not_allowlisted",
	}
	if payload.TimeoutSeconds != nil {
		details["timeout_seconds"] = *payload.TimeoutSeconds
	}
	return details
}

func gatewayScopeEntryHardeningViolation(entry GatewayScopeRule, payload gatewayEgressPayload) map[string]any {
	reason := gatewayDestinationHardeningReason(entry)
	if reason == "" && isGatewayRequestExecutionOperation(payload.Operation) {
		reason = gatewayRequestExecutionHardeningReason(entry, payload)
	}
	if reason == "" {
		return nil
	}
	return gatewayHardeningViolationDetails(payload, reason)
}

func gatewayDestinationHardeningReason(entry GatewayScopeRule) string {
	if !entry.Destination.TLSRequired {
		return "destination_tls_required_not_enforced"
	}
	if entry.Destination.PrivateRangeBlocking != "enforced" {
		return "destination_private_range_blocking_not_enforced"
	}
	if entry.Destination.DNSRebindingProtection != "enforced" {
		return "destination_dns_rebinding_protection_not_enforced"
	}
	if entry.RedirectPosture != gatewayRedirectPostureDeny && entry.RedirectPosture != gatewayRedirectPostureAllowlistOnly {
		return "unknown_redirect_posture_fail_closed"
	}
	return ""
}

func gatewayRequestExecutionHardeningReason(entry GatewayScopeRule, payload gatewayEgressPayload) string {
	if entry.MaxTimeoutSeconds == nil {
		return "allowlist_timeout_limit_missing"
	}
	if *entry.MaxTimeoutSeconds < 1 || *entry.MaxTimeoutSeconds > gatewayMaxTimeoutSecondsHardLimit {
		return "allowlist_timeout_limit_out_of_bounds"
	}
	if payload.TimeoutSeconds == nil {
		return "request_timeout_missing"
	}
	if *payload.TimeoutSeconds < 1 || *payload.TimeoutSeconds > gatewayMaxTimeoutSecondsHardLimit {
		return "request_timeout_out_of_bounds"
	}
	if *payload.TimeoutSeconds > *entry.MaxTimeoutSeconds {
		return "request_timeout_exceeds_allowlist_limit"
	}
	if entry.MaxResponseBytes == nil {
		return "allowlist_response_size_limit_missing"
	}
	if *entry.MaxResponseBytes < 1 || *entry.MaxResponseBytes > gatewayMaxResponseBytesHardLimit {
		return "allowlist_response_size_limit_out_of_bounds"
	}
	return ""
}

func gatewayHardeningViolationDetails(payload gatewayEgressPayload, reason string) map[string]any {
	details := map[string]any{
		"precedence":       "allowlist_active_manifest_set",
		"invariant":        "network_egress_hard_boundary",
		"non_approvable":   true,
		"destination_kind": payload.DestinationKind,
		"destination_ref":  payload.DestinationRef,
		"operation":        payload.Operation,
		"reason":           reason,
	}
	if payload.TimeoutSeconds != nil {
		details["timeout_seconds"] = *payload.TimeoutSeconds
	}
	return details
}
