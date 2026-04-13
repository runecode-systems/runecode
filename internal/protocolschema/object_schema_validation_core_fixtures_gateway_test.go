package protocolschema

func validPolicyAllowlist() map[string]any {
	return map[string]any{
		"schema_id":       "runecode.protocol.v0.PolicyAllowlist",
		"schema_version":  "0.1.0",
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": "runecode.protocol.v0.GatewayScopeRule",
		"entries": []any{
			validGatewayScopeRule("provider-a"),
			validGatewayScopeRule("provider-b"),
		},
	}
}

func invalidPolicyAllowlistKind() map[string]any {
	allowlist := validPolicyAllowlist()
	allowlist["allowlist_kind"] = "gateway_destination"
	return allowlist
}

func invalidPolicyAllowlistEntrySchemaID() map[string]any {
	allowlist := validPolicyAllowlist()
	allowlist["entry_schema_id"] = "runecode.protocol.v0.DestinationDescriptor"
	return allowlist
}

func validGatewayScopeRule(provider string) map[string]any {
	return map[string]any{
		"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":              "0.1.0",
		"scope_kind":                  "gateway_destination",
		"gateway_role_kind":           "model-gateway",
		"destination":                 validDestinationDescriptor(provider),
		"permitted_operations":        []any{"invoke_model"},
		"allowed_egress_data_classes": []any{"spec_text"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
}

func invalidGatewayScopeRuleKind() map[string]any {
	rule := validGatewayScopeRule("provider-a")
	rule["scope_kind"] = "gateway_destination_legacy"
	return rule
}

func validDestinationDescriptor(provider string) map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
		"schema_version":           "0.1.0",
		"descriptor_kind":          "model_endpoint",
		"canonical_host":           provider + ".example.com",
		"provider_or_namespace":    provider,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}

func invalidDestinationDescriptorKind() map[string]any {
	descriptor := validDestinationDescriptor("provider-a")
	descriptor["descriptor_kind"] = "raw_url"
	return descriptor
}

func validActionPayloadGatewayEgressRequestOperation() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.ActionPayloadGatewayEgress",
		"schema_version":    "0.1.0",
		"gateway_role_kind": "model-gateway",
		"destination_kind":  "model_endpoint",
		"destination_ref":   "provider-a.example.com/v1",
		"egress_data_class": "spec_text",
		"operation":         "invoke_model",
		"payload_hash":      testDigestValue("8"),
		"audit_context": map[string]any{
			"schema_id":            "runecode.protocol.v0.GatewayAuditContext",
			"schema_version":       "0.1.0",
			"outbound_bytes":       1024,
			"started_at":           "2026-03-13T12:00:00Z",
			"completed_at":         "2026-03-13T12:00:01Z",
			"outcome":              "succeeded",
			"request_hash":         testDigestValue("8"),
			"response_hash":        testDigestValue("7"),
			"lease_id":             "lease-model-1",
			"policy_decision_hash": testDigestValue("6"),
		},
		"quota_context": map[string]any{
			"schema_id":             "runecode.protocol.v0.GatewayQuotaContext",
			"schema_version":        "0.1.0",
			"quota_profile_kind":    "token_metered_api",
			"phase":                 "admission",
			"enforce_during_stream": false,
			"meters": map[string]any{
				"input_tokens":  512,
				"output_tokens": 128,
			},
		},
	}
}

func validActionPayloadGatewayEgressRequestOperationWithPortAndPath() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["destination_ref"] = "provider-a.example.com:8443/v1/chat/completions"
	return payload
}

func validActionPayloadGatewayEgressScopeOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["operation"] = "expand_scope"
	delete(payload, "payload_hash")
	return payload
}

func validActionPayloadGatewayEgressRequestOperationWithTimeout() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["timeout_seconds"] = 60
	return payload
}

func validActionPayloadGatewayEgressStreamQuotaOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["phase"] = "stream"
	quota["enforce_during_stream"] = true
	quota["stream_limit_bytes"] = 2048
	meters := quota["meters"].(map[string]any)
	meters["streamed_bytes"] = 1024
	return payload
}

func invalidActionPayloadGatewayEgressUnknownOperation() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["operation"] = "provider_specific_passthrough"
	return payload
}

func invalidActionPayloadGatewayEgressRequestMissingPayloadHash() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	delete(payload, "payload_hash")
	return payload
}

func invalidActionPayloadGatewayEgressDependencyRequestMissingPayloadHash() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["gateway_role_kind"] = "dependency-fetch"
	payload["destination_kind"] = "package_registry"
	payload["operation"] = "fetch_dependency"
	delete(payload, "payload_hash")
	return payload
}

func invalidActionPayloadGatewayEgressScopeWithPayloadHash() map[string]any {
	payload := validActionPayloadGatewayEgressScopeOperation()
	payload["payload_hash"] = testDigestValue("9")
	return payload
}

func invalidActionPayloadGatewayEgressRawURLDestinationRef() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["destination_ref"] = "https://provider-a.example.com/v1/chat/completions?model=test#frag"
	return payload
}

func invalidActionPayloadGatewayEgressTimeoutOutOfBounds() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["timeout_seconds"] = 301
	return payload
}

func invalidActionPayloadGatewayEgressMissingAuditContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	delete(payload, "audit_context")
	return payload
}

func invalidActionPayloadGatewayEgressMissingQuotaContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	delete(payload, "quota_context")
	return payload
}

func invalidActionPayloadGatewayEgressStreamPhaseWithoutLimit() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["phase"] = "stream"
	quota["enforce_during_stream"] = true
	delete(quota, "stream_limit_bytes")
	return payload
}

func invalidActionPayloadGatewayEgressAuthRequestMissingAuditContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["gateway_role_kind"] = "auth-gateway"
	payload["destination_kind"] = "auth_provider"
	payload["operation"] = "exchange_auth_code"
	delete(payload, "audit_context")
	return payload
}

func invalidActionPayloadGatewayEgressAuthRequestMissingQuotaContext() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	payload["gateway_role_kind"] = "auth-gateway"
	payload["destination_kind"] = "auth_provider"
	payload["operation"] = "refresh_auth_token"
	delete(payload, "quota_context")
	return payload
}

func invalidActionPayloadGatewayEgressHybridQuotaMissingEntitlementMeter() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["quota_profile_kind"] = "hybrid"
	meters := quota["meters"].(map[string]any)
	delete(meters, "request_units")
	delete(meters, "entitlement_units")
	meters["input_tokens"] = 256
	return payload
}

func invalidActionPayloadGatewayEgressHybridQuotaMissingTokenMeter() map[string]any {
	payload := validActionPayloadGatewayEgressRequestOperation()
	quota := payload["quota_context"].(map[string]any)
	quota["quota_profile_kind"] = "hybrid"
	meters := quota["meters"].(map[string]any)
	delete(meters, "input_tokens")
	delete(meters, "output_tokens")
	meters["request_units"] = 1
	return payload
}

func invalidGatewayScopeRuleTimeoutOutOfBounds() map[string]any {
	rule := validGatewayScopeRule("provider-a")
	rule["max_timeout_seconds"] = 301
	return rule
}
