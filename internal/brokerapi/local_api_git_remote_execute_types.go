package brokerapi

import (
	"context"
	"net/http"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type gitRemoteMutationExecutor interface {
	executePreparedMutation(ctx context.Context, req gitRemoteExecutionRequest) (gitRuntimeProofPayload, *gitRemoteExecutionError)
}

type gitRemoteExecutionRequest struct {
	Record              artifacts.GitRemotePreparedMutationRecord
	ProviderAuthLeaseID string
}

type gitRemoteExecutionError struct {
	code           string
	category       string
	reasonCode     string
	message        string
	retryable      bool
	executionState string
}

type nativeGitRemoteMutationExecutor struct {
	service    *Service
	httpClient *http.Client
}

type gitExecutionRepository struct {
	repositoryIdentity string
	remoteURL          string
	apiHost            string
}

type gitRefUpdateExecuteRequest struct {
	SchemaID                       string                `json:"schema_id"`
	SchemaVersion                  string                `json:"schema_version"`
	RequestKind                    string                `json:"request_kind"`
	RepositoryIdentity             destinationDescriptor `json:"repository_identity"`
	TargetRef                      string                `json:"target_ref"`
	ExpectedOldRefHash             trustpolicy.Digest    `json:"expected_old_ref_hash"`
	ReferencedPatchArtifactDigests []trustpolicy.Digest  `json:"referenced_patch_artifact_digests"`
	CommitIntent                   gitCommitIntent       `json:"commit_intent"`
	ExpectedResultTreeHash         trustpolicy.Digest    `json:"expected_result_tree_hash"`
	AllowForcePush                 bool                  `json:"allow_force_push"`
	AllowRefDeletion               bool                  `json:"allow_ref_deletion"`
}

type gitPullRequestCreateExecuteRequest struct {
	SchemaID                       string                `json:"schema_id"`
	SchemaVersion                  string                `json:"schema_version"`
	RequestKind                    string                `json:"request_kind"`
	BaseRepositoryIdentity         destinationDescriptor `json:"base_repository_identity"`
	BaseRef                        string                `json:"base_ref"`
	HeadRepositoryIdentity         destinationDescriptor `json:"head_repository_identity"`
	HeadRef                        string                `json:"head_ref"`
	Title                          string                `json:"title"`
	Body                           string                `json:"body"`
	HeadCommitOrTreeHash           trustpolicy.Digest    `json:"head_commit_or_tree_hash"`
	ReferencedPatchArtifactDigests []trustpolicy.Digest  `json:"referenced_patch_artifact_digests"`
	ExpectedResultTreeHash         trustpolicy.Digest    `json:"expected_result_tree_hash"`
}

type destinationDescriptor struct {
	CanonicalHost         string `json:"canonical_host"`
	CanonicalPort         *int   `json:"canonical_port,omitempty"`
	CanonicalPathPrefix   string `json:"canonical_path_prefix,omitempty"`
	GitRepositoryIdentity string `json:"git_repository_identity,omitempty"`
}

type gitCommitIntent struct {
	Message struct {
		Subject string `json:"subject"`
		Body    string `json:"body,omitempty"`
	} `json:"message"`
	Trailers []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"trailers"`
	Author struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	} `json:"author"`
	Committer struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	} `json:"committer"`
	Signoff struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	} `json:"signoff"`
}

type gitPullRequestProviderAdapter interface {
	CreatePullRequest(ctx context.Context, req gitPullRequestProviderRequest) (gitPullRequestProviderResult, error)
}

type gitPullRequestProviderRequest struct {
	Host          string
	Repository    string
	BaseRef       string
	HeadRef       string
	HeadOwner     string
	Title         string
	Body          string
	Token         string
	ExpectedState trustpolicy.Digest
}

type gitPullRequestProviderResult struct {
	Number int64
	URL    string
}

type gitHubPullRequestAdapter struct {
	httpClient      *http.Client
	apiBaseOverride string
}

func newNativeGitRemoteMutationExecutor(s *Service) *nativeGitRemoteMutationExecutor {
	return &nativeGitRemoteMutationExecutor{
		service:    s,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *Service) effectiveGitRemoteMutationExecutor() gitRemoteMutationExecutor {
	if s != nil && s.gitMutationExecutor != nil {
		return s.gitMutationExecutor
	}
	return newNativeGitRemoteMutationExecutor(s)
}

func (s *Service) executePreparedGitRemoteMutation(ctx context.Context, record artifacts.GitRemotePreparedMutationRecord, providerAuthLeaseID string) (gitRuntimeProofPayload, *gitRemoteExecutionError) {
	req := gitRemoteExecutionRequest{Record: record, ProviderAuthLeaseID: providerAuthLeaseID}
	return s.effectiveGitRemoteMutationExecutor().executePreparedMutation(ctx, req)
}
