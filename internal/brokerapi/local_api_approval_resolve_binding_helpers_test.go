package brokerapi

import (
	"strings"
	"testing"
)

func TestValidateStoredApprovalDigestMatchRejectsMissingStoredBinding(t *testing.T) {
	payload := map[string]any{
		"manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("a", 64)},
	}
	err := validateStoredApprovalDigestMatch("manifest_hash", "", payload)
	if err == nil {
		t.Fatal("validateStoredApprovalDigestMatch error = nil, want missing stored binding failure")
	}
	if !strings.Contains(err.Error(), "stored pending approval binding is missing") {
		t.Fatalf("error = %q, want missing stored binding message", err.Error())
	}
}

func TestValidateStoredApprovalDigestMatchAcceptsExactBinding(t *testing.T) {
	expected := "sha256:" + strings.Repeat("b", 64)
	payload := map[string]any{
		"manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(expected, "sha256:")},
	}
	if err := validateStoredApprovalDigestMatch("manifest_hash", expected, payload); err != nil {
		t.Fatalf("validateStoredApprovalDigestMatch returned error: %v", err)
	}
}
