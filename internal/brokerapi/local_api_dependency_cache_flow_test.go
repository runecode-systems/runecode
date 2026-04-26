package brokerapi

import (
	"context"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestDependencyCacheEnsureHitAndMiss(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
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
