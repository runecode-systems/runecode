package protocolschema

import "strings"

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

func invalidPolicyDecisionWithUnknownReasonCode() map[string]any {
	decision := validDenyPolicyDecision()
	decision["policy_reason_code"] = "runtime_defined_future_code"
	return decision
}

func invalidApprovalPolicyDecisionWithUnknownTriggerCode() map[string]any {
	decision := validApprovalPolicyDecision()
	decision["required_approval"] = map[string]any{"approval_trigger_code": "future_runtime_defined_trigger"}
	return decision
}

func validProjectLifecycleApplyApprovalPolicyDecision() map[string]any {
	decision := validApprovalPolicyDecision()
	decision["required_approval"] = map[string]any{"approval_trigger_code": "project_lifecycle_apply"}
	return decision
}

func validArtifactReference() map[string]any {
	return map[string]any{
		"schema_id":               "runecode.protocol.v0.ArtifactReference",
		"schema_version":          "0.4.0",
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
	artifact["data_class"] = "unknown_class"
	return artifact
}

func artifactReferenceWithDataClass(dataClass string) map[string]any {
	artifact := validArtifactReference()
	artifact["data_class"] = dataClass
	return artifact
}

func validArtifactPolicy() map[string]any {
	return map[string]any{
		"schema_id":                       "runecode.protocol.v0.ArtifactPolicy",
		"schema_version":                  "0.1.0",
		"handoff_reference_mode":          "hash_only",
		"cas":                             validArtifactPolicyCAS(),
		"storage_protection":              validArtifactPolicyStorageProtection(),
		"approval_promotion":              validArtifactPolicyApprovalPromotion(),
		"flow_matrix":                     validArtifactPolicyFlowMatrix(),
		"revoked_approved_excerpt_hashes": []any{testDigestValue("a")},
		"quotas":                          validArtifactPolicyQuotas(),
		"dependency_cache":                validArtifactPolicyDependencyCache(),
		"retention":                       validArtifactPolicyRetention(),
		"gc":                              validArtifactPolicyGC(),
	}
}

func validArtifactPolicyDependencyCache() map[string]any {
	return map[string]any{
		"read_only_artifacts_required":                 true,
		"batch_manifest_immutable":                     true,
		"resolved_unit_manifest_immutable":             true,
		"resolved_payload_immutable":                   true,
		"materialized_trees_derived_non_canonical":     true,
		"cache_hit_semantics":                          "exact_request_identity_and_resolved_unit_digest",
		"fail_closed_on_ambiguous_or_incomplete_state": true,
		"retain_canonical_before_derived":              true,
	}
}

func validArtifactPolicyCAS() map[string]any {
	return map[string]any{
		"put":             "put(stream)->{hash,size,metadata}",
		"get":             "get(hash)->stream",
		"head":            "head(hash)->metadata",
		"hashing_profile": "RFC8785-JCS",
	}
}

func validArtifactPolicyStorageProtection() map[string]any {
	return map[string]any{
		"encrypted_at_rest_default": true,
		"dev_plaintext_override":    "explicit_dev_only",
	}
}

func validArtifactPolicyApprovalPromotion() map[string]any {
	return map[string]any{
		"explicit_human_approval_required":          true,
		"promotion_mints_new_artifact_reference":    true,
		"max_promotion_request_bytes":               65536,
		"max_promotion_requests_per_minute":         30,
		"bulk_promotion_requires_separate_approval": true,
		"require_full_content_visibility":           "full_content_or_explicit_view_full",
		"require_origin_metadata":                   []any{"repo_path", "commit", "extractor_tool_version"},
		"artifact_data_class_immutability":          true,
		"unapproved_excerpt_egress":                 "deny",
		"approved_excerpt_egress":                   "manifest_opt_in_only",
		"audit_event_type_on_action":                "artifact_promotion_action",
	}
}

func validArtifactPolicyFlowMatrix() []any {
	return []any{map[string]any{"producer_role": "workspace", "consumer_role": "model_gateway", "allowed_data_classes": []any{"spec_text", "approved_file_excerpts"}}}
}

func validArtifactPolicyQuotas() map[string]any {
	return map[string]any{
		"per_role": []any{map[string]any{"scope_id": "workspace", "max_artifact_count": 1024, "max_total_bytes": 268435456, "max_single_artifact_bytes": 16777216, "audit_event_type_on_violation": "artifact_quota_violation"}},
		"per_step": []any{map[string]any{"scope_id": "step-1", "max_artifact_count": 256, "max_total_bytes": 67108864, "max_single_artifact_bytes": 8388608, "audit_event_type_on_violation": "artifact_quota_violation"}},
	}
}

func validArtifactPolicyRetention() map[string]any {
	return map[string]any{
		"retain_referenced_active_runs":         true,
		"retain_referenced_retained_runs":       true,
		"unreferenced_ttl_seconds":              604800,
		"delete_unreferenced_on_quota_pressure": true,
		"audit_event_type_on_action":            "artifact_retention_action",
	}
}

func validArtifactPolicyGC() map[string]any {
	return map[string]any{
		"audit_gc_actions":            true,
		"report_freed_bytes":          true,
		"deterministic_export_format": "hash_manifest_plus_metadata",
		"deterministic_restore_rules": "no_cross_boundary_secret_leakage",
		"audit_event_type_on_action":  "artifact_retention_action",
	}
}

func invalidArtifactPolicyWithNonHashHandoff() map[string]any {
	policy := validArtifactPolicy()
	policy["handoff_reference_mode"] = "inline_payload"
	return policy
}

func invalidArtifactPolicyWithoutExplicitHumanApproval() map[string]any {
	policy := validArtifactPolicy()
	promotion := policy["approval_promotion"].(map[string]any)
	promotion["explicit_human_approval_required"] = false
	return policy
}

func invalidArtifactPolicyWithUnknownFlowDataClass() map[string]any {
	policy := validArtifactPolicy()
	flow := policy["flow_matrix"].([]any)[0].(map[string]any)
	flow["allowed_data_classes"] = []any{"spec_text", "new_unregistered_class"}
	return policy
}

func validPolicyRuleSet() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.PolicyRuleSet",
		"schema_version": "0.1.0",
		"rules": []any{
			map[string]any{
				"rule_id":           "workspace_write_needs_approval",
				"effect":            "require_human_approval",
				"action_kind":       "workspace_write",
				"capability_id":     "workspace_write",
				"reason_code":       "approval_required",
				"details_schema_id": "runecode.protocol.details.policy.workspace-write.v0",
			},
		},
	}
}

func invalidPolicyRuleSetWithUnknownEffect() map[string]any {
	ruleSet := validPolicyRuleSet()
	rules := ruleSet["rules"].([]any)
	rule := rules[0].(map[string]any)
	rule["effect"] = "script_eval"
	return ruleSet
}

func testDigestString(nibble string) string {
	if len(nibble) != 1 || !strings.Contains("0123456789abcdef", nibble) {
		panic("testDigestString requires exactly one lowercase hex nibble")
	}
	return "sha256:" + strings.Repeat(nibble, 64)
}

func validRuntimeImageDescriptor() map[string]any {
	return map[string]any{
		"schema_id":              "runecode.protocol.v0.RuntimeImageDescriptor",
		"schema_version":         "0.2.0",
		"descriptor_digest":      testDigestString("a"),
		"backend_kind":           "microvm",
		"platform_compatibility": map[string]any{"os": "linux", "architecture": "amd64", "acceleration_kind": "kvm"},
		"boot_contract_version":  "v1",
		"component_digests": map[string]any{
			"kernel": testDigestString("b"),
			"rootfs": testDigestString("c"),
		},
		"signing":     map[string]any{"signer_ref": "signer:trusted-ci", "signature_digest": testDigestString("d")},
		"attestation": map[string]any{"measurement_profile": "sev-snp-v1", "expected_measurement_digests": []any{testDigestString("e")}},
	}
}

func invalidRuntimeImageDescriptorWithUnknownBackend() map[string]any {
	descriptor := validRuntimeImageDescriptor()
	descriptor["backend_kind"] = "qemu"
	return descriptor
}

func invalidRuntimeImageDescriptorWithoutComponents() map[string]any {
	descriptor := validRuntimeImageDescriptor()
	delete(descriptor, "component_digests")
	return descriptor
}

func invalidRuntimeImageDescriptorWithBadComponentDigest() map[string]any {
	descriptor := validRuntimeImageDescriptor()
	descriptor["component_digests"] = map[string]any{"kernel": "not-a-digest"}
	return descriptor
}

func invalidRuntimeImageDescriptorWithoutPlatformCompatibility() map[string]any {
	descriptor := validRuntimeImageDescriptor()
	delete(descriptor, "platform_compatibility")
	return descriptor
}

func invalidRuntimeImageDescriptorWithMissingMicroVMKernelRootfs() map[string]any {
	descriptor := validRuntimeImageDescriptor()
	descriptor["component_digests"] = map[string]any{"initrd": testDigestString("f")}
	return descriptor
}

func invalidRuntimeImageDescriptorWithEmptySigningObject() map[string]any {
	descriptor := validRuntimeImageDescriptor()
	descriptor["signing"] = map[string]any{}
	return descriptor
}

func invalidRuntimeImageDescriptorWithBadAttestationDigest() map[string]any {
	descriptor := validRuntimeImageDescriptor()
	descriptor["attestation"] = map[string]any{"measurement_profile": "sev-snp-v1", "expected_measurement_digests": []any{"sha256:INVALID"}}
	return descriptor
}

func validIsolateSessionStartedPayload() map[string]any {
	return map[string]any{
		"schema_id":                        "runecode.protocol.v0.IsolateSessionStartedPayload",
		"schema_version":                   "0.1.0",
		"run_id":                           "run-1",
		"isolate_id":                       "isolate-1",
		"session_id":                       "session-1",
		"backend_kind":                     "microvm",
		"isolation_assurance_level":        "isolated",
		"provisioning_posture":             "tofu",
		"launch_context_digest":            testDigestString("1"),
		"handshake_transcript_hash":        testDigestString("2"),
		"launch_receipt_digest":            testDigestString("3"),
		"runtime_image_descriptor_digest":  testDigestString("4"),
		"applied_hardening_posture_digest": testDigestString("5"),
	}
}

func invalidIsolateSessionStartedPayloadWithBadSchemaID() map[string]any {
	payload := validIsolateSessionStartedPayload()
	payload["schema_id"] = "runecode.protocol.v0.IsolateSessionStarted"
	return payload
}

func invalidIsolateSessionStartedPayloadWithBadBackendKind() map[string]any {
	payload := validIsolateSessionStartedPayload()
	payload["backend_kind"] = "qemu"
	return payload
}

func invalidIsolateSessionStartedPayloadWithBadDigest() map[string]any {
	payload := validIsolateSessionStartedPayload()
	payload["launch_receipt_digest"] = "sha256:INVALID"
	return payload
}

func validIsolateSessionBoundPayload() map[string]any {
	return map[string]any{
		"schema_id":                        "runecode.protocol.v0.IsolateSessionBoundPayload",
		"schema_version":                   "0.1.0",
		"run_id":                           "run-1",
		"isolate_id":                       "isolate-1",
		"session_id":                       "session-1",
		"backend_kind":                     "microvm",
		"isolation_assurance_level":        "isolated",
		"provisioning_posture":             "tofu",
		"launch_context_digest":            testDigestString("1"),
		"handshake_transcript_hash":        testDigestString("2"),
		"session_binding_digest":           testDigestString("6"),
		"runtime_image_descriptor_digest":  testDigestString("4"),
		"applied_hardening_posture_digest": testDigestString("5"),
	}
}

func invalidIsolateSessionBoundPayloadWithBadSchemaID() map[string]any {
	payload := validIsolateSessionBoundPayload()
	payload["schema_id"] = "runecode.protocol.v0.IsolateSessionBound"
	return payload
}

func invalidIsolateSessionBoundPayloadWithBadProvisioningPosture() map[string]any {
	payload := validIsolateSessionBoundPayload()
	payload["provisioning_posture"] = "pending"
	return payload
}

func invalidIsolateSessionBoundPayloadWithBadDigest() map[string]any {
	payload := validIsolateSessionBoundPayload()
	payload["session_binding_digest"] = "not-a-digest"
	return payload
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
