package policyengine

func NewGatewayEgressAction(input GatewayEgressActionInput) ActionRequest {
	payload := buildGatewayPayload(input)
	return buildActionRequest(ActionKindGatewayEgress, actionPayloadGatewaySchemaID, payload, input.ActionEnvelope)
}

func NewDependencyFetchAction(input GatewayEgressActionInput) ActionRequest {
	payload := buildGatewayPayload(input)
	return buildActionRequest(ActionKindDependencyFetch, actionPayloadGatewaySchemaID, payload, input.ActionEnvelope)
}

func buildGatewayPayload(input GatewayEgressActionInput) map[string]any {
	payload := map[string]any{
		"schema_id":         actionPayloadGatewaySchemaID,
		"schema_version":    "0.1.0",
		"gateway_role_kind": input.GatewayRoleKind,
		"destination_kind":  input.DestinationKind,
		"destination_ref":   input.DestinationRef,
		"egress_data_class": input.EgressDataClass,
		"operation":         input.Operation,
	}
	if input.TimeoutSeconds != nil {
		payload["timeout_seconds"] = *input.TimeoutSeconds
	}
	if input.PayloadHash != nil {
		payload["payload_hash"] = *input.PayloadHash
	}
	if input.AuditContext != nil {
		payload["audit_context"] = buildGatewayAuditPayload(*input.AuditContext)
	}
	if input.QuotaContext != nil {
		payload["quota_context"] = buildGatewayQuotaPayload(*input.QuotaContext)
	}
	return payload
}

func buildGatewayAuditPayload(input GatewayAuditContextInput) map[string]any {
	audit := map[string]any{
		"schema_id":      "runecode.protocol.v0.GatewayAuditContext",
		"schema_version": "0.1.0",
		"outbound_bytes": input.OutboundBytes,
		"started_at":     input.StartedAt,
		"completed_at":   input.CompletedAt,
		"outcome":        input.Outcome,
	}
	if input.RequestHash != nil {
		audit["request_hash"] = *input.RequestHash
	}
	if input.ResponseHash != nil {
		audit["response_hash"] = *input.ResponseHash
	}
	if input.LeaseID != "" {
		audit["lease_id"] = input.LeaseID
	}
	if input.PolicyDecisionHash != nil {
		audit["policy_decision_hash"] = *input.PolicyDecisionHash
	}
	return audit
}

func buildGatewayQuotaPayload(input GatewayQuotaContextInput) map[string]any {
	quota := map[string]any{
		"schema_id":             "runecode.protocol.v0.GatewayQuotaContext",
		"schema_version":        "0.1.0",
		"quota_profile_kind":    input.QuotaProfileKind,
		"phase":                 input.Phase,
		"enforce_during_stream": input.EnforceDuringStream,
		"meters":                map[string]any{},
	}
	if input.StreamLimitBytes != nil {
		quota["stream_limit_bytes"] = *input.StreamLimitBytes
	}
	meters := quota["meters"].(map[string]any)
	setOptionalInt64Field(meters, "request_units", input.Meters.RequestUnits)
	setOptionalInt64Field(meters, "input_tokens", input.Meters.InputTokens)
	setOptionalInt64Field(meters, "output_tokens", input.Meters.OutputTokens)
	setOptionalInt64Field(meters, "streamed_bytes", input.Meters.StreamedBytes)
	setOptionalInt64Field(meters, "concurrency_units", input.Meters.ConcurrencyUnits)
	setOptionalInt64Field(meters, "spend_micros", input.Meters.SpendMicros)
	setOptionalInt64Field(meters, "entitlement_units", input.Meters.EntitlementUnits)
	return quota
}

func setOptionalInt64Field(payload map[string]any, key string, value *int64) {
	if value != nil {
		payload[key] = *value
	}
}
