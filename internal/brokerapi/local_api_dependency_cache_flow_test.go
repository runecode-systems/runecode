package brokerapi

import (
	"context"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestDependencyCacheEnsureHitAndMiss(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedDependencyFetchContextForRun(t, s, "run-deps")
	req := dependencyCacheEnsureRequestForTest("req-cache-hit-miss", "run-deps", "alpha")

	first, errResp := s.HandleDependencyCacheEnsure(context.Background(), req, RequestContext{})
	requireDependencyCacheEnsureResult(t, "first", first, errResp, "miss_filled", 1)

	second, errResp := s.HandleDependencyCacheEnsure(context.Background(), req, RequestContext{})
	requireDependencyCacheEnsureResult(t, "second", second, errResp, "hit_exact", 0)
	if second.BatchRequestHash != first.BatchRequestHash {
		t.Fatalf("batch_request_hash mismatch between calls")
	}

	events := auditEventsByType(t, s, "dependency_cache_ensure")
	requireDependencyCacheEnsureAuditHit(t, events)
}

func TestDependencyCacheEnsureIgnoresLockfileLocatorTopologyHintForUnitIdentity(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedDependencyFetchContextForRun(t, s, "run-deps")
	firstReq := dependencyCacheEnsureRequestForTest("req-cache-path-a", "run-deps", "portable-hint")
	secondReq := dependencyCacheEnsureRequestForTest("req-cache-path-b", "run-deps", "portable-hint")
	firstReq.BatchRequest.LockfileLocatorHint = `C:\Users\dev\workspace\deps.lock`
	secondReq.BatchRequest.LockfileLocatorHint = "/home/dev/workspace/deps.lock"
	secondReq.BatchRequest.BatchRequestID = "different-batch-id"

	first, firstErr := s.HandleDependencyCacheEnsure(context.Background(), firstReq, RequestContext{})
	if firstErr != nil {
		t.Fatalf("first HandleDependencyCacheEnsure error: %+v", firstErr)
	}
	second, secondErr := s.HandleDependencyCacheEnsure(context.Background(), secondReq, RequestContext{})
	if secondErr != nil {
		t.Fatalf("second HandleDependencyCacheEnsure error: %+v", secondErr)
	}
	if len(first.ResolvedUnitDigests) != 1 || len(second.ResolvedUnitDigests) != 1 {
		t.Fatalf("resolved unit digest counts = (%d,%d), want (1,1)", len(first.ResolvedUnitDigests), len(second.ResolvedUnitDigests))
	}
	if first.ResolvedUnitDigests[0] != second.ResolvedUnitDigests[0] {
		t.Fatalf("resolved_unit_digest mismatch for different lockfile locator hints")
	}
	if second.RegistryRequestCount != 0 {
		t.Fatalf("second registry_request_count = %d, want 0 with request-level cache reuse", second.RegistryRequestCount)
	}
	if first.BatchRequestHash != second.BatchRequestHash {
		t.Fatalf("batch_request_hash mismatch for non-authoritative metadata")
	}
}

func TestDependencyCacheHandoffUsesInternalArtifactFlow(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedDependencyFetchContextForRun(t, s, "run-deps")
	ensureReq := dependencyCacheEnsureRequestForTest("req-handoff", "run-deps", "handoff")
	ensureResp, errResp := s.HandleDependencyCacheEnsure(context.Background(), ensureReq, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyCacheEnsure error: %+v", errResp)
	}
	if len(ensureResp.ResolvedUnitDigests) != 1 {
		t.Fatalf("resolved_unit_digests len = %d, want 1", len(ensureResp.ResolvedUnitDigests))
	}
	requestIdentity, err := canonicalDependencyRequestIdentity(ensureReq.BatchRequest.DependencyRequests[0])
	if err != nil {
		t.Fatalf("canonicalDigestIdentity returned error: %v", err)
	}

	handoff, ok, err := s.DependencyCacheHandoffByRequest(artifacts.DependencyCacheHandoffRequest{RequestDigest: requestIdentity, ConsumerRole: "workspace"})
	if err != nil {
		t.Fatalf("DependencyCacheHandoffByRequest returned error: %v", err)
	}
	if !ok {
		t.Fatal("DependencyCacheHandoffByRequest ok=false, want true")
	}
	if handoff.HandoffMode != "broker_internal_artifact_handoff" {
		t.Fatalf("handoff_mode = %q, want broker_internal_artifact_handoff", handoff.HandoffMode)
	}
	if handoff.MaterializationMode != "derived_read_only" {
		t.Fatalf("materialization_mode = %q, want derived_read_only", handoff.MaterializationMode)
	}

	_, _, err = s.DependencyCacheHandoffByRequest(artifacts.DependencyCacheHandoffRequest{RequestDigest: requestIdentity, ConsumerRole: "model_gateway"})
	if err != artifacts.ErrFlowDenied {
		t.Fatalf("DependencyCacheHandoffByRequest consumer error = %v, want %v", err, artifacts.ErrFlowDenied)
	}
}

func TestDependencyCacheHandoffOperationReturnsBrokerAuthMetadata(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedDependencyFetchContextForRun(t, s, "run-deps")
	seedDependencyCacheForHandoff(t, s, "req-handoff-op", "run-deps", "handoff-op")
	resp := mustHandleDependencyCacheHandoff(t, s, dependencyCacheHandoffRequestForTest("req-handoff-op-call", "handoff-op", "workspace"))
	requireDependencyCacheHandoffMetadata(t, resp)
	requireDependencyCacheHandoffAudit(t, s)
}

func TestDependencyCacheHandoffOperationNormalizesWorkspaceConsumerRoles(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedDependencyFetchContextForRun(t, s, "run-deps")
	seedDependencyCacheForHandoff(t, s, "req-handoff-workspace-test", "run-deps", "handoff-workspace-test")

	resp := mustHandleDependencyCacheHandoff(t, s, dependencyCacheHandoffRequestForTest("req-handoff-workspace-test-call", "handoff-workspace-test", "workspace-test"))
	requireDependencyCacheHandoffMetadata(t, resp)

	events := auditEventsByType(t, s, "dependency_cache_handoff")
	last := events[len(events)-1]
	if got, _ := last["consumer_role"].(string); got != "workspace" {
		t.Fatalf("audit consumer_role = %q, want workspace", got)
	}
	if got, _ := last["requested_consumer_role"].(string); got != "workspace-test" {
		t.Fatalf("audit requested_consumer_role = %q, want workspace-test", got)
	}
}

func TestDependencyCacheHandoffOperationNotFoundAndValidationDenied(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	putTrustedDependencyFetchContextForRun(t, s, "run-deps")
	ensureReq := dependencyCacheEnsureRequestForTest("req-handoff-policy", "run-deps", "handoff-policy")
	_, ensureErr := s.HandleDependencyCacheEnsure(context.Background(), ensureReq, RequestContext{})
	if ensureErr != nil {
		t.Fatalf("HandleDependencyCacheEnsure error: %+v", ensureErr)
	}

	notFoundReq := dependencyCacheHandoffRequestForTest("req-handoff-not-found", "missing", "workspace")
	notFoundResp, notFoundErr := s.HandleDependencyCacheHandoff(context.Background(), notFoundReq, RequestContext{})
	if notFoundErr != nil {
		t.Fatalf("HandleDependencyCacheHandoff(not_found) error: %+v", notFoundErr)
	}
	if notFoundResp.Found {
		t.Fatal("not-found response found=true, want false")
	}
	if notFoundResp.Handoff != nil {
		t.Fatal("not-found response handoff present, want nil")
	}

	denyReq := dependencyCacheHandoffRequestForTest("req-handoff-denied", "handoff-policy", "model_gateway")
	_, denyErr := s.HandleDependencyCacheHandoff(context.Background(), denyReq, RequestContext{})
	if denyErr == nil {
		t.Fatal("HandleDependencyCacheHandoff(validation denied) succeeded, want error")
	}
	if denyErr.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", denyErr.Error.Code)
	}
}

func seedDependencyCacheForHandoff(t *testing.T, s *Service, requestID, runID, pkg string) {
	t.Helper()
	_, errResp := s.HandleDependencyCacheEnsure(context.Background(), dependencyCacheEnsureRequestForTest(requestID, runID, pkg), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyCacheEnsure error: %+v", errResp)
	}
}

func mustHandleDependencyCacheHandoff(t *testing.T, s *Service, req DependencyCacheHandoffRequest) DependencyCacheHandoffResponse {
	t.Helper()
	resp, errResp := s.HandleDependencyCacheHandoff(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyCacheHandoff error: %+v", errResp)
	}
	return resp
}

func requireDependencyCacheHandoffMetadata(t *testing.T, resp DependencyCacheHandoffResponse) {
	t.Helper()
	if !resp.Found {
		t.Fatal("found=false, want true")
	}
	if resp.Handoff == nil {
		t.Fatal("handoff=nil, want metadata")
	}
	if resp.Handoff.HandoffMode != "broker_internal_artifact_handoff" {
		t.Fatalf("handoff_mode = %q, want broker_internal_artifact_handoff", resp.Handoff.HandoffMode)
	}
	if resp.Handoff.MaterializationMode != "derived_read_only" {
		t.Fatalf("materialization_mode = %q, want derived_read_only", resp.Handoff.MaterializationMode)
	}
	if len(resp.Handoff.PayloadDigests) != 1 {
		t.Fatalf("payload_digests len = %d, want 1", len(resp.Handoff.PayloadDigests))
	}
	requireDigestIdentity(t, "request_digest", resp.Handoff.RequestDigest.Identity)
	requireDigestIdentity(t, "resolved_unit_digest", resp.Handoff.ResolvedUnitDigest.Identity)
	requireDigestIdentity(t, "manifest_digest", resp.Handoff.ManifestDigest.Identity)
}

func requireDigestIdentity(t *testing.T, field string, identity func() (string, error)) {
	t.Helper()
	if _, err := identity(); err != nil {
		t.Fatalf("%s identity invalid: %v", field, err)
	}
}

func requireDependencyCacheHandoffAudit(t *testing.T, s *Service) {
	t.Helper()
	if got := auditEventsByType(t, s, "dependency_cache_handoff"); len(got) == 0 {
		t.Fatal("dependency_cache_handoff audit event missing")
	}
}
