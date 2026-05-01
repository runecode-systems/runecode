package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestExternalAnchorMutationPrepareFailsClosedWithoutManifestOptIn(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-anchor-no-opt-in"
	putTrustedGitGatewayContextForRun(t, s, runID, []any{trustedGitGatewayAllowlistEntry()})

	_, errResp := s.HandleExternalAnchorMutationPrepare(context.Background(), externalAnchorPrepareRequest(runID, "req-anchor-prepare-no-opt-in", "sha256:"+strings.Repeat("a", 64), 0), RequestContext{})
	if errResp == nil {
		t.Fatal("HandleExternalAnchorMutationPrepare expected fail-closed opt-in error")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error.code=%q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
	if got := s.ExternalAnchorPreparedRefsForRun(runID); len(got) != 0 {
		t.Fatalf("prepared refs = %v, want empty", got)
	}
}

func TestExternalAnchorMutationPrepareFailsClosedWhenTargetDescriptorDigestNotAllowlisted(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-anchor-allowlist-mismatch"
	allowlistedDigest := "sha256:" + strings.Repeat("b", 64)
	requestedDigest := "sha256:" + strings.Repeat("c", 64)
	putTrustedExternalAnchorGatewayContextForRun(t, s, runID, allowlistedDigest)

	_, errResp := s.HandleExternalAnchorMutationPrepare(context.Background(), externalAnchorPrepareRequest(runID, "req-anchor-prepare-allowlist-mismatch", requestedDigest, 0), RequestContext{})
	if errResp == nil {
		t.Fatal("HandleExternalAnchorMutationPrepare expected fail-closed allowlist mismatch error")
	}
	if errResp.Error.Code != "broker_limit_policy_rejected" {
		t.Fatalf("error.code=%q, want broker_limit_policy_rejected", errResp.Error.Code)
	}
	if got := s.ExternalAnchorPreparedRefsForRun(runID); len(got) != 0 {
		t.Fatalf("prepared refs = %v, want empty", got)
	}
}

func TestExternalAnchorMutationPrepareUsesCanonicalTargetDescriptorDigestIdentity(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-anchor-canonical-target"
	targetDigest := "sha256:" + strings.Repeat("d", 64)
	putTrustedExternalAnchorGatewayContextForRun(t, s, runID, targetDigest)

	resp := mustPrepareExternalAnchorMutation(t, s, runID, "req-anchor-prepare-canonical-target", targetDigest, 0)
	wantDestinationRef := "sha256/" + strings.TrimPrefix(targetDigest, "sha256:")
	if resp.Prepared.DestinationRef != wantDestinationRef {
		t.Fatalf("prepared.destination_ref=%q, want %q", resp.Prepared.DestinationRef, wantDestinationRef)
	}
	if resp.Prepared.ExecutionPathway != "non_workspace_gateway" {
		t.Fatalf("prepared.execution_pathway=%q, want non_workspace_gateway", resp.Prepared.ExecutionPathway)
	}
	if resp.Prepared.AnchorPosture != "external_configured_not_run" {
		t.Fatalf("prepared.anchor_posture=%q, want external_configured_not_run", resp.Prepared.AnchorPosture)
	}
	if got, _ := resp.TypedRequestHash.Identity(); got == "" {
		t.Fatal("typed_request_hash identity empty")
	}
	if got := s.ExternalAnchorPreparedRefsForRun(runID); len(got) != 1 {
		t.Fatalf("prepared refs count = %d, want 1", len(got))
	}
}

func putTrustedExternalAnchorGatewayContextForRun(t *testing.T, s *Service, runID, targetDescriptorDigest string) {
	t.Helper()
	verifier, privateKey := newSignedContextVerifierFixture(t)
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	allowlistEntry := trustedExternalAnchorGatewayAllowlistEntry(targetDescriptorDigest)
	allowlistDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, trustedPolicyAllowlistPayloadWithEntries(t, []any{allowlistEntry}))
	rolePayload := signedPayloadForTrustedContext(t, map[string]any{"schema_id": "runecode.protocol.v0.RoleManifest", "schema_version": "0.2.0", "principal": signedContextPrincipal("gateway", "git-gateway", runID, ""), "role_family": "gateway", "role_kind": "git-gateway", "approval_profile": "moderate", "capability_opt_ins": []any{"cap_external_anchor"}, "allowlist_refs": []any{digestObject(allowlistDigest)}}, verifier, privateKey)
	runPayload := signedPayloadForTrustedContext(t, map[string]any{"schema_id": "runecode.protocol.v0.CapabilityManifest", "schema_version": "0.2.0", "principal": signedContextPrincipal("gateway", "git-gateway", runID, ""), "manifest_scope": "run", "run_id": runID, "approval_profile": "moderate", "capability_opt_ins": []any{"cap_external_anchor"}, "allowlist_refs": []any{digestObject(allowlistDigest)}}, verifier, privateKey)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRoleManifest, rolePayload)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRunCapability, runPayload)
}

func trustedExternalAnchorGatewayAllowlistEntry(targetDescriptorDigest string) map[string]any {
	digestHex := strings.TrimPrefix(targetDescriptorDigest, "sha256:")
	entry := map[string]any{
		"schema_id":         "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":    "0.1.0",
		"scope_kind":        "gateway_destination",
		"entry_id":          "anchor_target",
		"gateway_role_kind": "git-gateway",
		"destination":       trustedExternalAnchorDestinationDescriptor(digestHex),
		"external_anchor_target_descriptor_digests": []any{digestObject(targetDescriptorDigest)},
		"permitted_operations":                      []any{"external_anchor_submit"},
		"allowed_egress_data_classes":               []any{"audit_events"},
		"redirect_posture":                          "allowlist_only",
		"max_timeout_seconds":                       120,
		"max_response_bytes":                        16777216,
	}
	addTrustedExternalAnchorGitPolicies(entry)
	return entry
}

func trustedExternalAnchorDestinationDescriptor(digestHex string) map[string]any {
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

func addTrustedExternalAnchorGitPolicies(entry map[string]any) {
	entry["git_ref_update_policy"] = trustedExactRefPolicy("refs/heads/main")
	entry["git_tag_update_policy"] = trustedPrefixRefPolicy("refs/tags/release/")
	entry["git_pull_request_base_ref_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}}
	entry["git_pull_request_head_namespace_policy"] = map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/heads/rune/"}}}
}

func trustedExactRefPolicy(ref string) map[string]any {
	return map[string]any{
		"rules":              []any{map[string]any{"rule_kind": "exact", "ref": ref}},
		"allow_force_push":   false,
		"allow_ref_deletion": false,
	}
}

func trustedPrefixRefPolicy(prefix string) map[string]any {
	return map[string]any{
		"rules":              []any{map[string]any{"rule_kind": "prefix_glob", "prefix": prefix}},
		"allow_force_push":   false,
		"allow_ref_deletion": false,
	}
}

func trustedExternalAnchorTypedRequest(targetDescriptorDigest string) map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.ExternalAnchorSubmitRequest",
		"schema_version":           "0.1.0",
		"request_kind":             "external_anchor_submit_v0",
		"target_kind":              "transparency_log",
		"target_descriptor_digest": digestObject(targetDescriptorDigest),
		"seal_digest":              digestObject("sha256:" + strings.Repeat("1", 64)),
		"outbound_payload_digest":  digestObject("sha256:" + strings.Repeat("2", 64)),
	}
}

func trustedExternalAnchorTypedRequestWithExecutionMode(targetDescriptorDigest string, deferredPollCount int) map[string]any {
	req := trustedExternalAnchorTypedRequest(targetDescriptorDigest)
	if deferredPollCount > 0 {
		req["deferred_poll_count"] = deferredPollCount
	}
	return req
}
