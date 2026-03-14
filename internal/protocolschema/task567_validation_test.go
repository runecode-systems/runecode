package protocolschema

import "testing"

func TestErrorSchemaRequiresCategoryRetryabilityAndTypedDetails(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/Error.schema.json")

	for _, testCase := range errorCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestPolicyDecisionRequiresHashBindingsAndApprovalPayloads(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/PolicyDecision.schema.json")

	for _, testCase := range policyDecisionCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestArtifactAndProvenanceSchemasRequireAuditLinkage(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	artifactSchema := mustCompileObjectSchema(t, bundle, "objects/ArtifactReference.schema.json")
	provenanceSchema := mustCompileObjectSchema(t, bundle, "objects/ProvenanceReceipt.schema.json")

	for _, testCase := range artifactReferenceCases() {
		testCase := testCase
		t.Run("artifact/"+testCase.name, func(t *testing.T) {
			err := artifactSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range provenanceReceiptCases() {
		testCase := testCase
		t.Run("provenance/"+testCase.name, func(t *testing.T) {
			err := provenanceSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestAuditSchemasRequireTypedPayloadsAndSignatures(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	auditEventSchema := mustCompileObjectSchema(t, bundle, "objects/AuditEvent.schema.json")
	auditReceiptSchema := mustCompileObjectSchema(t, bundle, "objects/AuditReceipt.schema.json")

	for _, testCase := range auditEventCases() {
		testCase := testCase
		t.Run("event/"+testCase.name, func(t *testing.T) {
			err := auditEventSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range auditReceiptCases() {
		testCase := testCase
		t.Run("receipt/"+testCase.name, func(t *testing.T) {
			err := auditReceiptSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func errorCases() []validationCase {
	return []validationCase{
		{name: "minimal error", value: validErrorEnvelope()},
		{name: "typed details pair stays valid", value: validErrorEnvelopeWithDetails()},
		{name: "details require schema id", value: invalidErrorEnvelopeWithoutDetailsSchema(), wantErr: true},
		{name: "error code enforces identifier format", value: invalidErrorEnvelopeCode(), wantErr: true},
		{name: "category enum fails closed", value: invalidErrorEnvelopeCategory(), wantErr: true},
	}
}

func policyDecisionCases() []validationCase {
	return []validationCase{
		{name: "allow decision", value: validAllowPolicyDecision()},
		{name: "deny decision", value: validDenyPolicyDecision()},
		{name: "approval decision", value: validApprovalPolicyDecision()},
		{name: "policy reason code enforces identifier format", value: invalidPolicyDecisionWithBadReasonCode(), wantErr: true},
		{name: "approval outcome requires payload", value: invalidApprovalPolicyDecisionWithoutPayload(), wantErr: true},
		{name: "deny decision rejects approval payload", value: invalidDenyPolicyDecisionWithApprovalPayload(), wantErr: true},
	}
}

func artifactReferenceCases() []validationCase {
	return []validationCase{
		{name: "artifact reference", value: validArtifactReference()},
		{name: "artifact enforces content type format", value: invalidArtifactReferenceWithBadContentType(), wantErr: true},
		{name: "artifact enforces data class format", value: invalidArtifactReferenceWithBadDataClass(), wantErr: true},
		{name: "artifact requires provenance", value: invalidArtifactReferenceWithoutProvenance(), wantErr: true},
	}
}

func provenanceReceiptCases() []validationCase {
	return []validationCase{
		{name: "audit event linkage", value: validProvenanceReceipt()},
		{name: "audit receipt linkage", value: validProvenanceReceiptWithReceiptHash()},
		{name: "audit linkage is mutually exclusive", value: invalidProvenanceReceiptWithBothAuditLinks(), wantErr: true},
		{name: "must link to audit evidence", value: invalidProvenanceReceiptWithoutAuditLinkage(), wantErr: true},
	}
}

func auditEventCases() []validationCase {
	return []validationCase{
		{name: "typed audit event", value: validAuditEvent()},
		{name: "gateway audit event", value: validGatewayAuditEvent()},
		{name: "audit event type enforces identifier format", value: invalidAuditEventWithBadType(), wantErr: true},
		{name: "audit event requires payload hash", value: invalidAuditEventWithoutPayloadHash(), wantErr: true},
	}
}

func auditReceiptCases() []validationCase {
	return []validationCase{
		{name: "minimal receipt", value: validAuditReceipt()},
		{name: "typed payload receipt", value: validAuditReceiptWithPayload()},
		{name: "receipt kind enforces identifier format", value: invalidAuditReceiptWithBadKind(), wantErr: true},
		{name: "payload requires schema id", value: invalidAuditReceiptWithoutPayloadSchema(), wantErr: true},
	}
}

func validErrorEnvelope() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.Error",
		"schema_version": "0.3.0",
		"code":           "unsupported_schema_version",
		"category":       "validation",
		"retryable":      false,
		"message":        "Schema version 0.9.0 is not supported by this verifier.",
	}
}

func validErrorEnvelopeWithDetails() map[string]any {
	err := validErrorEnvelope()
	err["details_schema_id"] = "runecode.protocol.details.error.unsupported-schema.v0"
	err["details"] = map[string]any{"supported_versions": []any{"0.2.0"}}
	return err
}

func invalidErrorEnvelopeWithoutDetailsSchema() map[string]any {
	err := validErrorEnvelope()
	err["details"] = map[string]any{"field": "schema_version"}
	return err
}

func invalidErrorEnvelopeCode() map[string]any {
	err := validErrorEnvelope()
	err["code"] = "unsupported-schema-version"
	return err
}

func invalidErrorEnvelopeCategory() map[string]any {
	err := validErrorEnvelope()
	err["category"] = "network"
	return err
}

func validDenyPolicyDecision() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.PolicyDecision",
		"schema_version":           "0.3.0",
		"decision_outcome":         "deny",
		"policy_reason_code":       "deny_by_default",
		"manifest_hash":            testDigestValue("1"),
		"action_request_hash":      testDigestValue("2"),
		"relevant_artifact_hashes": []any{testDigestValue("3")},
		"policy_input_hashes":      []any{testDigestValue("4")},
		"details_schema_id":        "runecode.protocol.details.policy.decision.v0",
		"details":                  map[string]any{"rule": "deny_by_default"},
	}
}

func validAllowPolicyDecision() map[string]any {
	decision := validDenyPolicyDecision()
	decision["decision_outcome"] = "allow"
	decision["policy_reason_code"] = "allow_manifest_opt_in"
	decision["details"] = map[string]any{"rule": "manifest_opt_in"}
	return decision
}

func validApprovalPolicyDecision() map[string]any {
	decision := validDenyPolicyDecision()
	decision["decision_outcome"] = "require_human_approval"
	decision["policy_reason_code"] = "approval_required"
	decision["required_approval_schema_id"] = "runecode.protocol.details.policy.required-approval.v0"
	decision["required_approval"] = map[string]any{"approval_trigger_code": "gateway_egress_scope_change"}
	return decision
}

func invalidApprovalPolicyDecisionWithoutPayload() map[string]any {
	decision := validApprovalPolicyDecision()
	delete(decision, "required_approval")
	return decision
}

func invalidDenyPolicyDecisionWithApprovalPayload() map[string]any {
	decision := validDenyPolicyDecision()
	decision["required_approval_schema_id"] = "runecode.protocol.details.policy.required-approval.v0"
	decision["required_approval"] = map[string]any{"approval_trigger_code": "gateway_egress_scope_change"}
	return decision
}

func invalidPolicyDecisionWithBadReasonCode() map[string]any {
	decision := validDenyPolicyDecision()
	decision["policy_reason_code"] = "deny-by-default"
	return decision
}

func validArtifactReference() map[string]any {
	return map[string]any{
		"schema_id":               "runecode.protocol.v0.ArtifactReference",
		"schema_version":          "0.2.0",
		"digest":                  testDigestValue("7"),
		"size_bytes":              128,
		"content_type":            "application/json",
		"data_class":              "spec_text",
		"provenance_receipt_hash": testDigestValue("8"),
	}
}

func invalidArtifactReferenceWithoutProvenance() map[string]any {
	artifact := validArtifactReference()
	delete(artifact, "provenance_receipt_hash")
	return artifact
}

func invalidArtifactReferenceWithBadContentType() map[string]any {
	artifact := validArtifactReference()
	artifact["content_type"] = "not a mime type"
	return artifact
}

func invalidArtifactReferenceWithBadDataClass() map[string]any {
	artifact := validArtifactReference()
	artifact["data_class"] = "SpecText"
	return artifact
}

func validProvenanceReceipt() map[string]any {
	return map[string]any{
		"schema_id":                  "runecode.protocol.v0.ProvenanceReceipt",
		"schema_version":             "0.2.0",
		"artifact_digest":            testDigestValue("7"),
		"producer":                   manifestPrincipal(),
		"source_artifact_hashes":     []any{testDigestValue("9")},
		"produced_at":                "2026-03-13T12:10:00Z",
		"producing_audit_event_hash": testDigestValue("a"),
		"signatures":                 []any{signatureBlock()},
	}
}

func validProvenanceReceiptWithReceiptHash() map[string]any {
	receipt := validProvenanceReceipt()
	delete(receipt, "producing_audit_event_hash")
	receipt["producing_audit_receipt_hash"] = testDigestValue("b")
	return receipt
}

func invalidProvenanceReceiptWithBothAuditLinks() map[string]any {
	receipt := validProvenanceReceipt()
	receipt["producing_audit_receipt_hash"] = testDigestValue("b")
	return receipt
}

func invalidProvenanceReceiptWithoutAuditLinkage() map[string]any {
	receipt := validProvenanceReceipt()
	delete(receipt, "producing_audit_event_hash")
	return receipt
}

func validAuditEvent() map[string]any {
	return map[string]any{
		"schema_id":               "runecode.protocol.v0.AuditEvent",
		"schema_version":          "0.3.0",
		"audit_event_type":        "session_start",
		"seq":                     1,
		"occurred_at":             "2026-03-13T12:15:00Z",
		"principal":               manifestPrincipal(),
		"event_payload_schema_id": "runecode.protocol.audit.payload.session-start.v0",
		"event_payload":           map[string]any{"session_id": "session-1"},
		"event_payload_hash":      testDigestValue("c"),
		"signatures":              []any{signatureBlock()},
	}
}

func validGatewayAuditEvent() map[string]any {
	event := validAuditEvent()
	event["audit_event_type"] = "model_egress"
	event["gateway_context"] = map[string]any{
		"egress_category":        "model",
		"allowlist_ref":          testDigestValue("d"),
		"destination_descriptor": "api.openai.com:443",
	}
	event["schema_bundle_version"] = "0.3.0"
	event["related_artifact_hashes"] = []any{testDigestValue("7")}
	event["related_decision_hashes"] = []any{testDigestValue("e")}
	event["related_receipt_hashes"] = []any{testDigestValue("f")}
	return event
}

func invalidAuditEventWithoutPayloadHash() map[string]any {
	event := validAuditEvent()
	delete(event, "event_payload_hash")
	return event
}

func invalidAuditEventWithBadType() map[string]any {
	event := validAuditEvent()
	event["audit_event_type"] = "model-egress"
	return event
}

func validAuditReceipt() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.AuditReceipt",
		"schema_version": "0.2.0",
		"event_digest":   testDigestValue("c"),
		"receipt_kind":   "write_ack",
		"recorder":       manifestPrincipal(),
		"recorded_at":    "2026-03-13T12:16:00Z",
		"signatures":     []any{signatureBlock()},
	}
}

func validAuditReceiptWithPayload() map[string]any {
	receipt := validAuditReceipt()
	receipt["receipt_payload_schema_id"] = "runecode.protocol.audit.receipt.anchor.v0"
	receipt["receipt_payload"] = map[string]any{"anchor_kind": "local"}
	return receipt
}

func invalidAuditReceiptWithoutPayloadSchema() map[string]any {
	receipt := validAuditReceiptWithPayload()
	delete(receipt, "receipt_payload_schema_id")
	return receipt
}

func invalidAuditReceiptWithBadKind() map[string]any {
	receipt := validAuditReceipt()
	receipt["receipt_kind"] = "Write-Ack"
	return receipt
}
