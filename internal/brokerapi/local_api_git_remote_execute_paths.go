package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type workdirExecutionInput struct {
	workdir       string
	env           []string
	record        artifacts.GitRemotePreparedMutationRecord
	repository    gitExecutionRepository
	proof         gitRuntimeProofPayload
	providerToken string
}

func (n *nativeGitRemoteMutationExecutor) executeGitRefUpdate(ctx context.Context, input workdirExecutionInput) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	request, proof, errResp := decodeGitRefUpdateExecutionRequest(input.record, input.proof)
	if errResp != nil {
		return proof, errResp
	}
	proof, errResp = verifyGitRefUpdatePreconditions(ctx, input.workdir, input.env, input.repository, request, proof)
	if errResp != nil {
		return proof, errResp
	}
	if errResp := checkoutGitRefUpdateBase(ctx, input.workdir, input.env, request.TargetRef); errResp != nil {
		return proof, errResp
	}
	if errResp := applyPatchArtifacts(ctx, n.service, input.workdir, input.env, request.ReferencedPatchArtifactDigests); errResp != nil {
		return proof, errResp
	}
	if err := commitWithIntent(ctx, input.workdir, input.env, request.CommitIntent); err != nil {
		return proof, executionFailure("gateway_failure", "internal", "git_commit_failed", err.Error())
	}
	proof, errResp = verifyAndPushGitRefUpdate(ctx, input.workdir, input.env, request, proof)
	if errResp != nil {
		return proof, errResp
	}
	return finalizeSuccessfulExecution(proof), nil
}

func (n *nativeGitRemoteMutationExecutor) executeGitPullRequestCreate(ctx context.Context, input workdirExecutionInput) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	request, proof, errResp := decodeGitPullRequestExecutionRequest(input.record, input.proof)
	if errResp != nil {
		return proof, errResp
	}
	proof, errResp = verifyGitPullRequestBase(ctx, input.workdir, input.env, input.repository, request, proof)
	if errResp != nil {
		return proof, errResp
	}
	if errResp := checkoutGitPullRequestBase(ctx, input.workdir, input.env, request.BaseRef); errResp != nil {
		return proof, errResp
	}
	if errResp := applyPatchArtifacts(ctx, n.service, input.workdir, input.env, request.ReferencedPatchArtifactDigests); errResp != nil {
		return proof, errResp
	}
	if err := commitSimple(ctx, input.workdir, input.env, request.Title, request.Body); err != nil {
		return proof, executionFailure("gateway_failure", "internal", "git_commit_failed", err.Error())
	}
	proof, errResp = verifyAndPushGitPullRequestBranch(ctx, input.workdir, input.env, request, proof)
	if errResp != nil {
		return proof, errResp
	}
	prResult, err := n.createProviderPullRequest(ctx, input.record, input.repository, request, input.providerToken)
	if err != nil {
		return proof, executionFailure("gateway_failure", "internal", "git_pull_request_create_failed", err.Error())
	}
	proof.PullRequestNumber = &prResult.Number
	proof.PullRequestURL = strings.TrimSpace(prResult.URL)
	return finalizeSuccessfulExecution(proof), nil
}

func decodeGitRefUpdateExecutionRequest(record artifacts.GitRemotePreparedMutationRecord, proof gitRuntimeProofPayload) (gitRefUpdateExecuteRequest, gitRuntimeProofPayload, *gitRemoteExecutionError) {
	request := gitRefUpdateExecuteRequest{}
	if err := remarshalValue(record.TypedRequest, &request); err != nil {
		return gitRefUpdateExecuteRequest{}, proof, executionFailure("broker_validation_schema_invalid", "validation", "git_remote_request_invalid", "typed_request decode failed for git_ref_update")
	}
	expectedOldID := strings.TrimSpace(request.ExpectedOldRefHash.Hash)
	proof.PatchArtifactDigests = append([]trustpolicy.Digest{}, request.ReferencedPatchArtifactDigests...)
	proof.ExpectedResultTreeHash = request.ExpectedResultTreeHash
	proof.ExpectedOldObjectID = expectedOldID
	return request, proof, nil
}

func verifyGitRefUpdatePreconditions(ctx context.Context, workdir string, env []string, repo gitExecutionRepository, request gitRefUpdateExecuteRequest, proof gitRuntimeProofPayload) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	observedRemoteOID, err := lsRemoteOID(ctx, workdir, env, repo.remoteURL, request.TargetRef)
	if err != nil {
		return proof, executionFailure("gateway_failure", "internal", "git_remote_query_failed", err.Error())
	}
	observedComparable := comparableObjectIdentity(observedRemoteOID, proof.ExpectedOldObjectID)
	proof.ObservedOldObjectID = observedComparable
	if !strings.EqualFold(observedComparable, proof.ExpectedOldObjectID) {
		proof.DriftDetected = true
		return proof, executionFailureWithState("broker_approval_state_invalid", "auth", "git_remote_drift_detected", "expected old state mismatch during execute", gitRemoteMutationExecutionBlocked)
	}
	if observedRemoteOID == "" {
		return proof, executionFailureWithState("broker_approval_state_invalid", "auth", "git_remote_old_state_missing", "remote target ref does not exist", gitRemoteMutationExecutionBlocked)
	}
	return proof, nil
}

func checkoutGitRefUpdateBase(ctx context.Context, workdir string, env []string, targetRef string) *gitRemoteExecutionError {
	if err := runGit(ctx, workdir, env, nil, "fetch", "--no-tags", "origin", targetRef); err != nil {
		return executionFailure("gateway_failure", "internal", "git_fetch_failed", err.Error())
	}
	if err := runGit(ctx, workdir, env, nil, "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return executionFailure("gateway_failure", "internal", "git_checkout_failed", err.Error())
	}
	return nil
}

func verifyAndPushGitRefUpdate(ctx context.Context, workdir string, env []string, request gitRefUpdateExecuteRequest, proof gitRuntimeProofPayload) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	observedTree, treeErr := verifyExpectedResultTree(ctx, workdir, env, request.ExpectedResultTreeHash)
	proof.ObservedResultTreeHash = observedTree
	if treeErr != nil {
		return proof, treeErr
	}
	if err := runGit(ctx, workdir, env, nil, "push", "origin", "HEAD:"+request.TargetRef); err != nil {
		return proof, executionFailure("gateway_failure", "internal", "git_push_failed", err.Error())
	}
	return proof, nil
}

func decodeGitPullRequestExecutionRequest(record artifacts.GitRemotePreparedMutationRecord, proof gitRuntimeProofPayload) (gitPullRequestCreateExecuteRequest, gitRuntimeProofPayload, *gitRemoteExecutionError) {
	request := gitPullRequestCreateExecuteRequest{}
	if err := remarshalValue(record.TypedRequest, &request); err != nil {
		return gitPullRequestCreateExecuteRequest{}, proof, executionFailure("broker_validation_schema_invalid", "validation", "git_remote_request_invalid", "typed_request decode failed for git_pull_request_create")
	}
	proof.PatchArtifactDigests = append([]trustpolicy.Digest{}, request.ReferencedPatchArtifactDigests...)
	proof.ExpectedResultTreeHash = request.ExpectedResultTreeHash
	return request, proof, nil
}

func verifyGitPullRequestBase(ctx context.Context, workdir string, env []string, repo gitExecutionRepository, request gitPullRequestCreateExecuteRequest, proof gitRuntimeProofPayload) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	baseRemoteOID, err := lsRemoteOID(ctx, workdir, env, repo.remoteURL, request.BaseRef)
	if err != nil {
		return proof, executionFailure("gateway_failure", "internal", "git_remote_query_failed", err.Error())
	}
	if strings.TrimSpace(baseRemoteOID) == "" {
		return proof, executionFailureWithState("broker_approval_state_invalid", "auth", "git_remote_old_state_missing", "base ref missing on remote", gitRemoteMutationExecutionBlocked)
	}
	proof.ExpectedOldObjectID = comparableObjectIdentity(baseRemoteOID, request.HeadCommitOrTreeHash.Hash)
	proof.ObservedOldObjectID = proof.ExpectedOldObjectID
	return proof, nil
}

func checkoutGitPullRequestBase(ctx context.Context, workdir string, env []string, baseRef string) *gitRemoteExecutionError {
	if err := runGit(ctx, workdir, env, nil, "fetch", "--no-tags", "origin", baseRef); err != nil {
		return executionFailure("gateway_failure", "internal", "git_fetch_failed", err.Error())
	}
	if err := runGit(ctx, workdir, env, nil, "checkout", "-B", "runecode-pr-work", "FETCH_HEAD"); err != nil {
		return executionFailure("gateway_failure", "internal", "git_checkout_failed", err.Error())
	}
	return nil
}

func verifyAndPushGitPullRequestBranch(ctx context.Context, workdir string, env []string, request gitPullRequestCreateExecuteRequest, proof gitRuntimeProofPayload) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	observedTree, treeErr := verifyExpectedResultTree(ctx, workdir, env, request.ExpectedResultTreeHash)
	proof.ObservedResultTreeHash = observedTree
	if treeErr != nil {
		return proof, treeErr
	}
	if err := runGit(ctx, workdir, env, nil, "push", "origin", "HEAD:"+request.HeadRef); err != nil {
		return proof, executionFailure("gateway_failure", "internal", "git_push_failed", err.Error())
	}
	return proof, nil
}

func finalizeSuccessfulExecution(proof gitRuntimeProofPayload) gitRuntimeProofPayload {
	proof.DriftDetected = false
	proof.DestructiveRefMutation = false
	return proof
}
