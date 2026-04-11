package protocolschema

import (
	"sort"
	"testing"
)

func assertRegistryCodeNamespacesSeparate(t *testing.T, registryNames []string, codesByRegistry map[string]map[string]struct{}) {
	t.Helper()

	sort.Strings(registryNames)
	for i := 0; i < len(registryNames); i++ {
		for j := i + 1; j < len(registryNames); j++ {
			assertNoCodeOverlap(t, codesByRegistry, registryNames[i], registryNames[j])
		}
	}
}

func assertErrorRegistryCodes(t *testing.T) {
	t.Helper()
	errorRegistry := loadRegistry(t, schemaPath(t, "registries/error.code.registry.json"))
	assertRegistryContainsCodes(t, errorRegistry, baseErrorRegistryCodes()...)
	assertRegistryContainsCodes(t, errorRegistry, brokerErrorRegistryCodes()...)
	assertRegistryContainsCodes(t, errorRegistry, backendErrorRegistryCodes()...)
}

func baseErrorRegistryCodes() []string {
	return []string{
		"unknown_schema_id",
		"unsupported_schema_version",
		"unsupported_hash_algorithm",
		"schema_bundle_version_mismatch",
		"stream_timeout",
		"gateway_failure",
		"request_cancelled",
	}
}

func brokerErrorRegistryCodes() []string {
	return []string{
		"broker_auth_peer_credentials_required",
		"broker_api_auth_admission_denied",
		"broker_validation_request_id_missing",
		"broker_validation_schema_invalid",
		"broker_validation_payload_base64_invalid",
		"broker_validation_data_class_invalid",
		"broker_validation_operation_invalid",
		"broker_validation_range_not_supported",
		"broker_not_found_artifact",
		"broker_not_found_run",
		"broker_not_found_approval",
		"broker_not_found_session",
		"broker_limit_message_size_exceeded",
		"broker_limit_structural_complexity_exceeded",
		"broker_limit_in_flight_exceeded",
		"broker_limit_rate_exceeded",
		"broker_limit_policy_rejected",
		"broker_limit_response_stream_size_exceeded",
		"broker_timeout_request_deadline_exceeded",
		"broker_approval_state_invalid",
		"broker_validation_runner_transition_invalid",
		"policy_input_hash_mismatch",
	}
}

func backendErrorRegistryCodes() []string {
	return []string{
		"backend_acceleration_unavailable",
		"backend_hypervisor_launch_failed",
		"backend_image_descriptor_signature_mismatch",
		"backend_attachment_plan_invalid",
		"backend_handshake_failed",
		"backend_replay_detected",
		"backend_session_binding_mismatch",
		"backend_guest_unresponsive",
		"backend_watchdog_timeout",
		"backend_required_hardening_unavailable",
		"backend_required_disk_encryption_unavailable",
		"backend_container_automatic_fallback_disallowed",
		"backend_container_opt_in_required",
	}
}

func assertPolicyRegistryCodes(t *testing.T) {
	t.Helper()
	policyRegistry := loadRegistry(t, schemaPath(t, "registries/policy_reason_code.registry.json"))
	assertRegistryContainsCodes(t, policyRegistry,
		"deny_by_default",
		"allow_manifest_opt_in",
		"approval_required",
		"allow_microvm_default",
		"deny_container_opt_in_required",
		"deny_container_automatic_fallback",
		"artifact_flow_denied",
		"unapproved_excerpt_egress_denied",
		"approved_excerpt_revoked",
		"artifact_quota_exceeded",
	)
}

func assertAuditRegistryCodes(t *testing.T) {
	t.Helper()
	auditRegistry := loadRegistry(t, schemaPath(t, "registries/audit_event_type.registry.json"))
	assertRegistryContainsCodes(t, auditRegistry,
		"session_open",
		"model_egress",
		"auth_egress",
		"artifact_flow_blocked",
		"artifact_promotion_action",
		"artifact_quota_violation",
		"artifact_retention_action",
		"policy_decision_recorded",
		"audit_segment_imported",
		"audit_segment_restored",
		"secrets_lease_acquired",
		"secrets_lease_released",
		"isolate_session_started",
		"isolate_session_bound",
	)
	assertAuditEventContractCatalogCoverage(t, auditRegistry)
}

func assertAuditEventContractCatalogCoverage(t *testing.T, auditRegistry registryFile) {
	t.Helper()

	type auditEventContractCatalogFixture struct {
		Entries []struct {
			AuditEventType string `json:"audit_event_type"`
		} `json:"entries"`
	}

	var catalog auditEventContractCatalogFixture
	loadJSON(t, fixturePath(t, "schema/audit-event-contract-catalog.valid.json"), &catalog)

	if len(catalog.Entries) == 0 {
		t.Fatal("audit event contract catalog fixture must include at least one entry")
	}

	seenCatalogTypes := map[string]struct{}{}
	for _, entry := range catalog.Entries {
		if entry.AuditEventType == "" {
			t.Fatal("audit event contract catalog entry must include audit_event_type")
		}
		if _, exists := seenCatalogTypes[entry.AuditEventType]; exists {
			t.Fatalf("audit event contract catalog reuses audit_event_type %q", entry.AuditEventType)
		}
		seenCatalogTypes[entry.AuditEventType] = struct{}{}
		assertRegistryCode(t, auditRegistry, entry.AuditEventType)
	}

	for _, code := range auditRegistry.Codes {
		if _, ok := seenCatalogTypes[code.Code]; !ok {
			t.Fatalf("audit event contract catalog missing registry code %q", code.Code)
		}
	}
}

func assertApprovalRegistryCodes(t *testing.T) {
	t.Helper()
	approvalRegistry := loadRegistry(t, schemaPath(t, "registries/approval_trigger_code.registry.json"))
	assertRegistryContainsCodes(t, approvalRegistry,
		"stage_sign_off",
		"reduced_assurance_backend",
		"gate_override",
		"gateway_egress_scope_change",
		"out_of_workspace_write",
		"secret_access_lease",
		"dependency_network_fetch",
		"system_command_execution",
		"excerpt_promotion",
	)
	hardFloorRegistry := loadRegistry(t, schemaPath(t, "registries/hard_floor_operation_class.registry.json"))
	assertRegistryContainsCodes(t, hardFloorRegistry,
		"trust_root_change",
		"security_posture_weakening",
		"authoritative_state_reconciliation",
		"deployment_bootstrap_authority_change",
	)
	actionRegistry := loadRegistry(t, schemaPath(t, "registries/action_kind.registry.json"))
	assertRegistryContainsCodes(t, actionRegistry,
		"workspace_write",
		"executor_run",
		"artifact_read",
		"promotion",
		"gateway_egress",
		"dependency_fetch",
		"backend_posture_change",
		"action_gate_override",
		"stage_summary_sign_off",
		"secret_access",
	)
}

func assertAuditReceiptRegistryCodes(t *testing.T) {
	t.Helper()
	auditReceiptRegistry := loadRegistry(t, schemaPath(t, "registries/audit_receipt_kind.registry.json"))
	assertRegistryContainsCodes(t, auditReceiptRegistry, "anchor", "import", "restore", "reconciliation")
}

func assertAuditVerificationReasonRegistryCodes(t *testing.T) {
	t.Helper()
	auditVerificationRegistry := loadRegistry(t, schemaPath(t, "registries/audit_verification_reason_code.registry.json"))
	assertRegistryContainsCodes(t, auditVerificationRegistry,
		"segment_frame_digest_mismatch",
		"segment_frame_byte_length_mismatch",
		"segment_file_hash_mismatch",
		"segment_merkle_root_mismatch",
		"segment_seal_invalid",
		"segment_seal_chain_mismatch",
		"stream_sequence_gap",
		"stream_sequence_rollback_or_duplicate",
		"stream_previous_hash_mismatch",
		"detached_signature_invalid",
		"signer_evidence_missing",
		"signer_evidence_invalid",
		"signer_historically_inadmissible",
		"signer_currently_revoked_or_compromised",
		"event_contract_mismatch",
		"event_contract_missing",
		"import_restore_provenance_inconsistent",
		"receipt_invalid",
		"anchor_receipt_missing",
		"anchor_receipt_invalid",
		"segment_lifecycle_inconsistent",
		"storage_posture_degraded",
		"storage_posture_invalid",
	)
}

func assertRegistryContainsCodes(t *testing.T, registry registryFile, codes ...string) {
	t.Helper()
	for _, code := range codes {
		assertRegistryCode(t, registry, code)
	}
}
