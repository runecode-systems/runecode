package policyengine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func moderateGitRemoteApprovalPayload(base map[string]any, action ActionRequest) (map[string]any, bool) {
	summary := decodeGitRequestSummary(action)
	if summary == nil {
		return nil, false
	}
	boundMutation, repoIdentity, err := moderateGitRemoteBoundMutation(*summary)
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
	payload["scope"] = approvalScopeForGitRemoteAction(action, summary.RequestKind, repoIdentity)
	payload["bound_remote_mutation"] = boundMutation
	return payload, true
}

func moderateGitRemoteBoundMutation(summary gitRequestSummary) (map[string]any, map[string]any, error) {
	repoIdentity := destinationDescriptorIdentity(summary.RepositoryIdentity)
	expectedTree, repoPolicyDigest, err := moderateGitRemoteDigests(summary)
	if err != nil {
		return nil, nil, err
	}
	return map[string]any{
		"request_kind":                      summary.RequestKind,
		"repository_identity":               repoIdentity,
		"target_refs":                       sortedStrings(summary.TargetRefs),
		"referenced_patch_artifact_digests": sortedStrings(digestIdentitySlice(summary.ReferencedPatchArtifactDigests)),
		"expected_result_tree_hash":         expectedTree,
		"metadata_summary":                  gitMetadataSummaryForApproval(summary.MetadataSummary),
		"repository_policy_digest":          repoPolicyDigest,
		"deterministic_required_trailers":   deterministicRequiredTrailers(summary.MetadataSummary),
	}, repoIdentity, nil
}

func moderateGitRemoteDigests(summary gitRequestSummary) (string, string, error) {
	expectedTree, err := summary.ExpectedResultTreeHash.Identity()
	if err != nil {
		return "", "", err
	}
	if summary.MetadataSummary.CommitPolicy == nil {
		return expectedTree, "", nil
	}
	repoPolicyDigest, err := summary.MetadataSummary.CommitPolicy.RepositoryPolicyDigest.Identity()
	if err != nil {
		return "", "", err
	}
	return expectedTree, repoPolicyDigest, nil
}

func approvalScopeForGitRemoteAction(action ActionRequest, requestKind string, repoIdentity map[string]any) map[string]any {
	scope := approvalScopeForAction(action)
	scope["destination_kind"] = "git_remote"
	scope["request_kind"] = requestKind
	scope["repository_identity"] = repoIdentity
	return scope
}

func decodeGitRequestSummary(action ActionRequest) *gitRequestSummary {
	raw, ok := action.ActionPayload["git_request_summary"]
	if !ok {
		return nil
	}
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	summary := gitRequestSummary{}
	if err := decodeActionPayload(rawMap, &summary); err != nil {
		return nil
	}
	return &summary
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

func gitMetadataSummaryForApproval(metadata gitRequestMetadata) map[string]any {
	out := map[string]any{}
	if commit := gitCommitSummaryForApproval(metadata.Commit); commit != nil {
		out["commit"] = commit
	}
	if pullRequest := gitPullRequestSummaryForApproval(metadata.PullRequest); pullRequest != nil {
		out["pull_request"] = pullRequest
	}
	if commitPolicy := gitCommitPolicySummaryForApproval(metadata.CommitPolicy); commitPolicy != nil {
		out["commit_policy"] = commitPolicy
	}
	return out
}

func gitCommitSummaryForApproval(commit *gitCommitMetadata) map[string]any {
	if commit == nil {
		return nil
	}
	return map[string]any{
		"subject":   commit.Subject,
		"author":    gitIdentitySummary(commit.Author),
		"committer": gitIdentitySummary(commit.Committer),
		"signoff":   gitIdentitySummary(commit.Signoff),
	}
}

func gitIdentitySummary(identity gitIdentity) map[string]any {
	return map[string]any{
		"display_name": identity.DisplayName,
		"email":        identity.Email,
	}
}

func gitPullRequestSummaryForApproval(pullRequest *gitPullRequestMetadata) map[string]any {
	if pullRequest == nil {
		return nil
	}
	return map[string]any{
		"title":    pullRequest.Title,
		"base_ref": pullRequest.BaseRef,
		"head_ref": pullRequest.HeadRef,
	}
}

func gitCommitPolicySummaryForApproval(commitPolicy *gitCommitPolicy) map[string]any {
	if commitPolicy == nil {
		return nil
	}
	return map[string]any{
		"required_trailer_rules": gitRequiredTrailerRuleSummaries(commitPolicy.RequiredTrailerRules),
	}
}

func gitRequiredTrailerRuleSummaries(rules []gitRequiredTrailer) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for i := range rules {
		out = append(out, map[string]any{
			"trailer_key":   rules[i].TrailerKey,
			"identity_role": rules[i].IdentityRole,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		ik, _ := out[i]["trailer_key"].(string)
		jk, _ := out[j]["trailer_key"].(string)
		if ik != jk {
			return ik < jk
		}
		ii, _ := out[i]["identity_role"].(string)
		ji, _ := out[j]["identity_role"].(string)
		return ii < ji
	})
	return out
}

func deterministicRequiredTrailers(metadata gitRequestMetadata) []string {
	if metadata.Commit == nil || metadata.CommitPolicy == nil {
		return []string{}
	}
	trailers := make([]string, 0, len(metadata.CommitPolicy.RequiredTrailerRules))
	for i := range metadata.CommitPolicy.RequiredTrailerRules {
		trailer, ok := deterministicRequiredTrailer(*metadata.Commit, metadata.CommitPolicy.RequiredTrailerRules[i])
		if !ok {
			continue
		}
		trailers = append(trailers, trailer)
	}
	return sortedStrings(trailers)
}

func deterministicRequiredTrailer(commit gitCommitMetadata, rule gitRequiredTrailer) (string, bool) {
	identity, ok := gitIdentityForRole(commit, rule.IdentityRole)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s: %s <%s>", rule.TrailerKey, identity.DisplayName, identity.Email), true
}

func gitIdentityForRole(commit gitCommitMetadata, role string) (gitIdentity, bool) {
	switch role {
	case "author":
		return commit.Author, true
	case "committer":
		return commit.Committer, true
	case "signoff":
		return commit.Signoff, true
	default:
		return gitIdentity{}, false
	}
}

func sortedStrings(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}
