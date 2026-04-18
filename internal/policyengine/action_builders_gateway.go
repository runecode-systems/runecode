package policyengine

import "github.com/runecode-ai/runecode/internal/trustpolicy"

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
	if input.GitRequest != nil {
		payload["git_request"] = buildGitTypedRequestPayload(*input.GitRequest)
	}
	if input.GitRuntimeProof != nil {
		payload["git_runtime_proof"] = buildGitRuntimeProofPayload(*input.GitRuntimeProof)
	}
	if input.QuotaContext != nil {
		payload["quota_context"] = buildGatewayQuotaPayload(*input.QuotaContext)
	}
	return payload
}

func buildGitTypedRequestPayload(input GitTypedRequestInput) map[string]any {
	if input.RefUpdate != nil {
		return buildGitRefUpdateRequestPayload(*input.RefUpdate)
	}
	if input.PullRequestCreate != nil {
		return buildGitPullRequestCreateRequestPayload(*input.PullRequestCreate)
	}
	return map[string]any{}
}

func buildDestinationDescriptorPayload(input DestinationDescriptor) map[string]any {
	payload := map[string]any{
		"schema_id":                destinationDescriptorSchemaID,
		"schema_version":           destinationDescriptorVersion,
		"descriptor_kind":          input.DescriptorKind,
		"canonical_host":           input.CanonicalHost,
		"tls_required":             input.TLSRequired,
		"private_range_blocking":   input.PrivateRangeBlocking,
		"dns_rebinding_protection": input.DNSRebindingProtection,
	}
	if input.CanonicalPort != nil {
		payload["canonical_port"] = *input.CanonicalPort
	}
	if input.CanonicalPathPrefix != "" {
		payload["canonical_path_prefix"] = input.CanonicalPathPrefix
	}
	if input.ProviderOrNamespace != "" {
		payload["provider_or_namespace"] = input.ProviderOrNamespace
	}
	if input.GitRepositoryIdentity != "" {
		payload["git_repository_identity"] = input.GitRepositoryIdentity
	}
	return payload
}

func buildGitRuntimeProofPayload(input GitRuntimeProofInput) map[string]any {
	proof := map[string]any{
		"schema_id":                 "runecode.protocol.v0.GitRuntimeProof",
		"schema_version":            "0.1.0",
		"typed_request_hash":        input.TypedRequestHash,
		"expected_old_object_id":    input.ExpectedOldObjectID,
		"observed_old_object_id":    input.ObservedOldObjectID,
		"expected_result_tree_hash": input.ExpectedResultTreeHash,
		"observed_result_tree_hash": input.ObservedResultTreeHash,
		"sparse_checkout_applied":   input.SparseCheckoutApplied,
		"drift_detected":            input.DriftDetected,
		"destructive_ref_mutation":  input.DestructiveRefMutation,
		"evidence_refs":             append([]string{}, input.EvidenceRefs...),
	}
	if input.ProviderKind != "" {
		proof["provider_kind"] = input.ProviderKind
	}
	if input.PullRequestNumber != nil {
		proof["pull_request_number"] = *input.PullRequestNumber
	}
	if input.PullRequestURL != "" {
		proof["pull_request_url"] = input.PullRequestURL
	}
	if len(input.PatchArtifactDigests) > 0 {
		values := make([]any, 0, len(input.PatchArtifactDigests))
		for _, digest := range input.PatchArtifactDigests {
			values = append(values, digest)
		}
		proof["patch_artifact_digests"] = values
	}
	return proof
}

func buildGitRefUpdateRequestPayload(input GitRefUpdateRequestInput) map[string]any {
	payload := map[string]any{
		"schema_id":                         "runecode.protocol.v0.GitRefUpdateRequest",
		"schema_version":                    "0.1.0",
		"request_kind":                      "git_ref_update",
		"repository_identity":               buildDestinationDescriptorPayload(input.RepositoryIdentity),
		"target_ref":                        input.TargetRef,
		"expected_old_ref_hash":             input.ExpectedOldRefHash,
		"commit_intent":                     buildGitCommitIntentPayload(input.CommitIntent),
		"expected_result_tree_hash":         input.ExpectedResultTreeHash,
		"allow_force_push":                  input.AllowForcePush,
		"allow_ref_deletion":                input.AllowRefDeletion,
		"referenced_patch_artifact_digests": digestSliceToAny(input.ReferencedPatchArtifactDigests),
	}
	if input.RefPurpose != "" {
		payload["ref_purpose"] = input.RefPurpose
	}
	if input.BaseRef != "" {
		payload["base_ref"] = input.BaseRef
	}
	return payload
}

func buildGitPullRequestCreateRequestPayload(input GitPullRequestCreateRequestInput) map[string]any {
	return map[string]any{
		"schema_id":                         "runecode.protocol.v0.GitPullRequestCreateRequest",
		"schema_version":                    "0.1.0",
		"request_kind":                      "git_pull_request_create",
		"base_repository_identity":          buildDestinationDescriptorPayload(input.BaseRepositoryIdentity),
		"base_ref":                          input.BaseRef,
		"head_repository_identity":          buildDestinationDescriptorPayload(input.HeadRepositoryIdentity),
		"head_ref":                          input.HeadRef,
		"title":                             input.Title,
		"body":                              input.Body,
		"head_commit_or_tree_hash":          input.HeadCommitOrTreeHash,
		"referenced_patch_artifact_digests": digestSliceToAny(input.ReferencedPatchArtifactDigests),
		"expected_result_tree_hash":         input.ExpectedResultTreeHash,
	}
}

func buildGitCommitIntentPayload(input GitCommitIntentInput) map[string]any {
	trailers := make([]any, 0, len(input.Trailers))
	for _, trailer := range input.Trailers {
		trailers = append(trailers, map[string]any{"key": trailer.Key, "value": trailer.Value})
	}
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.GitCommitIntent",
		"schema_version": "0.1.0",
		"message": map[string]any{
			"subject": input.Message.Subject,
			"body":    input.Message.Body,
		},
		"trailers":  trailers,
		"author":    map[string]any{"display_name": input.Author.DisplayName, "email": input.Author.Email},
		"committer": map[string]any{"display_name": input.Committer.DisplayName, "email": input.Committer.Email},
		"signoff":   map[string]any{"display_name": input.Signoff.DisplayName, "email": input.Signoff.Email},
	}
}

func digestSliceToAny(digests []trustpolicy.Digest) []any {
	values := make([]any, 0, len(digests))
	for _, digest := range digests {
		values = append(values, digest)
	}
	return values
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
