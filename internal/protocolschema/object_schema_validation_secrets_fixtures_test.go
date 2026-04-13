package protocolschema

func validActionPayloadSecretAccessLeaseIssue() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.ActionPayloadSecretAccess",
		"schema_version": "0.1.0",
		"secret_ref":     "secrets/prod/db-password",
		"access_mode":    "lease_issue",
	}
}

func validActionPayloadSecretAccessLeaseRenew() map[string]any {
	payload := validActionPayloadSecretAccessLeaseIssue()
	delete(payload, "secret_ref")
	payload["access_mode"] = "lease_renew"
	payload["lease_id"] = "lease-prod-db-001"
	payload["lease_ttl_seconds"] = 600
	payload["renewal_context"] = map[string]any{
		"consumer_principal_ref": "principal:run-123:workspace-editor",
		"target_ref":             "db.prod.internal:5432",
		"policy_context_hash":    testDigestValue("5"),
	}
	return payload
}

func validActionPayloadSecretAccessLeaseRevoke() map[string]any {
	payload := validActionPayloadSecretAccessLeaseIssue()
	delete(payload, "secret_ref")
	payload["access_mode"] = "lease_revoke"
	payload["lease_id"] = "lease-prod-db-001"
	return payload
}

func invalidActionPayloadSecretAccessLeaseRenewWithoutLeaseID() map[string]any {
	payload := validActionPayloadSecretAccessLeaseRenew()
	delete(payload, "lease_id")
	return payload
}

func invalidActionPayloadSecretAccessLeaseRenewWithoutRenewalContext() map[string]any {
	payload := validActionPayloadSecretAccessLeaseRenew()
	delete(payload, "renewal_context")
	return payload
}

func invalidActionPayloadSecretAccessLeaseRevokeWithTTL() map[string]any {
	payload := validActionPayloadSecretAccessLeaseRevoke()
	payload["lease_ttl_seconds"] = 120
	return payload
}

func validSecretLeasePersistedSecretActive() map[string]any {
	return map[string]any{
		"schema_id":             "runecode.protocol.v0.SecretLease",
		"schema_version":        "0.1.0",
		"lease_id":              "lease-prod-db-001",
		"lease_kind":            "persisted_secret",
		"lease_subject":         secretLeasePersistedSubject(),
		"consumer_binding":      secretLeaseWorkspaceConsumerBinding(),
		"target_binding":        secretLeasePersistedTargetBinding(),
		"delivery_binding":      secretLeaseDeliveryBinding(),
		"action_policy_binding": secretLeaseActionPolicyBinding(),
		"lifecycle":             secretLeaseActiveLifecycle(),
		"revocation":            map[string]any{"status": "active"},
		"durable_state":         secretLeaseDurableState(),
	}
}

func secretLeasePersistedSubject() map[string]any {
	return map[string]any{"subject_ref": "secrets/prod/db-password", "issuer_ref": "secretsd"}
}

func secretLeaseWorkspaceConsumerBinding() map[string]any {
	return map[string]any{
		"consumer_principal_ref": "principal:run-123:workspace-editor",
		"role_family":            "workspace",
		"role_kind":              "workspace-edit",
		"session_id":             "session-123",
	}
}

func secretLeasePersistedTargetBinding() map[string]any {
	return map[string]any{"target_kind": "destination_ref", "target_ref": "db.prod.internal:5432"}
}

func secretLeaseDeliveryBinding() map[string]any {
	return map[string]any{
		"retrieval_path_kind":       "secretsd_trusted_local",
		"lease_identity_field":      "lease_id",
		"consumer_binding_enforced": true,
		"bridge_handoff_posture":    "lease_boundary_only",
	}
}

func secretLeaseActionPolicyBinding() map[string]any {
	return map[string]any{
		"action_kind":         "secret_access",
		"policy_context_hash": testDigestValue("1"),
		"policy_binding_hash": testDigestValue("2"),
		"action_request_hash": testDigestValue("3"),
		"audit_binding_hash":  testDigestValue("4"),
	}
}

func secretLeaseActiveLifecycle() map[string]any {
	return map[string]any{
		"issued_at":             "2026-04-12T11:00:00Z",
		"expires_at":            "2026-04-12T11:15:00Z",
		"effective_ttl_seconds": 900,
		"lease_mode":            "use",
		"renewal_allowed":       true,
		"last_renewed_at":       "2026-04-12T11:05:00Z",
		"renewal_count":         1,
	}
}

func secretLeaseDurableState() map[string]any {
	return map[string]any{"state_persisted": true, "deny_outcomes_preserved": true, "expiry_indexed": true}
}

func validSecretLeaseDerivedTokenRevoked() map[string]any {
	lease := validSecretLeasePersistedSecretActive()
	lease["lease_id"] = "lease-auth-token-009"
	lease["lease_kind"] = "derived_token"
	lease["lease_subject"] = map[string]any{
		"subject_ref":     "tokens/auth/provider-access",
		"source_lease_id": "lease-prod-db-001",
		"issuer_ref":      "auth-gateway",
	}
	lease["consumer_binding"] = map[string]any{
		"consumer_principal_ref": "principal:run-123:auth-gateway",
		"role_family":            "gateway",
		"role_kind":              "auth-gateway",
	}
	lease["target_binding"] = map[string]any{
		"target_kind": "destination_ref",
		"target_ref":  "auth.provider.example.com/oauth/token",
	}
	lease["revocation"] = map[string]any{
		"status":               "revoked",
		"revoked_at":           "2026-04-12T10:40:00Z",
		"revoked_by":           "principal:operator:security",
		"revocation_reason":    "Revoked after incident response containment.",
		"revocation_persisted": true,
	}
	return lease
}

func invalidSecretLeaseDerivedTokenMissingSourceLeaseID() map[string]any {
	lease := validSecretLeaseDerivedTokenRevoked()
	lease["revocation"] = map[string]any{"status": "active"}
	delete(lease["lease_subject"].(map[string]any), "source_lease_id")
	return lease
}

func invalidSecretLeaseRevokedWithoutRevocationMetadata() map[string]any {
	lease := validSecretLeasePersistedSecretActive()
	lease["revocation"] = map[string]any{"status": "revoked"}
	return lease
}

func invalidSecretLeaseRevokedWithUndurableState() map[string]any {
	lease := validSecretLeaseDerivedTokenRevoked()
	lease["durable_state"] = map[string]any{
		"state_persisted":         false,
		"deny_outcomes_preserved": true,
		"expiry_indexed":          true,
	}
	return lease
}

func invalidSecretLeaseWithoutTrustedLocalDeliveryBinding() map[string]any {
	lease := validSecretLeasePersistedSecretActive()
	delete(lease, "delivery_binding")
	return lease
}

func validSecretStoragePostureSecureDefault() map[string]any {
	return map[string]any{
		"schema_id":                                   "runecode.protocol.v0.SecretStoragePosture",
		"schema_version":                              "0.1.0",
		"long_lived_secret_store_authority":           "secretsd_only",
		"secure_storage_preference":                   "os_or_hardware_backed",
		"secure_storage_available":                    true,
		"fail_closed_when_secure_storage_unavailable": true,
		"effective_custody_posture":                   "secure_default",
		"portable_passphrase_opt_in":                  map[string]any{"enabled": false},
		"onboarding_contract":                         secretStorageOnboardingContract([]any{"stdin"}),
		"durable_state":                               secretStorageDurableState(),
	}
}

func secretStorageOnboardingContract(sources []any) map[string]any {
	return map[string]any{
		"canonical_portable_source":    "stdin",
		"supported_sources":            sources,
		"file_descriptor_support":      "platform_appropriate",
		"audit_metadata_only":          true,
		"audit_includes_secret_values": false,
	}
}

func secretStorageDurableState() map[string]any {
	return map[string]any{
		"secret_metadata": map[string]any{"record_count": 3, "encrypted_material_present": true, "last_updated_at": "2026-04-12T10:00:00Z"},
		"lease_state":     map[string]any{"active_lease_count": 1, "expired_lease_count": 2, "last_recovered_at": "2026-04-12T10:05:00Z"},
		"revocation_state": map[string]any{
			"revoked_lease_count":        1,
			"revocation_index_persisted": true,
		},
		"linkage_metadata": map[string]any{"policy_binding_hash_count": 4, "audit_link_hash_count": 6},
	}
}

func validSecretStoragePosturePortableDegraded() map[string]any {
	posture := validSecretStoragePostureSecureDefault()
	posture["secure_storage_available"] = false
	posture["effective_custody_posture"] = "portable_passphrase_derived_degraded"
	posture["portable_passphrase_opt_in"] = map[string]any{
		"enabled":                 true,
		"kdf_profile":             "argon2id_v1",
		"passphrase_source":       "stdin",
		"opt_in_audit_event_hash": testDigestValue("1"),
		"opted_in_at":             "2026-04-12T10:10:00Z",
		"justification":           "Portable setup approved for local-only degraded operation.",
	}
	posture["onboarding_contract"] = secretStorageOnboardingContract([]any{"stdin", "fd"})
	return posture
}

func invalidSecretStoragePostureUnavailableWithoutOptIn() map[string]any {
	posture := validSecretStoragePostureSecureDefault()
	posture["secure_storage_available"] = false
	posture["onboarding_contract"] = secretStorageOnboardingContract([]any{"fd"})
	return posture
}

func validBrokerReadinessWithSecretsHealthAndMetrics() map[string]any {
	return map[string]any{
		"schema_id":                   "runecode.protocol.v0.BrokerReadiness",
		"schema_version":              "0.1.0",
		"ready":                       true,
		"local_only":                  true,
		"consumption_channel":         "broker_local_api",
		"recovery_complete":           true,
		"append_position_stable":      true,
		"current_segment_writable":    true,
		"verifier_material_available": true,
		"derived_index_caught_up":     true,
		"secrets_ready":               true,
		"secrets_health_state":        "ok",
		"secrets_operational_metrics": map[string]any{"lease_issue_count": 12, "lease_renew_count": 7, "lease_revoke_count": 3, "lease_denied_count": 1, "active_lease_count": 4},
		"secrets_storage_posture":     validSecretStoragePostureSecureDefault(),
	}
}

func invalidBrokerReadinessSecretsHealthWithoutReadyFlag() map[string]any {
	readiness := validBrokerReadinessWithSecretsHealthAndMetrics()
	delete(readiness, "secrets_ready")
	return readiness
}

func invalidBrokerReadinessSecretsMetricsWithoutReadyFlag() map[string]any {
	readiness := validBrokerReadinessWithSecretsHealthAndMetrics()
	delete(readiness, "secrets_ready")
	delete(readiness, "secrets_health_state")
	return readiness
}
