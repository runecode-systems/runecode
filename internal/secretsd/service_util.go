package secretsd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func validateBinding(secretRef, consumerID, roleKind, scope string) error {
	if secretRef != "placeholder" && strings.TrimSpace(secretRef) == "" {
		return fmt.Errorf("secret_ref is required")
	}
	if strings.TrimSpace(consumerID) == "" {
		return fmt.Errorf("consumer_id is required")
	}
	if strings.TrimSpace(roleKind) == "" {
		return fmt.Errorf("role_kind is required")
	}
	if strings.TrimSpace(scope) == "" {
		return fmt.Errorf("scope is required")
	}
	return nil
}

func normalizeDeliveryKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

func normalizeOperationSet(operations []string) []string {
	if len(operations) == 0 {
		return nil
	}
	unique := map[string]struct{}{}
	for _, op := range operations {
		normalized := strings.ToLower(strings.TrimSpace(op))
		if normalized == "" {
			continue
		}
		unique[normalized] = struct{}{}
	}
	if len(unique) == 0 {
		return nil
	}
	out := make([]string, 0, len(unique))
	for op := range unique {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}

func validateDigestIdentity(value string, fieldName string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) != 2 {
		return fmt.Errorf("%s invalid: must be digest identity", fieldName)
	}
	if _, err := (trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}).Identity(); err != nil {
		return fmt.Errorf("%s invalid: %w", fieldName, err)
	}
	return nil
}

func validateGitLeaseBinding(binding *GitLeaseBinding) error {
	if binding == nil {
		return nil
	}
	if strings.TrimSpace(binding.RepositoryIdentity) == "" {
		return fmt.Errorf("git_binding.repository_identity is required")
	}
	ops := normalizeOperationSet(binding.AllowedOperations)
	if len(ops) == 0 {
		return fmt.Errorf("git_binding.allowed_operations is required")
	}
	if err := validateDigestIdentity(binding.ActionRequestHash, "git_binding.action_request_hash"); err != nil {
		return err
	}
	if err := validateDigestIdentity(binding.PolicyContextHash, "git_binding.policy_context_hash"); err != nil {
		return err
	}
	return nil
}

func cloneGitLeaseBinding(binding *GitLeaseBinding) *GitLeaseBinding {
	if binding == nil {
		return nil
	}
	return &GitLeaseBinding{
		RepositoryIdentity: strings.TrimSpace(binding.RepositoryIdentity),
		AllowedOperations:  append([]string{}, normalizeOperationSet(binding.AllowedOperations)...),
		ActionRequestHash:  strings.TrimSpace(binding.ActionRequestHash),
		PolicyContextHash:  strings.TrimSpace(binding.PolicyContextHash),
	}
}

func gitLeaseBindingMatches(binding *GitLeaseBinding, use *GitLeaseUseContext) bool {
	if binding == nil {
		return false
	}
	if use == nil {
		return false
	}
	if strings.TrimSpace(binding.RepositoryIdentity) != strings.TrimSpace(use.RepositoryIdentity) {
		return false
	}
	if strings.TrimSpace(binding.ActionRequestHash) != strings.TrimSpace(use.ActionRequestHash) {
		return false
	}
	if strings.TrimSpace(binding.PolicyContextHash) != strings.TrimSpace(use.PolicyContextHash) {
		return false
	}
	operation := strings.ToLower(strings.TrimSpace(use.Operation))
	if operation == "" {
		return false
	}
	for _, allowed := range binding.AllowedOperations {
		if allowed == operation {
			return true
		}
	}
	return false
}

func effectiveTTL(requested int) int {
	if requested <= 0 {
		return defaultTTLSeconds
	}
	if requested > hardCapTTLSeconds {
		return hardCapTTLSeconds
	}
	return requested
}

func randomID(r io.Reader, prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

func randomSecretID(r io.Reader) (string, error) {
	return randomID(r, "secret_")
}

func randomLeaseID(r io.Reader) (string, error) {
	return randomID(r, "lease_")
}

func digestHex(b []byte) string {
	d := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(d[:])
}
