package brokerapi

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestPolicyRuntimeGitGatewayFailsClosedOnRemoteDriftAndTreeMismatch(t *testing.T) {
	s, payload, compiled := trustedGitGatewayVerificationCase(t, "run-gateway-git-drift", trustedGitRuntimeProofPayload(t, map[string]any{
		"expected_old_object_id":    strings.Repeat("a", 40),
		"observed_old_object_id":    strings.Repeat("b", 40),
		"observed_result_tree_hash": digestObject("sha256:" + strings.Repeat("4", 64)),
		"drift_detected":            true,
	}))
	_ = s
	reason, details, denied := runtimeGitOutboundVerificationReason(payload)
	if !denied {
		t.Fatal("runtimeGitOutboundVerificationReason denied=false, want true")
	}
	decision := runtimeGatewayDenyDecision(compiled, allowGatewayDecision(compiled), payload, reason, details)
	if decision.DecisionOutcome != policyengine.DecisionDeny {
		t.Fatalf("decision_outcome = %q, want deny", decision.DecisionOutcome)
	}
	if got, _ := decision.Details["reason"].(string); got != "runtime_git_remote_drift_detected" {
		t.Fatalf("reason = %q, want runtime_git_remote_drift_detected", got)
	}
}

func TestPolicyRuntimeGitGatewayFailsClosedWhenTypedRequestMissing(t *testing.T) {
	s, payload, compiled := trustedGitGatewayVerificationCase(t, "run-gateway-git-missing-request", trustedGitRuntimeProofPayload(t, map[string]any{}))
	_ = s
	payload.GitRequest = nil
	reason, details, denied := runtimeGitOutboundVerificationReason(payload)
	if !denied {
		t.Fatal("runtimeGitOutboundVerificationReason denied=false, want true")
	}
	decision := runtimeGatewayDenyDecision(compiled, allowGatewayDecision(compiled), payload, reason, details)
	if got, _ := decision.Details["reason"].(string); got != "runtime_git_request_missing" {
		t.Fatalf("reason = %q, want runtime_git_request_missing", got)
	}
}

func TestPolicyRuntimeGitGatewayEmitsGitEgressAuditProofFields(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-gateway-git-audit"
	putTrustedGitGatewayContextForRun(t, s, runID, []any{trustedGitGatewayAllowlistEntry()})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"git.example.com": {"93.184.216.34"}}}
	payload, compiled, match := prepareTrustedGitGatewayAuditCase(t, s, runID)
	if err := s.gatewayRuntime.emitGatewayAuditEvent(runID, allowGatewayDecision(compiled), payload, match); err != nil {
		t.Fatalf("emitGatewayAuditEvent returned error: %v", err)
	}
	assertGitEgressAuditProofFields(t, requireAuditEventDetails(t, s, "git_egress"))
}

func trustedGitGatewayVerificationCase(t *testing.T, runID string, proof map[string]any) (*Service, gatewayActionPayloadRuntime, *policyengine.CompiledContext) {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedGitGatewayContextForRun(t, s, runID, []any{trustedGitGatewayAllowlistEntry()})
	s.gatewayRuntime.resolver = fakeResolver{hosts: map[string][]string{"git.example.com": {"93.184.216.34"}}}
	action := trustedGitGatewayPushAction(t)
	action.ActionPayload["git_runtime_proof"] = proof
	bindGitRequestHashes(t, &action)
	payload, compiled := trustedGitGatewayPayloadAndCompile(t, s, runID, action)
	_, _, found, reason := findMatchingGatewayAllowlistEntry(compiled, payload)
	if !found {
		t.Fatalf("findMatchingGatewayAllowlistEntry found=%v reason=%q, want match", found, reason)
	}
	return s, payload, compiled
}

func prepareTrustedGitGatewayAuditCase(t *testing.T, s *Service, runID string) (gatewayActionPayloadRuntime, *policyengine.CompiledContext, gatewayAllowlistMatch) {
	t.Helper()
	payload, compiled := trustedGitGatewayPayloadAndCompile(t, s, runID, trustedGitGatewayPullRequestAction(t))
	entry, match, found, reason := findMatchingGatewayAllowlistEntry(compiled, payload)
	if !found {
		t.Fatalf("findMatchingGatewayAllowlistEntry found=%v reason=%q, want match", found, reason)
	}
	if reason, details, denied := s.gatewayRuntime.runtimeEnforcementDenyReason(runID, entry, payload); denied {
		t.Fatalf("runtimeEnforcementDenyReason denied with reason=%q details=%v, want allow", reason, details)
	}
	return payload, compiled, match
}

func trustedGitGatewayPayloadAndCompile(t *testing.T, s *Service, runID string, action policyengine.ActionRequest) (gatewayActionPayloadRuntime, *policyengine.CompiledContext) {
	t.Helper()
	payload, err := decodeGatewayRuntimePayload(action.ActionPayload)
	if err != nil {
		t.Fatalf("decodeGatewayRuntimePayload returned error: %v", err)
	}
	compileInput, err := policyRuntime{service: s}.loadCompileInput(runID)
	if err != nil {
		t.Fatalf("loadCompileInput returned error: %v", err)
	}
	compiled, err := policyengine.Compile(compileInput)
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	return payload, compiled
}

func allowGatewayDecision(compiled *policyengine.CompiledContext) policyengine.PolicyDecision {
	return policyengine.PolicyDecision{SchemaID: gatewayPolicyDecisionSchemaID, SchemaVersion: gatewayPolicyDecisionSchemaVersion, DecisionOutcome: policyengine.DecisionAllow, PolicyReasonCode: "allow_manifest_opt_in", ManifestHash: compiled.ManifestHash, PolicyInputHashes: append([]string{}, compiled.PolicyInputHashes...)}
}

func requireAuditEventDetails(t *testing.T, s *Service, eventType string) map[string]interface{} {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == eventType {
			return events[i].Details
		}
	}
	t.Fatalf("%s audit event not found", eventType)
	return nil
}

func assertGitEgressAuditProofFields(t *testing.T, auditDetails map[string]interface{}) {
	t.Helper()
	if got, _ := auditDetails["matched_allowlist_entry_id"].(string); got != "git_repo_main" {
		t.Fatalf("matched_allowlist_entry_id = %q, want git_repo_main", got)
	}
	if got, _ := auditDetails["provider_kind"].(string); got != "github" {
		t.Fatalf("provider_kind = %q, want github", got)
	}
	if got, _ := auditDetails["pull_request_url"].(string); got == "" {
		t.Fatal("pull_request_url missing from git_egress audit details")
	}
	assertGitAuditDigestField(t, auditDetails, "expected_result_tree_hash")
	assertGitAuditDigestField(t, auditDetails, "observed_result_tree_hash")
}

func assertGitAuditDigestField(t *testing.T, auditDetails map[string]interface{}, field string) {
	t.Helper()
	if got, _ := auditDetails[field].(string); got == "" || !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("%s = %v, want sha256 digest", field, auditDetails[field])
	}
}

func putTrustedGitGatewayContextForRun(t *testing.T, s *Service, runID string, allowlistEntries []any) {
	t.Helper()
	verifier, privateKey := newSignedContextVerifierFixture(t)
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	allowlistDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, trustedPolicyAllowlistPayloadWithEntries(t, allowlistEntries))
	rolePayload := signedPayloadForTrustedContext(t, map[string]any{"schema_id": "runecode.protocol.v0.RoleManifest", "schema_version": "0.2.0", "principal": signedContextPrincipal("gateway", "git-gateway", runID, ""), "role_family": "gateway", "role_kind": "git-gateway", "approval_profile": "moderate", "capability_opt_ins": []any{"cap_gateway"}, "allowlist_refs": []any{digestObject(allowlistDigest)}}, verifier, privateKey)
	runPayload := signedPayloadForTrustedContext(t, map[string]any{"schema_id": "runecode.protocol.v0.CapabilityManifest", "schema_version": "0.2.0", "principal": signedContextPrincipal("gateway", "git-gateway", runID, ""), "manifest_scope": "run", "run_id": runID, "approval_profile": "moderate", "capability_opt_ins": []any{"cap_gateway"}, "allowlist_refs": []any{digestObject(allowlistDigest)}}, verifier, privateKey)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRoleManifest, rolePayload)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRunCapability, runPayload)
}

func trustedGitGatewayAllowlistEntry() map[string]any {
	return map[string]any{"schema_id": "runecode.protocol.v0.GatewayScopeRule", "schema_version": "0.1.0", "scope_kind": "gateway_destination", "entry_id": "git_repo_main", "gateway_role_kind": "git-gateway", "destination": trustedGitGatewayDestination(), "permitted_operations": []any{"git_ref_update", "git_pull_request_create"}, "allowed_egress_data_classes": []any{"diffs"}, "redirect_posture": "allowlist_only", "max_timeout_seconds": 120, "max_response_bytes": 16777216, "git_ref_update_policy": map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}, "allow_force_push": false, "allow_ref_deletion": false}, "git_tag_update_policy": map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/tags/release/"}}, "allow_force_push": false, "allow_ref_deletion": false}, "git_pull_request_base_ref_policy": map[string]any{"rules": []any{map[string]any{"rule_kind": "exact", "ref": "refs/heads/main"}}}, "git_pull_request_head_namespace_policy": map[string]any{"rules": []any{map[string]any{"rule_kind": "prefix_glob", "prefix": "refs/heads/rune/"}}}}
}

func trustedGitGatewayDestination() map[string]any {
	return map[string]any{"schema_id": "runecode.protocol.v0.DestinationDescriptor", "schema_version": "0.1.0", "descriptor_kind": "git_remote", "canonical_host": "git.example.com", "canonical_path_prefix": "/org/repo.git", "provider_or_namespace": "github", "git_repository_identity": "git.example.com/org/repo.git", "tls_required": true, "private_range_blocking": "enforced", "dns_rebinding_protection": "enforced"}
}

func trustedGitGatewayPushAction(t *testing.T) policyengine.ActionRequest {
	t.Helper()
	policyHash, patchDigest, treeDigest := trustedGitDigests(t)
	action := policyengine.NewGatewayEgressAction(policyengine.GatewayEgressActionInput{ActionEnvelope: policyengine.ActionEnvelope{CapabilityID: "cap_gateway", RelevantArtifactHashes: []trustpolicy.Digest{}, Actor: policyengine.ActionActor{ActorKind: "role_instance", RoleFamily: "gateway", RoleKind: "git-gateway"}}, GatewayRoleKind: "git-gateway", DestinationKind: "git_remote", DestinationRef: "git.example.com/org/repo.git", EgressDataClass: "diffs", Operation: "git_ref_update", AuditContext: trustedGitAuditContextInput(policyHash, treeDigest), GitRequest: trustedGitRequestInput(patchDigest, treeDigest), GitRuntimeProof: trustedGitRuntimeProofInput(patchDigest, treeDigest)})
	bindGitRequestHashes(t, &action)
	return action
}

func trustedGitGatewayPullRequestAction(t *testing.T) policyengine.ActionRequest {
	t.Helper()
	action := trustedGitGatewayPushAction(t)
	action.ActionPayload["operation"] = "git_pull_request_create"
	action.ActionPayload["git_request"] = trustedGitRequestPayload(t, "git_pull_request_create")
	action.ActionPayload["git_runtime_proof"] = trustedGitRuntimeProofPayload(t, map[string]any{"provider_kind": "github", "pull_request_number": int64(42), "pull_request_url": "https://github.example/org/repo/pull/42"})
	bindGitRequestHashes(t, &action)
	return action
}

func trustedGitDigests(t *testing.T) (trustpolicy.Digest, trustpolicy.Digest, trustpolicy.Digest) {
	t.Helper()
	return mustDigestIdentityForE2E(t, "sha256:"+strings.Repeat("2", 64)), mustDigestIdentityForE2E(t, "sha256:"+strings.Repeat("5", 64)), mustDigestIdentityForE2E(t, "sha256:"+strings.Repeat("4", 64))
}

func trustedGitAuditContextInput(policyHash, treeDigest trustpolicy.Digest) *policyengine.GatewayAuditContextInput {
	return &policyengine.GatewayAuditContextInput{OutboundBytes: 4096, StartedAt: "2026-04-12T10:00:00Z", CompletedAt: "2026-04-12T10:00:02Z", Outcome: "succeeded", ResponseHash: &treeDigest, LeaseID: "lease-git-1", PolicyDecisionHash: &policyHash}
}

func trustedGitRequestInput(patchDigest, treeDigest trustpolicy.Digest) *policyengine.GitTypedRequestInput {
	return &policyengine.GitTypedRequestInput{RefUpdate: &policyengine.GitRefUpdateRequestInput{RepositoryIdentity: policyengine.DestinationDescriptor{SchemaID: "runecode.protocol.v0.DestinationDescriptor", SchemaVersion: "0.1.0", DescriptorKind: "git_remote", CanonicalHost: "git.example.com", CanonicalPathPrefix: "/org/repo.git", ProviderOrNamespace: "github", GitRepositoryIdentity: "git.example.com/org/repo.git", TLSRequired: true, PrivateRangeBlocking: "enforced", DNSRebindingProtection: "enforced"}, TargetRef: "refs/heads/main", ExpectedOldRefHash: treeDigest, ReferencedPatchArtifactDigests: []trustpolicy.Digest{patchDigest}, CommitIntent: policyengine.GitCommitIntentInput{Message: policyengine.GitCommitMessageInput{Subject: "Update README", Body: "typed request authority"}, Trailers: []policyengine.GitCommitTrailerInput{{Key: "Signed-off-by", Value: "Rune <rune@example.com>"}}, Author: policyengine.GitIdentityInput{DisplayName: "Rune", Email: "rune@example.com"}, Committer: policyengine.GitIdentityInput{DisplayName: "Rune", Email: "rune@example.com"}, Signoff: policyengine.GitIdentityInput{DisplayName: "Rune", Email: "rune@example.com"}}, ExpectedResultTreeHash: treeDigest, AllowForcePush: false, AllowRefDeletion: false, RefPurpose: "branch"}}
}

func trustedGitRuntimeProofInput(patchDigest, treeDigest trustpolicy.Digest) *policyengine.GitRuntimeProofInput {
	oldObject := strings.Repeat("a", 40)
	return &policyengine.GitRuntimeProofInput{TypedRequestHash: treeDigest, PatchArtifactDigests: []trustpolicy.Digest{patchDigest}, ExpectedOldObjectID: oldObject, ObservedOldObjectID: oldObject, ExpectedResultTreeHash: treeDigest, ObservedResultTreeHash: treeDigest, SparseCheckoutApplied: true, DriftDetected: false, DestructiveRefMutation: false, ProviderKind: "github", EvidenceRefs: []string{"artifact:gate-result", "artifact:run-summary"}}
}

func bindGitRequestHashes(t *testing.T, action *policyengine.ActionRequest) {
	t.Helper()
	payload, err := decodeGatewayRuntimePayload(action.ActionPayload)
	if err != nil {
		t.Fatalf("decodeGatewayRuntimePayload returned error: %v", err)
	}
	if payload.GitRequest == nil {
		t.Fatal("git_request missing after payload decode")
	}
	hashIdentity, err := canonicalGitTypedRequestHash(payload.GitRequest)
	if err != nil {
		t.Fatalf("canonicalGitTypedRequestHash returned error: %v", err)
	}
	requestHash := mustDigestIdentityForE2E(t, hashIdentity)
	action.ActionPayload["payload_hash"] = digestObject(hashIdentity)
	action.RelevantArtifactHashes = []trustpolicy.Digest{requestHash}
	if auditCtx, ok := action.ActionPayload["audit_context"].(map[string]any); ok {
		auditCtx["request_hash"] = digestObject(hashIdentity)
	}
	if proof, ok := action.ActionPayload["git_runtime_proof"].(map[string]any); ok {
		proof["typed_request_hash"] = digestObject(hashIdentity)
	}
}

func trustedGitRequestPayload(t *testing.T, operation string) map[string]any {
	t.Helper()
	if operation == "git_pull_request_create" {
		return map[string]any{"schema_id": "runecode.protocol.v0.GitPullRequestCreateRequest", "schema_version": "0.1.0", "request_kind": "git_pull_request_create", "base_repository_identity": map[string]any{"schema_id": "runecode.protocol.v0.DestinationDescriptor", "schema_version": "0.1.0", "descriptor_kind": "git_remote", "canonical_host": "git.example.com", "canonical_path_prefix": "/org/repo.git", "provider_or_namespace": "github", "git_repository_identity": "git.example.com/org/repo.git", "tls_required": true, "private_range_blocking": "enforced", "dns_rebinding_protection": "enforced"}, "base_ref": "refs/heads/main", "head_repository_identity": map[string]any{"schema_id": "runecode.protocol.v0.DestinationDescriptor", "schema_version": "0.1.0", "descriptor_kind": "git_remote", "canonical_host": "git.example.com", "canonical_path_prefix": "/org/repo.git", "provider_or_namespace": "github", "git_repository_identity": "git.example.com/org/repo.git", "tls_required": true, "private_range_blocking": "enforced", "dns_rebinding_protection": "enforced"}, "head_ref": "refs/heads/rune/docs", "title": "Update docs", "body": "typed request", "head_commit_or_tree_hash": digestObject("sha256:" + strings.Repeat("4", 64)), "referenced_patch_artifact_digests": []any{digestObject("sha256:" + strings.Repeat("5", 64))}, "expected_result_tree_hash": digestObject("sha256:" + strings.Repeat("4", 64))}
	}
	return map[string]any{"schema_id": "runecode.protocol.v0.GitRefUpdateRequest", "schema_version": "0.1.0", "request_kind": "git_ref_update", "repository_identity": map[string]any{"schema_id": "runecode.protocol.v0.DestinationDescriptor", "schema_version": "0.1.0", "descriptor_kind": "git_remote", "canonical_host": "git.example.com", "canonical_path_prefix": "/org/repo.git", "provider_or_namespace": "github", "git_repository_identity": "git.example.com/org/repo.git", "tls_required": true, "private_range_blocking": "enforced", "dns_rebinding_protection": "enforced"}, "target_ref": "refs/heads/main", "expected_old_ref_hash": digestObject("sha256:" + strings.Repeat("4", 64)), "referenced_patch_artifact_digests": []any{digestObject("sha256:" + strings.Repeat("5", 64))}, "commit_intent": map[string]any{"schema_id": "runecode.protocol.v0.GitCommitIntent", "schema_version": "0.1.0", "message": map[string]any{"subject": "Update README", "body": "typed request authority"}, "trailers": []any{map[string]any{"key": "Signed-off-by", "value": "Rune <rune@example.com>"}}, "author": map[string]any{"display_name": "Rune", "email": "rune@example.com"}, "committer": map[string]any{"display_name": "Rune", "email": "rune@example.com"}, "signoff": map[string]any{"display_name": "Rune", "email": "rune@example.com"}}, "expected_result_tree_hash": digestObject("sha256:" + strings.Repeat("4", 64)), "allow_force_push": false, "allow_ref_deletion": false}
}

func trustedGitRuntimeProofPayload(t *testing.T, overrides map[string]any) map[string]any {
	t.Helper()
	base := map[string]any{"schema_id": "runecode.protocol.v0.GitRuntimeProof", "schema_version": "0.1.0", "typed_request_hash": digestObject("sha256:" + strings.Repeat("3", 64)), "patch_artifact_digests": []any{digestObject("sha256:" + strings.Repeat("5", 64))}, "expected_old_object_id": strings.Repeat("a", 40), "observed_old_object_id": strings.Repeat("a", 40), "expected_result_tree_hash": digestObject("sha256:" + strings.Repeat("4", 64)), "observed_result_tree_hash": digestObject("sha256:" + strings.Repeat("4", 64)), "sparse_checkout_applied": true, "drift_detected": false, "destructive_ref_mutation": false, "provider_kind": "github", "evidence_refs": []any{"artifact:gate-result", "artifact:run-summary"}}
	for k, v := range overrides {
		base[k] = v
	}
	return base
}
