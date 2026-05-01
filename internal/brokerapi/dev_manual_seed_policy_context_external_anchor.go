package brokerapi

import (
	"crypto/ed25519"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) seedDevManualPolicyAllowlist(runID string) (string, error) {
	externalAnchorTargetDigest, err := devManualExternalAnchorTargetDescriptorDigest()
	if err != nil {
		return "", err
	}
	allowlistPayload, err := devManualPolicyAllowlistPayload(externalAnchorTargetDigest)
	if err != nil {
		return "", err
	}
	return s.recordTrustedPolicyContextArtifact(runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload)
}

func (s *Service) seedDevManualExternalAnchorGatewayContext(runID, allowlistDigest string, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) error {
	if err := s.recordDevManualSignedTrustedContext(runID, artifacts.TrustedContractImportKindRoleManifest, devManualExternalAnchorGatewayRoleManifestPayload(runID, allowlistDigest), verifier, privateKey); err != nil {
		return err
	}
	return s.recordDevManualSignedTrustedContext(runID, artifacts.TrustedContractImportKindRunCapability, devManualExternalAnchorGatewayCapabilityManifestPayload(runID, allowlistDigest), verifier, privateKey)
}

func devManualPolicyAllowlistPayload(externalAnchorTargetDescriptorDigest string) ([]byte, error) {
	if strings.TrimSpace(externalAnchorTargetDescriptorDigest) == "" {
		return nil, fmt.Errorf("external anchor target descriptor digest is required")
	}
	return mustJSONBytesForDevSeed(map[string]any{
		"schema_id":       "runecode.protocol.v0.PolicyAllowlist",
		"schema_version":  "0.1.0",
		"allowlist_kind":  "gateway_scope_rule",
		"entry_schema_id": "runecode.protocol.v0.GatewayScopeRule",
		"entries":         []any{devManualModelGatewayAllowlistEntry(), devManualExternalAnchorGatewayAllowlistEntry(externalAnchorTargetDescriptorDigest)},
	}), nil
}

func devManualModelGatewayAllowlistEntry() map[string]any {
	return map[string]any{
		"schema_id":                   "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":              "0.1.0",
		"scope_kind":                  "gateway_destination",
		"entry_id":                    "model_default",
		"gateway_role_kind":           "model-gateway",
		"destination":                 map[string]any{"schema_id": "runecode.protocol.v0.DestinationDescriptor", "schema_version": "0.1.0", "descriptor_kind": "model_endpoint", "canonical_host": "model.example.com", "tls_required": true, "private_range_blocking": "enforced", "dns_rebinding_protection": "enforced"},
		"permitted_operations":        []any{"invoke_model"},
		"allowed_egress_data_classes": []any{"spec_text"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
}

func devManualExternalAnchorGatewayAllowlistEntry(targetDescriptorDigest string) map[string]any {
	digestHex := strings.TrimPrefix(strings.TrimSpace(targetDescriptorDigest), "sha256:")
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":    "0.1.0",
		"scope_kind":        "gateway_destination",
		"entry_id":          "external_anchor_default",
		"gateway_role_kind": "git-gateway",
		"destination":       devManualExternalAnchorDestinationDescriptor(digestHex),
		"external_anchor_target_descriptor_digests": []any{digestObjectForDevSeed(targetDescriptorDigest)},
		"permitted_operations":                      []any{"external_anchor_submit"},
		"allowed_egress_data_classes":               []any{"audit_events"},
		"redirect_posture":                          "allowlist_only",
		"max_timeout_seconds":                       120,
		"max_response_bytes":                        16777216,
		"git_ref_update_policy":                     devManualExactRefPolicy("refs/heads/main"),
		"git_tag_update_policy":                     devManualPrefixRefPolicy("refs/tags/release/"),
		"git_pull_request_base_ref_policy":          map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}},
		"git_pull_request_head_namespace_policy":    map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/heads/rune/"}}},
	}
}

func devManualExternalAnchorDestinationDescriptor(digestHex string) map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
		"schema_version":           "0.1.0",
		"descriptor_kind":          "git_remote",
		"canonical_host":           "sha256",
		"canonical_path_prefix":    "/" + digestHex,
		"provider_or_namespace":    "external-anchor",
		"git_repository_identity":  "sha256/" + digestHex,
		"tls_required":             true,
		"private_range_blocking":   "enforced",
		"dns_rebinding_protection": "enforced",
	}
}

func devManualExactRefPolicy(ref string) map[string]any {
	return map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": ref}}, "allow_force_push": false, "allow_ref_deletion": false}
}

func devManualPrefixRefPolicy(prefix string) map[string]any {
	return map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": prefix}}, "allow_force_push": false, "allow_ref_deletion": false}
}

func devManualExternalAnchorGatewayRoleManifestPayload(runID, allowlistDigest string) map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          devManualSignedContextPrincipal(runID, "gateway", "git-gateway"),
		"role_family":        "gateway",
		"role_kind":          "git-gateway",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_external_anchor"},
		"allowlist_refs":     []any{digestObjectForDevSeed(allowlistDigest)},
	}
}

func devManualExternalAnchorGatewayCapabilityManifestPayload(runID, allowlistDigest string) map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          devManualSignedContextPrincipal(runID, "gateway", "git-gateway"),
		"manifest_scope":     "run",
		"run_id":             runID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_external_anchor"},
		"allowlist_refs":     []any{digestObjectForDevSeed(allowlistDigest)},
	}
}

func devManualExternalAnchorTargetDescriptorDigest() (string, error) {
	descriptor := map[string]any{
		"descriptor_schema_id":   "runecode.protocol.audit.anchor_target.transparency_log.v0",
		"log_id":                 "manual-seed-transparency-log",
		"log_public_key_digest":  digestObjectForDevSeed("sha256:" + strings.Repeat("d", 64)),
		"entry_encoding_profile": "jcs_v1",
	}
	return externalAnchorCanonicalDescriptorDigestIdentity(descriptor)
}
