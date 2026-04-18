package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func resolveGitRemoteRequestKind(typedRequest map[string]any) string {
	return strings.TrimSpace(stringField(typedRequest, "request_kind"))
}

func validateGitRemoteRequestKind(requestKind string) error {
	if requestKind == "git_ref_update" || requestKind == "git_pull_request_create" {
		return nil
	}
	return fmt.Errorf("typed_request.request_kind must be git_ref_update or git_pull_request_create")
}

func resolveGitRemoteDestinationRef(req GitRemoteMutationPrepareRequest) string {
	destinationRef := strings.TrimSpace(req.DestinationRef)
	if destinationRef != "" {
		return destinationRef
	}
	return destinationRefFromTypedRequest(req.TypedRequest)
}

func canonicalizeGitRemoteTypedRequest(typedRequest map[string]any) (trustpolicy.Digest, string, error) {
	typedRequestHashIdentity, err := canonicalGitTypedRequestHash(typedRequest)
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request canonical hash failed: %w", err)
	}
	typedRequestHash, err := digestFromIdentity(typedRequestHashIdentity)
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request hash identity invalid: %w", err)
	}
	return typedRequestHash, typedRequestHashIdentity, nil
}

func gitRemoteMutationBaseSummary(typedRequest map[string]any) (GitRemoteMutationDerivedSummary, string, error) {
	requestKind := resolveGitRemoteRequestKind(typedRequest)
	if err := validateGitRemoteRequestKind(requestKind); err != nil {
		return GitRemoteMutationDerivedSummary{}, "", err
	}
	patchDigests, err := digestSliceField(typedRequest, "referenced_patch_artifact_digests")
	if err != nil {
		return GitRemoteMutationDerivedSummary{}, "", fmt.Errorf("typed_request.referenced_patch_artifact_digests invalid: %w", err)
	}
	expectedTree, err := digestField(typedRequest, "expected_result_tree_hash")
	if err != nil {
		return GitRemoteMutationDerivedSummary{}, "", fmt.Errorf("typed_request.expected_result_tree_hash invalid: %w", err)
	}
	return GitRemoteMutationDerivedSummary{
		SchemaID:                      "runecode.protocol.v0.GitRemoteMutationDerivedSummary",
		SchemaVersion:                 "0.1.0",
		ReferencedPatchArtifactHashes: patchDigests,
		ExpectedResultTreeHash:        expectedTree,
	}, requestKind, nil
}

func gitRuntimeProofBase(typedRequestHash trustpolicy.Digest, provider string, patchDigests []trustpolicy.Digest, expectedTree trustpolicy.Digest) gitRuntimeProofPayload {
	return gitRuntimeProofPayload{
		SchemaID:               "runecode.protocol.v0.GitRuntimeProof",
		SchemaVersion:          "0.1.0",
		TypedRequestHash:       typedRequestHash,
		PatchArtifactDigests:   patchDigests,
		ExpectedResultTreeHash: expectedTree,
		ObservedResultTreeHash: expectedTree,
		SparseCheckoutApplied:  true,
		DriftDetected:          false,
		DestructiveRefMutation: false,
		ProviderKind:           provider,
	}
}

func gitRemoteSummaryForRefUpdate(typedRequest map[string]any, summary GitRemoteMutationDerivedSummary) (GitRemoteMutationDerivedSummary, error) {
	summary.RepositoryIdentity = destinationRefFromDescriptorField(typedRequest, "repository_identity")
	targetRef := strings.TrimSpace(stringField(typedRequest, "target_ref"))
	if targetRef == "" {
		return GitRemoteMutationDerivedSummary{}, fmt.Errorf("typed_request.target_ref is required")
	}
	summary.TargetRefs = []string{targetRef}
	summary.CommitSubject = nestedStringField(typedRequest, "commit_intent", "message", "subject")
	return summary, nil
}

func gitRemoteSummaryForPullRequest(typedRequest map[string]any, summary GitRemoteMutationDerivedSummary) (GitRemoteMutationDerivedSummary, error) {
	summary.RepositoryIdentity = destinationRefFromDescriptorField(typedRequest, "base_repository_identity")
	baseRef := strings.TrimSpace(stringField(typedRequest, "base_ref"))
	headRef := strings.TrimSpace(stringField(typedRequest, "head_ref"))
	if baseRef == "" || headRef == "" {
		return GitRemoteMutationDerivedSummary{}, fmt.Errorf("typed_request base_ref and head_ref are required")
	}
	summary.TargetRefs = []string{baseRef, headRef}
	summary.PullRequestTitle = strings.TrimSpace(stringField(typedRequest, "title"))
	summary.PullRequestBaseRef = baseRef
	summary.PullRequestHeadRef = headRef
	return summary, nil
}

func requireGitRemoteSummaryFields(summary GitRemoteMutationDerivedSummary) error {
	if strings.TrimSpace(summary.RepositoryIdentity) == "" {
		return fmt.Errorf("typed_request repository identity is required")
	}
	if len(summary.TargetRefs) == 0 {
		return fmt.Errorf("typed_request target refs are required")
	}
	return nil
}

func gitRuntimeProofForRefUpdate(typedRequest map[string]any, proof gitRuntimeProofPayload) (gitRuntimeProofPayload, error) {
	expectedOldHash, err := digestField(typedRequest, "expected_old_ref_hash")
	if err != nil {
		return gitRuntimeProofPayload{}, fmt.Errorf("typed_request.expected_old_ref_hash is required")
	}
	proof.ExpectedOldObjectID = expectedOldHash.Hash
	proof.ObservedOldObjectID = expectedOldHash.Hash
	return proof, nil
}

func gitRuntimeProofForPullRequest(proof gitRuntimeProofPayload) gitRuntimeProofPayload {
	proof.ExpectedOldObjectID = gitRemoteMutationZeroObjectID
	proof.ObservedOldObjectID = gitRemoteMutationZeroObjectID
	proof.PullRequestURL = "prepared://pending"
	return proof
}
