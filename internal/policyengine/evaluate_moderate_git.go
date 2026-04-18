package policyengine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func moderateGitRemoteApprovalPayload(base map[string]any, action ActionRequest) (map[string]any, bool) {
	request := decodeGitTypedRequest(action)
	if request == nil {
		return nil, false
	}
	boundMutation, repoIdentity, requestKind, err := moderateGitRemoteBoundMutation(request)
	if err != nil {
		return nil, false
	}
	payload := cloneMap(base)
	payload["approval_trigger_code"] = "git_remote_ops"
	payload["approval_assurance_level"] = string(ApprovalAssuranceReauthenticated)
	payload["checkpoint_scope"] = "gateway_remote_state_mutation"
	payload["why_required"] = "Git remote mutation requires exact final approval and cannot be authorized by stage sign-off alone."
	payload["changes_if_approved"] = "One exact git remote mutation request may proceed for this bound repository, refs, patch set, and expected result tree hash."
	payload["security_posture_impact"] = "high"
	payload["required_final_exact_approval"] = true
	payload["stage_sign_off_is_prerequisite_only"] = true
	payload["scope"] = approvalScopeForGitRemoteAction(action, requestKind, repoIdentity)
	payload["bound_remote_mutation"] = boundMutation
	return payload, true
}

func moderateGitRemoteBoundMutation(request map[string]any) (map[string]any, map[string]any, string, error) {
	requestKind, _ := request["request_kind"].(string)
	switch requestKind {
	case "git_ref_update":
		return moderateGitRefUpdateBoundMutation(request)
	case "git_pull_request_create":
		return moderateGitPullRequestBoundMutation(request)
	default:
		return nil, nil, "", fmt.Errorf("unsupported request_kind %q", requestKind)
	}
}

func approvalScopeForGitRemoteAction(action ActionRequest, requestKind string, repoIdentity map[string]any) map[string]any {
	scope := approvalScopeForAction(action)
	scope["destination_kind"] = "git_remote"
	scope["request_kind"] = requestKind
	scope["repository_identity"] = repoIdentity
	return scope
}

func decodeGitTypedRequest(action ActionRequest) map[string]any {
	raw, ok := action.ActionPayload["git_request"]
	if !ok {
		return nil
	}
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	b, err := json.Marshal(rawMap)
	if err != nil {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}

func moderateGitRefUpdateBoundMutation(request map[string]any) (map[string]any, map[string]any, string, error) {
	decoded := gitRefUpdateRequest{}
	if err := decodeActionPayload(request, &decoded); err != nil {
		return nil, nil, "", err
	}
	repoIdentity := destinationDescriptorIdentity(decoded.RepositoryIdentity)
	expectedTree, err := decoded.ExpectedResultTreeHash.Identity()
	if err != nil {
		return nil, nil, "", err
	}
	commitSummary := map[string]any{
		"subject":   decoded.CommitIntent.Message.Subject,
		"author":    gitIdentitySummary(decoded.CommitIntent.Author),
		"committer": gitIdentitySummary(decoded.CommitIntent.Committer),
		"signoff":   gitIdentitySummary(decoded.CommitIntent.Signoff),
	}
	trailers := deterministicTrailerLines(decoded.CommitIntent.Trailers)
	return map[string]any{
		"request_kind":                      decoded.RequestKind,
		"repository_identity":               repoIdentity,
		"target_refs":                       []string{decoded.TargetRef},
		"referenced_patch_artifact_digests": sortedStrings(digestIdentitySlice(decoded.ReferencedPatchArtifactDigests)),
		"expected_result_tree_hash":         expectedTree,
		"metadata_summary": map[string]any{
			"commit": commitSummary,
		},
		"deterministic_required_trailers": trailers,
	}, repoIdentity, decoded.RequestKind, nil
}

func moderateGitPullRequestBoundMutation(request map[string]any) (map[string]any, map[string]any, string, error) {
	decoded := gitPullRequestCreateRequest{}
	if err := decodeActionPayload(request, &decoded); err != nil {
		return nil, nil, "", err
	}
	baseRepoIdentity := destinationDescriptorIdentity(decoded.BaseRepositoryIdentity)
	headRepoIdentity := destinationDescriptorIdentity(decoded.HeadRepositoryIdentity)
	expectedTree, err := decoded.ExpectedResultTreeHash.Identity()
	if err != nil {
		return nil, nil, "", err
	}
	return map[string]any{
		"request_kind":                      decoded.RequestKind,
		"repository_identity":               baseRepoIdentity,
		"head_repository_identity":          headRepoIdentity,
		"target_refs":                       []string{decoded.BaseRef, decoded.HeadRef},
		"referenced_patch_artifact_digests": sortedStrings(digestIdentitySlice(decoded.ReferencedPatchArtifactDigests)),
		"expected_result_tree_hash":         expectedTree,
		"metadata_summary": map[string]any{
			"pull_request": map[string]any{
				"title":    decoded.Title,
				"base_ref": decoded.BaseRef,
				"head_ref": decoded.HeadRef,
			},
		},
		"deterministic_required_trailers": []string{},
	}, baseRepoIdentity, decoded.RequestKind, nil
}

func destinationDescriptorIdentity(repo DestinationDescriptor) map[string]any {
	identity := map[string]any{
		"descriptor_kind": repo.DescriptorKind,
		"canonical_host":  repo.CanonicalHost,
	}
	if repo.CanonicalPort != nil {
		identity["canonical_port"] = *repo.CanonicalPort
	}
	if strings.TrimSpace(repo.CanonicalPathPrefix) != "" {
		identity["canonical_path_prefix"] = repo.CanonicalPathPrefix
	}
	if strings.TrimSpace(repo.ProviderOrNamespace) != "" {
		identity["provider_or_namespace"] = repo.ProviderOrNamespace
	}
	if strings.TrimSpace(repo.GitRepositoryIdentity) != "" {
		identity["git_repository_identity"] = repo.GitRepositoryIdentity
	}
	return identity
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

func gitIdentitySummary(identity gitIdentity) map[string]any {
	return map[string]any{
		"display_name": identity.DisplayName,
		"email":        identity.Email,
	}
}

func deterministicTrailerLines(trailers []gitCommitTrailer) []string {
	if len(trailers) == 0 {
		return []string{}
	}
	lines := make([]string, 0, len(trailers))
	for i := range trailers {
		lines = append(lines, fmt.Sprintf("%s: %s", trailers[i].Key, trailers[i].Value))
	}
	return sortedStrings(lines)
}

func sortedStrings(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}
