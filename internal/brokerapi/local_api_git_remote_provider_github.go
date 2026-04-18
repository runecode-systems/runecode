package brokerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (n *nativeGitRemoteMutationExecutor) createProviderPullRequest(ctx context.Context, record artifacts.GitRemotePreparedMutationRecord, repo gitExecutionRepository, request gitPullRequestCreateExecuteRequest, providerToken string) (gitPullRequestProviderResult, error) {
	adapter := n.providerAdapterFor(strings.TrimSpace(record.Provider))
	headOwner := repositoryOwner(request.HeadRepositoryIdentity.GitRepositoryIdentity)
	return adapter.CreatePullRequest(ctx, gitPullRequestProviderRequest{
		Host:          repo.apiHost,
		Repository:    repositoryPath(request.BaseRepositoryIdentity.GitRepositoryIdentity),
		BaseRef:       normalizeRefNameForProvider(request.BaseRef),
		HeadRef:       normalizeRefNameForProvider(request.HeadRef),
		HeadOwner:     headOwner,
		Title:         request.Title,
		Body:          request.Body,
		Token:         providerToken,
		ExpectedState: request.ExpectedResultTreeHash,
	})
}

func (n *nativeGitRemoteMutationExecutor) providerAdapterFor(provider string) gitPullRequestProviderAdapter {
	switch normalizeGitProvider(provider) {
	case "github":
		return &gitHubPullRequestAdapter{httpClient: n.httpClient}
	default:
		return &gitHubPullRequestAdapter{httpClient: n.httpClient}
	}
}

func normalizeRefNameForProvider(ref string) string {
	trimmed := strings.TrimSpace(ref)
	if strings.HasPrefix(trimmed, "refs/heads/") {
		return strings.TrimPrefix(trimmed, "refs/heads/")
	}
	return trimmed
}

func repositoryOwner(repoIdentity string) string {
	path := repositoryPath(repoIdentity)
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

func repositoryPath(repoIdentity string) string {
	trimmed := strings.TrimSpace(repoIdentity)
	if strings.HasPrefix(trimmed, "https://") {
		u, err := url.Parse(trimmed)
		if err == nil {
			trimmed = strings.TrimPrefix(strings.TrimSpace(u.Path), "/")
		}
	}
	if idx := strings.Index(trimmed, "/"); idx >= 0 && strings.Contains(trimmed[:idx], ".") {
		trimmed = strings.TrimPrefix(trimmed[idx+1:], "/")
	}
	trimmed = strings.TrimSuffix(trimmed, ".git")
	return strings.TrimPrefix(trimmed, "/")
}

func (g *gitHubPullRequestAdapter) CreatePullRequest(ctx context.Context, req gitPullRequestProviderRequest) (gitPullRequestProviderResult, error) {
	if strings.TrimSpace(req.Token) == "" {
		return gitPullRequestProviderResult{}, fmt.Errorf("provider token is required")
	}
	repo := repositoryPath(req.Repository)
	if repo == "" || !strings.Contains(repo, "/") {
		return gitPullRequestProviderResult{}, fmt.Errorf("repository path invalid")
	}
	bodyBytes, err := json.Marshal(gitHubPullRequestPayload(req))
	if err != nil {
		return gitPullRequestProviderResult{}, err
	}
	endpoint := strings.TrimSuffix(g.apiBase(strings.TrimSpace(req.Host)), "/") + "/repos/" + repo + "/pulls"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return gitPullRequestProviderResult{}, err
	}
	applyGitHubPullRequestHeaders(httpReq, req.Token)
	resp, err := g.httpClientOrDefault().Do(httpReq)
	if err != nil {
		return gitPullRequestProviderResult{}, err
	}
	defer resp.Body.Close()
	return decodeGitHubPullRequestResponse(resp)
}

func gitHubPullRequestPayload(req gitPullRequestProviderRequest) map[string]any {
	head := normalizeRefNameForProvider(req.HeadRef)
	if owner := strings.TrimSpace(req.HeadOwner); owner != "" {
		head = owner + ":" + head
	}
	return map[string]any{
		"title": strings.TrimSpace(req.Title),
		"body":  req.Body,
		"base":  normalizeRefNameForProvider(req.BaseRef),
		"head":  head,
	}
}

func applyGitHubPullRequestHeaders(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (g *gitHubPullRequestAdapter) httpClientOrDefault() *http.Client {
	if g.httpClient != nil {
		return g.httpClient
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func decodeGitHubPullRequestResponse(resp *http.Response) (gitPullRequestProviderResult, error) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return gitPullRequestProviderResult{}, fmt.Errorf("github create pull request failed: status=%d", resp.StatusCode)
	}
	decoded := struct {
		Number int64  `json:"number"`
		URL    string `json:"html_url"`
	}{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return gitPullRequestProviderResult{}, err
	}
	if decoded.Number < 1 || strings.TrimSpace(decoded.URL) == "" {
		return gitPullRequestProviderResult{}, fmt.Errorf("github create pull request response missing number/url")
	}
	return gitPullRequestProviderResult{Number: decoded.Number, URL: decoded.URL}, nil
}

func (g *gitHubPullRequestAdapter) apiBase(host string) string {
	if strings.TrimSpace(g.apiBaseOverride) != "" {
		return strings.TrimSpace(g.apiBaseOverride)
	}
	trimmedHost := strings.TrimSpace(host)
	if trimmedHost == "" || trimmedHost == "github.com" {
		return "https://api.github.com"
	}
	if strings.Contains(trimmedHost, ":") {
		return "https://" + trimmedHost + "/api/v3"
	}
	if strings.HasSuffix(trimmedHost, ".github.com") {
		return "https://api." + trimmedHost
	}
	return "https://" + trimmedHost + "/api/v3"
}

func (s *Service) appendGitRemoteExecutionAudit(record artifacts.GitRemotePreparedMutationRecord, proof gitRuntimeProofPayload, outcome string) error {
	if s == nil || s.gatewayRuntime == nil {
		return nil
	}
	typedHash, policyDigest, err := gitRemoteExecutionAuditDigests(record)
	if err != nil {
		return err
	}
	payload := gitRemoteExecutionAuditPayload(record, proof, outcome, typedHash, policyDigest, s.now().UTC())
	decision := policyengine.PolicyDecision{
		SchemaID:          gatewayPolicyDecisionSchemaID,
		SchemaVersion:     gatewayPolicyDecisionSchemaVersion,
		DecisionOutcome:   policyengine.DecisionAllow,
		PolicyReasonCode:  "allow_by_policy",
		ActionRequestHash: record.ActionRequestHash,
		PolicyInputHashes: []string{record.PolicyDecisionHash},
	}
	return s.gatewayRuntime.emitGatewayAuditEvent(record.RunID, decision, payload, gatewayAllowlistMatch{})
}

func gitRemoteExecutionAuditDigests(record artifacts.GitRemotePreparedMutationRecord) (trustpolicy.Digest, trustpolicy.Digest, error) {
	typedHash, err := digestFromIdentity(record.TypedRequestHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	policyDigest, err := digestFromIdentity(record.PolicyDecisionHash)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	return typedHash, policyDigest, nil
}

func gitRemoteExecutionAuditPayload(record artifacts.GitRemotePreparedMutationRecord, proof gitRuntimeProofPayload, outcome string, typedHash, policyDigest trustpolicy.Digest, now time.Time) gatewayActionPayloadRuntime {
	started := now.Add(-1 * time.Second)
	return gatewayActionPayloadRuntime{
		GatewayRoleKind: "git-gateway",
		DestinationKind: "git_remote",
		DestinationRef:  record.DestinationRef,
		EgressDataClass: "diffs",
		Operation:       record.RequestKind,
		PayloadHash:     &typedHash,
		GitRequest:      cloneStringAnyMap(record.TypedRequest),
		GitRuntimeProof: &proof,
		AuditContext: &gatewayAuditContextPayload{
			OutboundBytes:      1,
			StartedAt:          started.Format(time.RFC3339),
			CompletedAt:        now.Format(time.RFC3339),
			Outcome:            firstNonEmpty(strings.TrimSpace(outcome), "failed"),
			RequestHash:        &typedHash,
			ResponseHash:       &proof.ObservedResultTreeHash,
			PolicyDecisionHash: &policyDigest,
		},
	}
}

func decodePortFromHost(host string) (string, *int) {
	parts := strings.Split(host, ":")
	if len(parts) != 2 {
		return host, nil
	}
	p, err := strconv.Atoi(parts[1])
	if err != nil {
		return parts[0], nil
	}
	return parts[0], &p
}
