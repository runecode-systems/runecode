package brokerapi

import (
	"encoding/json"
	"sort"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func isGatewayRemoteMutationOperation(operation string) bool {
	switch operation {
	case "git_ref_update", "git_pull_request_create":
		return true
	default:
		return false
	}
}

func runtimeGitOutboundVerificationReason(payload gatewayActionPayloadRuntime) (string, map[string]any, bool) {
	if !isGatewayRemoteMutationOperation(payload.Operation) {
		return "", nil, false
	}
	summary, proof, reason, details, denied := runtimeGitBoundMutationPayload(payload)
	if denied {
		return reason, details, true
	}
	if reason, details, denied = runtimeGitResultTreeReason(*summary, *proof); denied {
		return reason, details, true
	}
	if reason, details, denied = runtimeGitRemoteStateReason(*proof); denied {
		return reason, details, true
	}
	if reason, details := runtimeGitPatchDigestBindingReason(*summary, *proof); reason != "" {
		return reason, details, true
	}
	return runtimeGitPullRequestOutcomeReason(payload.Operation, *proof)
}

func runtimeGitBoundMutationPayload(payload gatewayActionPayloadRuntime) (*gitRequestSummaryPayload, *gitRuntimeProofPayload, string, map[string]any, bool) {
	if payload.PayloadHash == nil {
		return nil, nil, "runtime_git_payload_hash_missing", nil, true
	}
	if payload.GitRequest == nil {
		return nil, nil, "runtime_git_request_summary_missing", nil, true
	}
	if payload.GitRuntimeProof == nil {
		return nil, nil, "runtime_git_runtime_proof_missing", nil, true
	}
	requestHash, err := canonicalGitRequestSummaryHash(*payload.GitRequest)
	if err != nil {
		return nil, nil, "runtime_git_request_summary_hash_invalid", map[string]any{"error": err.Error()}, true
	}
	payloadHash, err := payload.PayloadHash.Identity()
	if err != nil {
		return nil, nil, "runtime_git_payload_hash_invalid", nil, true
	}
	if payloadHash != requestHash {
		return nil, nil, "runtime_git_payload_hash_not_bound_to_request_summary", map[string]any{"payload_hash": payloadHash, "request_summary_hash": requestHash}, true
	}
	proofRequestHash, err := payload.GitRuntimeProof.TypedRequestHash.Identity()
	if err != nil {
		return nil, nil, "runtime_git_typed_request_hash_invalid", nil, true
	}
	if proofRequestHash != requestHash {
		return nil, nil, "runtime_git_typed_request_hash_not_bound", map[string]any{"typed_request_hash": proofRequestHash, "request_summary_hash": requestHash}, true
	}
	return payload.GitRequest, payload.GitRuntimeProof, "", nil, false
}

func runtimeGitResultTreeReason(summary gitRequestSummaryPayload, proof gitRuntimeProofPayload) (string, map[string]any, bool) {
	expectedTree, err := summary.ExpectedResultTreeHash.Identity()
	if err != nil {
		return "runtime_git_expected_result_tree_hash_invalid", nil, true
	}
	proofExpectedTree, err := proof.ExpectedResultTreeHash.Identity()
	if err != nil {
		return "runtime_git_proof_expected_result_tree_hash_invalid", nil, true
	}
	if expectedTree != proofExpectedTree {
		return "runtime_git_expected_result_tree_hash_mismatch", map[string]any{"request_expected_result_tree_hash": expectedTree, "proof_expected_result_tree_hash": proofExpectedTree}, true
	}
	observedTree, err := proof.ObservedResultTreeHash.Identity()
	if err != nil {
		return "runtime_git_observed_result_tree_hash_invalid", nil, true
	}
	if observedTree != expectedTree {
		return "runtime_git_observed_result_tree_hash_mismatch", map[string]any{"expected_result_tree_hash": expectedTree, "observed_result_tree_hash": observedTree}, true
	}
	return "", nil, false
}

func runtimeGitRemoteStateReason(proof gitRuntimeProofPayload) (string, map[string]any, bool) {
	if proof.ExpectedOldObjectID == "" || proof.ObservedOldObjectID == "" {
		return "runtime_git_expected_old_state_missing", nil, true
	}
	if proof.ExpectedOldObjectID != proof.ObservedOldObjectID {
		return "runtime_git_remote_drift_detected", map[string]any{"expected_old_object_id": proof.ExpectedOldObjectID, "observed_old_object_id": proof.ObservedOldObjectID}, true
	}
	if proof.DriftDetected {
		return "runtime_git_remote_drift_detected", nil, true
	}
	if !proof.SparseCheckoutApplied {
		return "runtime_git_sparse_checkout_required", nil, true
	}
	if proof.DestructiveRefMutation {
		return "runtime_git_destructive_ref_mutation_denied", nil, true
	}
	return "", nil, false
}

func runtimeGitPullRequestOutcomeReason(operation string, proof gitRuntimeProofPayload) (string, map[string]any, bool) {
	if operation != "git_pull_request_create" {
		return "", nil, false
	}
	if proof.ProviderKind == "" {
		return "runtime_git_pull_request_provider_missing", nil, true
	}
	if proof.PullRequestNumber == nil && proof.PullRequestURL == "" {
		return "runtime_git_pull_request_outcome_missing", nil, true
	}
	return "", nil, false
}

func canonicalGitRequestSummaryHash(summary gitRequestSummaryPayload) (string, error) {
	b, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalHashBytes(b)
}

func runtimeGitPatchDigestBindingReason(summary gitRequestSummaryPayload, proof gitRuntimeProofPayload) (string, map[string]any) {
	summaryDigests := digestIdentitySlice(summary.ReferencedPatchArtifactDigests)
	proofDigests := digestIdentitySlice(proof.PatchArtifactDigests)
	if len(summaryDigests) == 0 {
		return "runtime_git_patch_artifact_digests_missing", nil
	}
	if len(proofDigests) == 0 {
		return "runtime_git_patch_artifact_digests_missing", nil
	}
	sort.Strings(summaryDigests)
	sort.Strings(proofDigests)
	if len(summaryDigests) != len(proofDigests) {
		return "runtime_git_patch_artifact_digests_mismatch", map[string]any{"request_patch_artifact_digests": summaryDigests, "proof_patch_artifact_digests": proofDigests}
	}
	for i := range summaryDigests {
		if summaryDigests[i] != proofDigests[i] {
			return "runtime_git_patch_artifact_digests_mismatch", map[string]any{"request_patch_artifact_digests": summaryDigests, "proof_patch_artifact_digests": proofDigests}
		}
	}
	return "", nil
}

func digestIdentitySlice(digests []trustpolicy.Digest) []string {
	out := make([]string, 0, len(digests))
	for i := range digests {
		id, err := digests[i].Identity()
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out
}
