package brokerapi

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/secretsd"
)

func (n *nativeGitRemoteMutationExecutor) executePreparedMutation(ctx context.Context, req gitRemoteExecutionRequest) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	if n == nil || n.service == nil {
		return gitRuntimeProofPayload{}, executionFailure("gateway_failure", "internal", "git_remote_execution_unavailable", "git remote mutation executor unavailable")
	}
	record := req.Record
	repository, baseProof, errResp := n.prepareExecutionContext(record)
	if errResp != nil {
		return gitRuntimeProofPayload{}, errResp
	}
	providerToken, leaseID, workspaceRoot, taskEnv, errResp := n.prepareExecutionWorkspace(ctx, record, repository, req.ProviderAuthLeaseID)
	if errResp != nil {
		return baseProof, errResp
	}
	defer os.RemoveAll(workspaceRoot)

	proof, execErr := n.executePreparedMutationByKind(ctx, workdirExecutionInput{
		workdir:       workspaceRoot,
		env:           taskEnv,
		record:        record,
		repository:    repository,
		proof:         baseProof,
		providerToken: providerToken,
	})
	if execErr != nil {
		return proof, execErr
	}
	if leaseID != "" {
		proof.EvidenceRefs = append(proof.EvidenceRefs, "lease:"+leaseID)
	}
	return proof, nil
}

func (n *nativeGitRemoteMutationExecutor) prepareExecutionContext(record artifacts.GitRemotePreparedMutationRecord) (gitExecutionRepository, gitRuntimeProofPayload, *gitRemoteExecutionError) {
	repository, decodeErr := decodeExecutionRepository(record)
	if decodeErr != nil {
		return gitExecutionRepository{}, gitRuntimeProofPayload{}, executionFailure("broker_validation_schema_invalid", "validation", "git_remote_request_invalid", decodeErr.Error())
	}
	typedHash, err := digestFromIdentity(record.TypedRequestHash)
	if err != nil {
		return gitExecutionRepository{}, gitRuntimeProofPayload{}, executionFailure("broker_validation_schema_invalid", "validation", "typed_request_hash_invalid", "typed_request_hash invalid")
	}
	baseProof := gitRuntimeProofPayload{
		SchemaID:              "runecode.protocol.v0.GitRuntimeProof",
		SchemaVersion:         "0.1.0",
		TypedRequestHash:      typedHash,
		SparseCheckoutApplied: true,
		ProviderKind:          record.Provider,
		EvidenceRefs:          []string{"prepared_mutation:" + record.PreparedMutationID},
	}
	return repository, baseProof, nil
}

func (n *nativeGitRemoteMutationExecutor) prepareExecutionWorkspace(ctx context.Context, record artifacts.GitRemotePreparedMutationRecord, repository gitExecutionRepository, providerAuthLeaseID string) (string, string, string, []string, *gitRemoteExecutionError) {
	providerToken, leaseID, tokenErr := n.resolveProviderTokenForMutation(record, repository.repositoryIdentity, providerAuthLeaseID)
	if tokenErr != nil {
		return "", "", "", nil, tokenErr
	}
	workspaceRoot, errResp := createGitExecutionWorkspace(ctx, repository.remoteURL)
	if errResp != nil {
		return "", "", "", nil, errResp
	}
	taskEnv, envErr := buildGitCredentialEnv(workspaceRoot, providerToken)
	if envErr != nil {
		_ = os.RemoveAll(workspaceRoot)
		return "", "", "", nil, executionFailure("gateway_failure", "internal", "git_credential_binding_failed", envErr.Error())
	}
	return providerToken, leaseID, workspaceRoot, taskEnv, nil
}

func createGitExecutionWorkspace(ctx context.Context, remoteURL string) (string, *gitRemoteExecutionError) {
	workspaceRoot, err := os.MkdirTemp("", "runecode-git-mutation-")
	if err != nil {
		return "", executionFailure("gateway_failure", "internal", "git_workspace_init_failed", fmt.Sprintf("workspace create failed: %v", err))
	}
	if err := initializeGitWorkspace(ctx, workspaceRoot, remoteURL); err != nil {
		_ = os.RemoveAll(workspaceRoot)
		return "", executionFailure("gateway_failure", "internal", "git_workspace_init_failed", err.Error())
	}
	return workspaceRoot, nil
}

func (n *nativeGitRemoteMutationExecutor) executePreparedMutationByKind(ctx context.Context, input workdirExecutionInput) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	switch strings.TrimSpace(input.record.RequestKind) {
	case "git_ref_update":
		return n.executeGitRefUpdate(ctx, input)
	case "git_pull_request_create":
		return n.executeGitPullRequestCreate(ctx, input)
	default:
		return input.proof, executionFailure("broker_validation_schema_invalid", "validation", "git_remote_request_kind_invalid", "typed_request.request_kind is unsupported")
	}
}

func initializeGitWorkspace(ctx context.Context, workspaceRoot, remoteURL string) error {
	if err := runGit(ctx, workspaceRoot, nil, nil, "init", "."); err != nil {
		return err
	}
	if err := runGit(ctx, workspaceRoot, nil, nil, "remote", "add", "origin", remoteURL); err != nil {
		return err
	}
	if err := runGit(ctx, workspaceRoot, nil, nil, "config", "core.sparseCheckout", "true"); err != nil {
		return err
	}
	return nil
}

func (n *nativeGitRemoteMutationExecutor) resolveProviderTokenForMutation(record artifacts.GitRemotePreparedMutationRecord, repositoryIdentity, leaseID string) (string, string, *gitRemoteExecutionError) {
	if n.service == nil || n.service.secretsSvc == nil {
		return "", "", executionFailureWithState("gateway_failure", "internal", "git_provider_credential_unavailable", "secrets service unavailable for git mutation execution", gitRemoteMutationExecutionFailed)
	}
	trimmedLeaseID := strings.TrimSpace(leaseID)
	if trimmedLeaseID == "" {
		return "", "", executionFailureWithState("broker_validation_schema_invalid", "validation", "git_provider_credential_lease_required", "provider_auth_lease_id is required for git remote execute", gitRemoteMutationExecutionBlocked)
	}
	material, lease, err := n.service.secretsSvc.Retrieve(secretsd.RetrieveRequest{
		LeaseID:      trimmedLeaseID,
		ConsumerID:   "principal:gateway:git:1",
		RoleKind:     "git-gateway",
		Scope:        "run:" + strings.TrimSpace(record.RunID),
		DeliveryKind: "git_gateway",
		GitUseContext: &secretsd.GitLeaseUseContext{
			RepositoryIdentity: strings.TrimSpace(repositoryIdentity),
			Operation:          gitSecretOperationForRequestKind(record.RequestKind),
			ActionRequestHash:  strings.TrimSpace(record.ActionRequestHash),
			PolicyContextHash:  strings.TrimSpace(record.PolicyDecisionHash),
		},
	})
	if err != nil {
		return "", "", executionFailureWithState("broker_approval_state_invalid", "auth", "git_provider_credential_lease_invalid", "provider auth lease retrieval failed for execute", gitRemoteMutationExecutionBlocked)
	}
	token := strings.TrimSpace(string(material))
	if token == "" {
		return "", "", executionFailureWithState("gateway_failure", "internal", "git_provider_credential_empty", "provider token material is empty", gitRemoteMutationExecutionFailed)
	}
	return token, strings.TrimSpace(lease.LeaseID), nil
}

func decodeExecutionRepository(record artifacts.GitRemotePreparedMutationRecord) (gitExecutionRepository, error) {
	repositoryIdentity, err := gitRemoteRepositoryIdentity(record.TypedRequest, record.RequestKind)
	if err != nil {
		return gitExecutionRepository{}, err
	}
	if err := validateGitRemoteDestinationRef(record.DestinationRef, repositoryIdentity); err != nil {
		return gitExecutionRepository{}, err
	}
	host, pathPart := splitHostAndPath(repositoryIdentity)
	if host == "" || pathPart == "" {
		return gitExecutionRepository{}, fmt.Errorf("typed request repository identity must include host and repository path")
	}
	if !strings.HasSuffix(pathPart, ".git") {
		pathPart += ".git"
	}
	canonicalIdentity := host + "/" + strings.TrimPrefix(pathPart, "/")
	remoteURL := "https://" + canonicalIdentity
	return gitExecutionRepository{repositoryIdentity: canonicalIdentity, remoteURL: remoteURL, apiHost: host}, nil
}

func splitHostAndPath(destinationRef string) (string, string) {
	trimmed := strings.TrimSpace(destinationRef)
	if trimmed == "" {
		return "", ""
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), "/" + strings.TrimSpace(parts[1])
}

func gitSecretOperationForRequestKind(requestKind string) string {
	if strings.TrimSpace(requestKind) == "git_pull_request_create" {
		return "create_pull_request"
	}
	return "push_ref"
}

func validateGitRemoteDestinationRef(destinationRef, repositoryIdentity string) error {
	trimmedDestination := strings.TrimSpace(destinationRef)
	trimmedIdentity := strings.TrimSpace(repositoryIdentity)
	if trimmedIdentity == "" {
		return fmt.Errorf("typed request repository identity is required")
	}
	if trimmedDestination == "" {
		return nil
	}
	if !strings.EqualFold(trimmedDestination, trimmedIdentity) {
		return fmt.Errorf("destination_ref must match authoritative typed request repository identity")
	}
	return nil
}

func gitRemoteRepositoryIdentity(typedRequest map[string]any, requestKind string) (string, error) {
	field := "repository_identity"
	if strings.TrimSpace(requestKind) == "git_pull_request_create" {
		field = "base_repository_identity"
	}
	identity := destinationRefFromDescriptorField(typedRequest, field)
	if strings.TrimSpace(identity) == "" {
		return "", fmt.Errorf("typed request repository identity is required")
	}
	return identity, nil
}
