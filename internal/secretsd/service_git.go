package secretsd

import (
	"fmt"
	"strings"
)

func validateIssueLeaseRequest(req IssueLeaseRequest) (string, error) {
	if err := validateBinding(req.SecretRef, req.ConsumerID, req.RoleKind, req.Scope); err != nil {
		return "", err
	}
	deliveryKind := normalizeDeliveryKind(req.DeliveryKind)
	if deniedDeliveryKind(deliveryKind) {
		return "", fmt.Errorf("delivery_kind %q is forbidden", deliveryKind)
	}
	return validateGitIssueLeaseRequest(deliveryKind, req.GitBinding)
}

func validateGitIssueLeaseRequest(deliveryKind string, gitBinding *GitLeaseBinding) (string, error) {
	if gitBinding == nil {
		if deliveryKind == deliveryKindGitGateway {
			return "", fmt.Errorf("git lease requires git_binding")
		}
		return deliveryKind, nil
	}
	if deliveryKind == "" {
		deliveryKind = deliveryKindGitGateway
	}
	if deliveryKind != deliveryKindGitGateway {
		return "", fmt.Errorf("git lease delivery_kind must be %q", deliveryKindGitGateway)
	}
	if err := validateGitLeaseBinding(gitBinding); err != nil {
		return "", err
	}
	return deliveryKind, nil
}

func validateGitLeaseRevocationRequest(req RevokeGitLeasesRequest) (string, string, string, error) {
	repositoryIdentity := strings.TrimSpace(req.RepositoryIdentity)
	if repositoryIdentity == "" {
		return "", "", "", fmt.Errorf("repository_identity is required")
	}
	actionRequestHash, err := validatedOptionalDigestIdentity(req.ActionRequestHash, "action_request_hash")
	if err != nil {
		return "", "", "", err
	}
	policyContextHash, err := validatedOptionalDigestIdentity(req.PolicyContextHash, "policy_context_hash")
	if err != nil {
		return "", "", "", err
	}
	return repositoryIdentity, actionRequestHash, policyContextHash, nil
}

func validatedOptionalDigestIdentity(value, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if err := validateDigestIdentity(trimmed, field); err != nil {
		return "", err
	}
	return trimmed, nil
}

func gitLeaseRecordMatchesRevocation(lease leaseRecord, repositoryIdentity, actionRequestHash, policyContextHash string) bool {
	if lease.Status != leaseStatusActive || lease.GitBinding == nil {
		return false
	}
	if strings.TrimSpace(lease.GitBinding.RepositoryIdentity) != repositoryIdentity {
		return false
	}
	return optionalGitRevocationFieldMatches(strings.TrimSpace(lease.GitBinding.ActionRequestHash), actionRequestHash) &&
		optionalGitRevocationFieldMatches(strings.TrimSpace(lease.GitBinding.PolicyContextHash), policyContextHash)
}

func optionalGitRevocationFieldMatches(actual, expected string) bool {
	return expected == "" || actual == expected
}
