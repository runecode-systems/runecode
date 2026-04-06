package protocolschema

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
		"schema_version":          "0.3.0",
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
		"retention":                       validArtifactPolicyRetention(),
		"gc":                              validArtifactPolicyGC(),
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
