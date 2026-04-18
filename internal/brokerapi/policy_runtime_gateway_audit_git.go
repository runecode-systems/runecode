package brokerapi

import "strings"

func addGitRuntimeProofAuditDetails(details map[string]any, payload gatewayActionPayloadRuntime) {
	if payload.GitRuntimeProof == nil {
		return
	}
	proof := payload.GitRuntimeProof
	addGitRuntimeDigestDetails(details, proof)
	addGitRuntimeStateDetails(details, proof)
	addGitRuntimeOutcomeDetails(details, proof)
	addGitPatchArtifactDetails(details, proof)
}

func addGitRuntimeDigestDetails(details map[string]any, proof *gitRuntimeProofPayload) {
	addGatewayDigestIdentityValue(details, "typed_request_hash", proof.TypedRequestHash)
	addGatewayDigestIdentityValue(details, "expected_result_tree_hash", proof.ExpectedResultTreeHash)
	addGatewayDigestIdentityValue(details, "observed_result_tree_hash", proof.ObservedResultTreeHash)
}

func addGitRuntimeStateDetails(details map[string]any, proof *gitRuntimeProofPayload) {
	if proof.ExpectedOldObjectID != "" {
		details["expected_old_object_id"] = proof.ExpectedOldObjectID
	}
	if proof.ObservedOldObjectID != "" {
		details["observed_old_object_id"] = proof.ObservedOldObjectID
	}
	details["sparse_checkout_applied"] = proof.SparseCheckoutApplied
	details["drift_detected"] = proof.DriftDetected
	details["destructive_ref_mutation"] = proof.DestructiveRefMutation
}

func addGitRuntimeOutcomeDetails(details map[string]any, proof *gitRuntimeProofPayload) {
	if strings.TrimSpace(proof.ProviderKind) != "" {
		details["provider_kind"] = proof.ProviderKind
	}
	if proof.PullRequestNumber != nil {
		details["pull_request_number"] = *proof.PullRequestNumber
	}
	if strings.TrimSpace(proof.PullRequestURL) != "" {
		details["pull_request_url"] = proof.PullRequestURL
	}
	if len(proof.EvidenceRefs) > 0 {
		details["evidence_refs"] = append([]string{}, proof.EvidenceRefs...)
	}
}

func addGitPatchArtifactDetails(details map[string]any, proof *gitRuntimeProofPayload) {
	if len(proof.PatchArtifactDigests) == 0 {
		return
	}
	digests := digestIdentitySlice(proof.PatchArtifactDigests)
	if len(digests) > 0 {
		details["patch_artifact_digests"] = digests
	}
}
