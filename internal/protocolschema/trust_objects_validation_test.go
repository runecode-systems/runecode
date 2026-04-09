package protocolschema

import "testing"

func TestPrincipalIdentityRequiresRunForStageContext(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/PrincipalIdentity.schema.json")

	for _, testCase := range principalLifecycleCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestCapabilityManifestScopeRequiresLifecycleIDs(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/CapabilityManifest.schema.json")

	for _, testCase := range capabilityManifestCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestRoleManifestArraySemantics(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/RoleManifest.schema.json")

	for _, testCase := range roleManifestCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestApprovalRequestRequiresHashBindingsAndModerateProfile(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/ApprovalRequest.schema.json")

	for _, testCase := range approvalRequestCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestApprovalDecisionRequiresMachineCheckableRestrictionSchema(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/ApprovalDecision.schema.json")

	for _, testCase := range approvalDecisionCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestVerifierRecordRequiresDeterministicKeyIdentityAndClosedPostures(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/VerifierRecord.schema.json")

	for _, testCase := range verifierRecordCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func capabilityManifestCases() []validationCase {
	return []validationCase{
		{name: "run scope omits stage id", value: validRunCapabilityManifest()},
		{name: "empty capabilities stay valid", value: validRunCapabilityManifestWithoutCapabilities()},
		{name: "stage scope requires stage id", value: validStageCapabilityManifest()},
		{name: "run scope rejects stage id", value: invalidRunScopedManifestWithStageID(), wantErr: true},
		{name: "duplicate allowlist refs fail", value: invalidRunCapabilityManifestWithDuplicateAllowlists(), wantErr: true},
		{name: "stage scope requires stage lifecycle id", value: invalidStageScopedManifestWithoutStageID(), wantErr: true},
	}
}

func roleManifestCases() []validationCase {
	return []validationCase{
		{name: "empty capabilities stay valid", value: validRoleManifestWithoutCapabilities()},
		{name: "duplicate allowlist refs fail", value: invalidRoleManifestWithDuplicateAllowlists(), wantErr: true},
	}
}

func approvalRequestCases() []validationCase {
	return []validationCase{
		{name: "valid request", value: validApprovalRequest()},
		{name: "empty artifact hashes stay valid", value: validApprovalRequestWithoutArtifactHashes()},
		{name: "runtime-owned timestamp ordering remains schema-valid", value: validApprovalRequestWithInvertedTimes()},
		{name: "runtime-owned trigger registry lookup remains schema-valid", value: validApprovalRequestWithUnknownTriggerCode()},
		{name: "unknown profile fails closed", value: invalidApprovalRequestProfile(), wantErr: true},
		{name: "unknown assurance level fails closed", value: invalidApprovalRequestAssuranceLevel(), wantErr: true},
		{name: "unknown presence mode fails closed", value: invalidApprovalRequestPresenceMode(), wantErr: true},
		{name: "details schema id is required", value: invalidApprovalRequestWithoutDetailsSchema(), wantErr: true},
	}
}

func approvalDecisionCases() []validationCase {
	return []validationCase{
		{name: "valid decision", value: validApprovalDecision()},
		{name: "minimal decision omits optional expiry", value: validMinimalApprovalDecision()},
		{name: "unknown assurance level fails closed", value: invalidApprovalDecisionAssuranceLevel(), wantErr: true},
		{name: "unknown presence mode fails closed", value: invalidApprovalDecisionPresenceMode(), wantErr: true},
		{name: "restrictions require schema id", value: invalidApprovalDecisionWithoutRestrictionSchema(), wantErr: true},
		{name: "restrictions schema id requires restrictions", value: invalidApprovalDecisionWithoutRestrictions(), wantErr: true},
	}
}

func verifierRecordCases() []validationCase {
	return []validationCase{
		{name: "valid verifier record", value: validVerifierRecord()},
		{name: "unknown key posture fails closed", value: invalidVerifierRecordUnknownKeyPosture(), wantErr: true},
		{name: "unknown presence mode fails closed", value: invalidVerifierRecordUnknownPresenceMode(), wantErr: true},
		{name: "path-like key id value fails", value: invalidVerifierRecordPathLikeKeyID(), wantErr: true},
	}
}

func principalLifecycleCases() []validationCase {
	return []validationCase{
		{name: "stage context requires run id", value: stageScopedPrincipalWithoutRun(), wantErr: true},
		{name: "stage context with run id stays valid", value: validStageScopedPrincipal()},
	}
}

func stageScopedPrincipalWithoutRun() map[string]any {
	principal := manifestPrincipal()
	principal["stage_id"] = "stage-1"
	return principal
}

func manifestPrincipal() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
		"schema_version": "0.2.0",
		"actor_kind":     "daemon",
		"principal_id":   "broker",
		"instance_id":    "broker-1",
		"role_family":    "workspace",
		"role_kind":      "workspace-edit",
	}
}

func approverPrincipal() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
		"schema_version": "0.2.0",
		"actor_kind":     "user",
		"principal_id":   "alice",
		"instance_id":    "approval-session-1",
	}
}

func validStageScopedPrincipal() map[string]any {
	principal := manifestPrincipal()
	principal["run_id"] = "run-1"
	principal["stage_id"] = "stage-1"
	return principal
}

func validRoleManifest() map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          manifestPrincipal(),
		"role_family":        "workspace",
		"role_kind":          "workspace-edit",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"workspace_write"},
		"allowlist_refs":     []any{testDigestValue("a")},
		"signatures":         []any{signatureBlock()},
	}
}

func validRoleManifestWithoutCapabilities() map[string]any {
	manifest := validRoleManifest()
	manifest["capability_opt_ins"] = []any{}
	return manifest
}

func invalidRoleManifestWithDuplicateAllowlists() map[string]any {
	manifest := validRoleManifest()
	manifest["allowlist_refs"] = []any{testDigestValue("a"), testDigestValue("a")}
	return manifest
}

func validRunCapabilityManifest() map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          manifestPrincipal(),
		"manifest_scope":     "run",
		"run_id":             "run-1",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"model_egress"},
		"allowlist_refs":     []any{testDigestValue("a")},
		"signatures":         []any{signatureBlock()},
	}
}

func validRunCapabilityManifestWithoutCapabilities() map[string]any {
	manifest := validRunCapabilityManifest()
	manifest["capability_opt_ins"] = []any{}
	return manifest
}

func validStageCapabilityManifest() map[string]any {
	manifest := validRunCapabilityManifest()
	manifest["manifest_scope"] = "stage"
	manifest["stage_id"] = "stage-1"
	return manifest
}

func invalidRunCapabilityManifestWithDuplicateAllowlists() map[string]any {
	manifest := validRunCapabilityManifest()
	manifest["allowlist_refs"] = []any{testDigestValue("a"), testDigestValue("a")}
	return manifest
}

func invalidRunScopedManifestWithStageID() map[string]any {
	manifest := validRunCapabilityManifest()
	manifest["stage_id"] = "stage-1"
	return manifest
}

func invalidStageScopedManifestWithoutStageID() map[string]any {
	manifest := validRunCapabilityManifest()
	manifest["manifest_scope"] = "stage"
	return manifest
}

func validApprovalRequest() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.ApprovalRequest",
		"schema_version":           "0.3.0",
		"approval_profile":         "moderate",
		"requester":                manifestPrincipal(),
		"approval_trigger_code":    "gateway_egress_scope_change",
		"manifest_hash":            testDigestValue("a"),
		"action_request_hash":      testDigestValue("b"),
		"relevant_artifact_hashes": []any{testDigestValue("c")},
		"details_schema_id":        "runecode.protocol.details.approval.gateway-egress.v0",
		"details":                  map[string]any{"requested_category": "model"},
		"approval_assurance_level": "session_authenticated",
		"presence_mode":            "os_confirmation",
		"requested_at":             "2026-03-13T12:00:00Z",
		"expires_at":               "2026-03-13T12:30:00Z",
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Enable model-gateway egress for spec_text artifacts to the signed allowlist.",
		"signatures":               []any{signatureBlock()},
	}
}

func validApprovalRequestWithoutArtifactHashes() map[string]any {
	request := validApprovalRequest()
	request["relevant_artifact_hashes"] = []any{}
	return request
}

func validApprovalRequestWithInvertedTimes() map[string]any {
	request := validApprovalRequest()
	request["expires_at"] = "2026-03-13T11:59:00Z"
	return request
}

func validApprovalRequestWithUnknownTriggerCode() map[string]any {
	request := validApprovalRequest()
	request["approval_trigger_code"] = "future_runtime_defined_code"
	return request
}

func invalidApprovalRequestProfile() map[string]any {
	request := validApprovalRequest()
	request["approval_profile"] = "strict"
	return request
}

func invalidApprovalRequestAssuranceLevel() map[string]any {
	request := validApprovalRequest()
	request["approval_assurance_level"] = "channel_click"
	return request
}

func invalidApprovalRequestPresenceMode() map[string]any {
	request := validApprovalRequest()
	request["presence_mode"] = "biometric"
	return request
}

func invalidApprovalRequestWithoutDetailsSchema() map[string]any {
	request := validApprovalRequest()
	delete(request, "details_schema_id")
	return request
}

func validApprovalDecision() map[string]any {
	return map[string]any{
		"schema_id":                   "runecode.protocol.v0.ApprovalDecision",
		"schema_version":              "0.3.0",
		"approval_request_hash":       testDigestValue("d"),
		"approver":                    approverPrincipal(),
		"decision_outcome":            "approve",
		"approval_assurance_level":    "reauthenticated",
		"presence_mode":               "hardware_touch",
		"key_protection_posture":      "hardware_backed",
		"identity_binding_posture":    "attested",
		"approval_assertion_hash":     testDigestValue("1"),
		"decided_at":                  "2026-03-13T12:05:00Z",
		"consumption_posture":         "single_use",
		"decision_expires_at":         "2026-03-13T12:30:00Z",
		"restrictions_schema_id":      "runecode.protocol.details.approval.restrictions.v0",
		"restrictions":                map[string]any{"max_uses": 1},
		"policy_decision_hash":        testDigestValue("e"),
		"stage_manifest_summary_hash": testDigestValue("f"),
		"signatures":                  []any{signatureBlock()},
	}
}

func validMinimalApprovalDecision() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.ApprovalDecision",
		"schema_version":           "0.3.0",
		"approval_request_hash":    testDigestValue("d"),
		"approver":                 approverPrincipal(),
		"decision_outcome":         "approve",
		"approval_assurance_level": "none",
		"presence_mode":            "none",
		"key_protection_posture":   "os_keystore",
		"identity_binding_posture": "tofu",
		"decided_at":               "2026-03-13T12:05:00Z",
		"consumption_posture":      "single_use",
		"signatures":               []any{signatureBlock()},
	}
}

func invalidApprovalDecisionWithoutRestrictionSchema() map[string]any {
	decision := validApprovalDecision()
	delete(decision, "restrictions_schema_id")
	return decision
}

func invalidApprovalDecisionAssuranceLevel() map[string]any {
	decision := validApprovalDecision()
	decision["approval_assurance_level"] = "delivery_channel_only"
	return decision
}

func invalidApprovalDecisionPresenceMode() map[string]any {
	decision := validApprovalDecision()
	decision["presence_mode"] = "push"
	return decision
}

func invalidApprovalDecisionWithoutRestrictions() map[string]any {
	decision := validApprovalDecision()
	delete(decision, "restrictions")
	return decision
}

func validVerifierRecord() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.VerifierRecord",
		"schema_version":           "0.1.0",
		"key_id":                   "key_sha256",
		"key_id_value":             "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"alg":                      "ed25519",
		"public_key":               map[string]any{"encoding": "base64", "value": "AQID"},
		"logical_purpose":          "approval_authority",
		"logical_scope":            "user",
		"owner_principal":          approverPrincipal(),
		"key_protection_posture":   "hardware_backed",
		"identity_binding_posture": "attested",
		"presence_mode":            "hardware_touch",
		"created_at":               "2026-03-13T12:00:00Z",
		"status":                   "active",
	}
}

func invalidVerifierRecordUnknownKeyPosture() map[string]any {
	record := validVerifierRecord()
	record["key_protection_posture"] = "plaintext_disk"
	return record
}

func invalidVerifierRecordUnknownPresenceMode() map[string]any {
	record := validVerifierRecord()
	record["presence_mode"] = "biometric"
	return record
}

func invalidVerifierRecordPathLikeKeyID() map[string]any {
	record := validVerifierRecord()
	record["key_id_value"] = "../../keys/local-user"
	return record
}
