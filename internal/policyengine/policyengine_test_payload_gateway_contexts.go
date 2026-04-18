package policyengine

import "strings"

func gatewayPayloadHash() map[string]any {
	return mustDigestObject("sha256:" + strings.Repeat("f", 64))
}

func validGatewayAuditContext(requestHash map[string]any) map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.GatewayAuditContext",
		"schema_version":       "0.1.0",
		"outbound_bytes":       float64(1024),
		"started_at":           "2026-03-13T12:00:00Z",
		"completed_at":         "2026-03-13T12:00:01Z",
		"outcome":              "succeeded",
		"request_hash":         requestHash,
		"response_hash":        mustDigestObject("sha256:" + strings.Repeat("a", 64)),
		"lease_id":             "lease-model-1",
		"policy_decision_hash": mustDigestObject("sha256:" + strings.Repeat("b", 64)),
	}
}

func validGatewayQuotaContextTokenMetered() map[string]any {
	return map[string]any{
		"schema_id":             "runecode.protocol.v0.GatewayQuotaContext",
		"schema_version":        "0.1.0",
		"quota_profile_kind":    "token_metered_api",
		"phase":                 "admission",
		"enforce_during_stream": false,
		"meters": map[string]any{
			"input_tokens":  float64(512),
			"output_tokens": float64(128),
		},
	}
}

func validGatewayDependencyAuditContext(requestHash map[string]any) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GatewayAuditContext",
		"schema_version": "0.1.0",
		"outbound_bytes": float64(4096),
		"started_at":     "2026-03-13T12:00:00Z",
		"completed_at":   "2026-03-13T12:00:01Z",
		"outcome":        "succeeded",
		"request_hash":   requestHash,
		"response_hash":  mustDigestObject("sha256:" + strings.Repeat("d", 64)),
	}
}

func validGitRemoteMutationAuditContext(payloadHash map[string]any) map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GatewayAuditContext",
		"schema_version": "0.1.0",
		"outbound_bytes": float64(2048),
		"started_at":     "2026-03-13T12:00:00Z",
		"completed_at":   "2026-03-13T12:00:02Z",
		"outcome":        "succeeded",
		"request_hash":   payloadHash,
		"response_hash":  mustDigestObject("sha256:" + strings.Repeat("8", 64)),
	}
}

func validGitRemoteMutationRuntimeProof(payloadHash map[string]any) map[string]any {
	return map[string]any{
		"schema_id":                 "runecode.protocol.v0.GitRuntimeProof",
		"schema_version":            "0.1.0",
		"typed_request_hash":        payloadHash,
		"expected_old_object_id":    strings.Repeat("a", 40),
		"observed_old_object_id":    strings.Repeat("a", 40),
		"patch_artifact_digests":    []any{mustDigestObject("sha256:" + strings.Repeat("7", 64))},
		"expected_result_tree_hash": mustDigestObject("sha256:" + strings.Repeat("6", 64)),
		"observed_result_tree_hash": mustDigestObject("sha256:" + strings.Repeat("6", 64)),
		"sparse_checkout_applied":   true,
		"drift_detected":            false,
		"destructive_ref_mutation":  false,
		"provider_kind":             "github",
		"evidence_refs":             []any{"artifact:gate-result"},
	}
}

func validGitRemoteMutationSummary() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GitRemoteMutationSummary",
		"schema_version": "0.1.0",
		"request_kind":   "git_ref_update",
		"repository_identity": map[string]any{
			"schema_id":                destinationDescriptorSchemaID,
			"schema_version":           destinationDescriptorVersion,
			"descriptor_kind":          "git_remote",
			"canonical_host":           "git.example.com",
			"git_repository_identity":  "git.example.com/org/repo",
			"provider_or_namespace":    "org/repo",
			"tls_required":             true,
			"private_range_blocking":   "enforced",
			"dns_rebinding_protection": "enforced",
		},
		"target_refs":                       []any{"refs/heads/main"},
		"referenced_patch_artifact_digests": []any{mustDigestObject("sha256:" + strings.Repeat("7", 64))},
		"expected_result_tree_hash":         mustDigestObject("sha256:" + strings.Repeat("6", 64)),
		"metadata_summary": map[string]any{
			"commit": map[string]any{
				"subject":   "Apply approved patch",
				"author":    map[string]any{"display_name": "Author Example", "email": "author@example.com"},
				"committer": map[string]any{"display_name": "Committer Example", "email": "committer@example.com"},
				"signoff":   map[string]any{"display_name": "Signoff Example", "email": "signoff@example.com"},
			},
			"commit_policy": map[string]any{
				"repository_policy_digest": mustDigestObject("sha256:" + strings.Repeat("5", 64)),
				"required_trailer_rules":   []any{map[string]any{"trailer_key": "Signed-off-by", "identity_role": "signoff"}},
			},
		},
	}
}
