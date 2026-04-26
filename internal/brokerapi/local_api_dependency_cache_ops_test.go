package brokerapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
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

func requireDependencyCacheEnsureResult(t *testing.T, label string, resp DependencyCacheEnsureResponse, errResp *ErrorResponse, cacheOutcome string, registryRequests int) {
	t.Helper()
	if errResp != nil {
		t.Fatalf("%s HandleDependencyCacheEnsure error: %+v", label, errResp)
	}
	if resp.CacheOutcome != cacheOutcome {
		t.Fatalf("%s cache_outcome = %q, want %s", label, resp.CacheOutcome, cacheOutcome)
	}
	if resp.RegistryRequestCount != registryRequests {
		t.Fatalf("%s registry_request_count = %d, want %d", label, resp.RegistryRequestCount, registryRequests)
	}
}

func requireDependencyCacheEnsureAuditHit(t *testing.T, events []map[string]interface{}) {
	t.Helper()
	if len(events) < 2 {
		t.Fatalf("dependency_cache_ensure events = %d, want >= 2", len(events))
	}
	last := events[len(events)-1]
	if got, _ := last["gateway_role_kind"].(string); got != "dependency-fetch" {
		t.Fatalf("gateway_role_kind = %q, want dependency-fetch", got)
	}
	if got, _ := last["operation"].(string); got != "fetch_dependency" {
		t.Fatalf("operation = %q, want fetch_dependency", got)
	}
	if got, _ := last["audit_outcome"].(string); got != "succeeded" {
		t.Fatalf("audit_outcome = %q, want succeeded", got)
	}
	if got, _ := last["cache_outcome"].(string); got != "hit_exact" {
		t.Fatalf("audit cache_outcome = %q, want hit_exact", got)
	}
	bindings, ok := last["request_bindings"].([]any)
	if !ok || len(bindings) != 1 {
		t.Fatalf("request_bindings = %#v, want one entry", last["request_bindings"])
	}
	binding, ok := bindings[0].(map[string]any)
	if !ok {
		t.Fatalf("request_bindings[0] type = %T, want map[string]any", bindings[0])
	}
	if bound, _ := binding["request_payload_hash_bound"].(bool); !bound {
		t.Fatalf("request_payload_hash_bound = %v, want true", binding["request_payload_hash_bound"])
	}
}

func TestDependencyCacheEnsureIgnoresLockfileLocatorTopologyHintForUnitIdentity(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	firstReq := dependencyCacheEnsureRequestForTest("req-cache-path-a", "run-deps", "portable-hint")
	secondReq := dependencyCacheEnsureRequestForTest("req-cache-path-b", "run-deps", "portable-hint")
	firstReq.BatchRequest.LockfileLocatorHint = `C:\\Users\\dev\\workspace\\deps.lock`
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

func TestDependencyFetchRegistryDigestMismatchRollsBackPayloadArtifact(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	fetcher := &mismatchDigestFetcher{payload: "payload-with-bad-expected-digest"}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	before := s.List()
	_, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-mismatch", "run-deps", "mismatch"), RequestContext{})
	if errResp == nil {
		t.Fatal("HandleDependencyFetchRegistry expected digest mismatch error")
	}
	after := s.List()
	if len(after) != len(before) {
		t.Fatalf("artifact count after mismatch = %d, want %d", len(after), len(before))
	}
}

func TestDependencyFetchRegistryCoalescesMisses(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{DependencyFetch: DependencyFetchConfig{MaxParallelFetches: 8}})
	fetcher := &gatedCountingFetcher{gate: make(chan struct{}), started: make(chan struct{})}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	request := dependencyFetchRegistryRequestForTest("req-fetch-coalesce", "run-deps", "alpha")
	requestHash := request.RequestHash

	const callers = 6
	responses := make([]DependencyFetchRegistryResponse, callers)
	errs := make([]*ErrorResponse, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			fetcher.entered.Add(1)
			localReq := request
			localReq.RequestID = localReq.RequestID + "-" + string(rune('a'+idx))
			resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), localReq, RequestContext{})
			responses[idx] = resp
			errs[idx] = errResp
		}(i)
	}
	close(start)
	select {
	case <-fetcher.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first coalesced fetch start")
	}
	deadline := time.Now().Add(2 * time.Second)
	for fetcher.entered.Load() < callers {
		if time.Now().After(deadline) {
			t.Fatalf("entered callers = %d, want %d", fetcher.entered.Load(), callers)
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(fetcher.gate)
	wg.Wait()
	for i := range errs {
		if errs[i] != nil {
			t.Fatalf("caller %d error: %+v", i, errs[i])
		}
		if responses[i].RequestHash != requestHash {
			t.Fatalf("caller %d request_hash mismatch", i)
		}
	}
	if got := fetcher.calls.Load(); got != 1 {
		t.Fatalf("fetcher calls = %d, want exact single-flight call", got)
	}
}

func TestDependencyFetchRegistryBoundedParallelism(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{DependencyFetch: DependencyFetchConfig{MaxParallelFetches: 2}})
	fetcher := &concurrencyCountingFetcher{gate: make(chan struct{})}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	requests := []DependencyFetchRegistryRequest{
		dependencyFetchRegistryRequestForTest("req-par-1", "run-deps", "a"),
		dependencyFetchRegistryRequestForTest("req-par-2", "run-deps", "b"),
		dependencyFetchRegistryRequestForTest("req-par-3", "run-deps", "c"),
		dependencyFetchRegistryRequestForTest("req-par-4", "run-deps", "d"),
	}

	var wg sync.WaitGroup
	for i := range requests {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, errResp := s.HandleDependencyFetchRegistry(context.Background(), requests[i], RequestContext{}); errResp != nil {
				t.Errorf("request %d error: %+v", i, errResp)
			}
		}(i)
	}
	close(fetcher.gate)
	wg.Wait()
	if got := fetcher.maxConcurrent.Load(); got > 2 {
		t.Fatalf("max concurrent fetches = %d, want <= 2", got)
	}
}

func TestDependencyFetchRegistryStreamsToCAS(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	payload := strings.Repeat("stream-me-", 8192)
	s.SetDependencyRegistryFetcherForTests(streamingFetcher{payload: payload})

	resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-stream", "run-deps", "stream"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}
	if len(resp.PayloadDigests) != 1 {
		t.Fatalf("payload_digests len = %d, want 1", len(resp.PayloadDigests))
	}
	payloadDigestIdentity, err := resp.PayloadDigests[0].Identity()
	if err != nil {
		t.Fatalf("payload digest identity error: %v", err)
	}
	r, err := s.Get(payloadDigestIdentity)
	if err != nil {
		t.Fatalf("Get payload digest returned error: %v", err)
	}
	b, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("ReadAll payload returned error: %v", readErr)
	}
	if string(b) != payload {
		t.Fatalf("stored payload mismatch")
	}
	if resp.FetchedBytes != int64(len(payload)) {
		t.Fatalf("fetched_bytes = %d, want %d", resp.FetchedBytes, len(payload))
	}
}

func TestDependencyFetchRegistryStreamsWithBoundedReadChunks(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	fetcher := &boundedChunkFetcher{payloadSize: 3 << 20, maxReadBuf: 128 << 10}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-stream-bounded", "run-deps", "stream-bounded"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}
	if resp.FetchedBytes != fetcher.payloadSize {
		t.Fatalf("fetched_bytes = %d, want %d", resp.FetchedBytes, fetcher.payloadSize)
	}
	if got := fetcher.maxSeenBuf.Load(); got > int64(fetcher.maxReadBuf) {
		t.Fatalf("max read buffer = %d, want <= %d", got, fetcher.maxReadBuf)
	}
	if got := fetcher.readCalls.Load(); got <= 1 {
		t.Fatalf("read calls = %d, want chunked streaming (>1)", got)
	}
}

func TestDependencyFetchIdentityPortableAcrossStoreRoots(t *testing.T) {
	left := newBrokerAPIServiceForTests(t, APIConfig{})
	right := newBrokerAPIServiceForTests(t, APIConfig{})
	leftReq := dependencyFetchRegistryRequestForTest("req-portable-left", "run-deps-portability-left", "portable")
	rightReq := dependencyFetchRegistryRequestForTest("req-portable-right", "run-deps-portability-right", "portable")

	leftResp, leftErr := left.HandleDependencyFetchRegistry(context.Background(), leftReq, RequestContext{})
	if leftErr != nil {
		t.Fatalf("left HandleDependencyFetchRegistry error: %+v", leftErr)
	}
	rightResp, rightErr := right.HandleDependencyFetchRegistry(context.Background(), rightReq, RequestContext{})
	if rightErr != nil {
		t.Fatalf("right HandleDependencyFetchRegistry error: %+v", rightErr)
	}
	if leftResp.RequestHash != rightResp.RequestHash {
		t.Fatalf("request_hash mismatch across store roots")
	}
	if leftResp.ResolvedUnitDigest != rightResp.ResolvedUnitDigest {
		t.Fatalf("resolved_unit_digest mismatch across store roots")
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

	handoff, ok, err := s.DependencyCacheHandoffByRequest(artifacts.DependencyCacheHandoffRequest{
		RequestDigest: requestIdentity,
		ConsumerRole:  "workspace",
	})
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

	_, _, err = s.DependencyCacheHandoffByRequest(artifacts.DependencyCacheHandoffRequest{
		RequestDigest: requestIdentity,
		ConsumerRole:  "model_gateway",
	})
	if err != artifacts.ErrFlowDenied {
		t.Fatalf("DependencyCacheHandoffByRequest consumer error = %v, want %v", err, artifacts.ErrFlowDenied)
	}
}

func TestDependencyFetchAuditIncludesPolicyAndAllowlistLinkage(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	runID := "run-deps-policy-audit"
	allowlistDigest := putTrustedDependencyFetchContextForRun(t, s, runID)

	req := dependencyFetchRegistryRequestForTest("req-policy-audit", runID, "alpha")
	requestHash, err := req.RequestHash.Identity()
	if err != nil {
		t.Fatalf("request hash identity error: %v", err)
	}
	decision := policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionAllow,
		PolicyReasonCode:       "allow_manifest_opt_in",
		ManifestHash:           "sha256:" + strings.Repeat("1", 64),
		PolicyInputHashes:      []string{"sha256:" + strings.Repeat("2", 64)},
		ActionRequestHash:      "sha256:" + strings.Repeat("3", 64),
		RelevantArtifactHashes: []string{requestHash},
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details: map[string]any{
			"operation":         "fetch_dependency",
			"gateway_role_kind": "dependency-fetch",
			"destination_kind":  "package_registry",
		},
	}
	if err := s.RecordPolicyDecision(runID, "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	decisionHash := decisionDigestIdentity(decision)

	_, errResp := s.HandleDependencyFetchRegistry(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}

	events := auditEventsByType(t, s, "dependency_registry_fetch")
	if len(events) == 0 {
		t.Fatal("dependency_registry_fetch audit event not found")
	}
	last := events[len(events)-1]
	if got, _ := last["action_request_hash"].(string); got != decision.ActionRequestHash {
		t.Fatalf("action_request_hash = %q, want %q", got, decision.ActionRequestHash)
	}
	if got, _ := last["policy_decision_hash"].(string); got != decisionHash {
		t.Fatalf("policy_decision_hash = %q, want %q", got, decisionHash)
	}
	if got, _ := last["matched_allowlist_ref"].(string); got != allowlistDigest {
		t.Fatalf("matched_allowlist_ref = %q, want %q", got, allowlistDigest)
	}
	if got, _ := last["matched_allowlist_entry_id"].(string); got != "dependency_default" {
		t.Fatalf("matched_allowlist_entry_id = %q, want dependency_default", got)
	}
	if got, _ := last["destination_ref"].(string); got != "registry.npmjs.org/" {
		t.Fatalf("destination_ref = %q, want registry.npmjs.org/", got)
	}
}

func auditEventsByType(t *testing.T, s *Service, eventType string) []map[string]interface{} {
	t.Helper()
	events, err := s.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	out := []map[string]interface{}{}
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		out = append(out, event.Details)
	}
	return out
}

func putTrustedDependencyFetchContextForRun(t *testing.T, s *Service, runID string) string {
	t.Helper()
	verifier, privateKey := newSignedContextVerifierFixture(t)
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	allowlistPayload := trustedPolicyAllowlistPayloadWithEntries(t, []any{trustedDependencyFetchAllowlistEntryForTests()})
	allowlistDigest := putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindPolicyAllowlist, allowlistPayload)
	rolePayload := signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal("gateway", "dependency-fetch", runID, ""),
		"role_family":        "gateway",
		"role_kind":          "dependency-fetch",
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{digestObject(allowlistDigest)},
	}, verifier, privateKey)
	runPayload := signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal("gateway", "dependency-fetch", runID, ""),
		"manifest_scope":     "run",
		"run_id":             runID,
		"approval_profile":   "moderate",
		"capability_opt_ins": []any{"cap_gateway"},
		"allowlist_refs":     []any{digestObject(allowlistDigest)},
	}, verifier, privateKey)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRoleManifest, rolePayload)
	putTrustedPolicyArtifact(t, s, runID, artifacts.TrustedContractImportKindRunCapability, runPayload)
	return allowlistDigest
}

func trustedDependencyFetchAllowlistEntryForTests() map[string]any {
	return map[string]any{
		"schema_id":         "runecode.protocol.v0.GatewayScopeRule",
		"schema_version":    "0.1.0",
		"scope_kind":        "gateway_destination",
		"entry_id":          "dependency_default",
		"gateway_role_kind": "dependency-fetch",
		"destination": map[string]any{
			"schema_id":                "runecode.protocol.v0.DestinationDescriptor",
			"schema_version":           "0.1.0",
			"descriptor_kind":          "package_registry",
			"canonical_host":           "registry.npmjs.org",
			"canonical_path_prefix":    "/",
			"provider_or_namespace":    "npm",
			"tls_required":             true,
			"private_range_blocking":   "enforced",
			"dns_rebinding_protection": "enforced",
		},
		"permitted_operations":        []any{"fetch_dependency"},
		"allowed_egress_data_classes": []any{"dependency_resolved_payload"},
		"redirect_posture":            "allowlist_only",
		"max_timeout_seconds":         120,
		"max_response_bytes":          16777216,
	}
}

func dependencyCacheEnsureRequestForTest(requestID, runID, pkg string) DependencyCacheEnsureRequest {
	dep := dependencyFetchRequestForTest(pkg)
	batch := DependencyFetchBatchRequestObject{
		SchemaID:            "runecode.protocol.v0.DependencyFetchBatchRequest",
		SchemaVersion:       "0.1.0",
		LockfileKind:        "generic_lock",
		LockfileDigest:      digestForDependencyTest(artifacts.DigestBytes([]byte("lock:" + pkg))),
		RequestSetHash:      digestForDependencyTest(artifacts.DigestBytes([]byte("request-set:" + pkg))),
		DependencyRequests:  []DependencyFetchRequestObject{dep},
		BatchRequestID:      "batch-" + pkg,
		LockfileLocatorHint: "deps.lock",
	}
	return DependencyCacheEnsureRequest{SchemaID: "runecode.protocol.v0.DependencyCacheEnsureRequest", SchemaVersion: "0.1.0", RequestID: requestID, RunID: runID, BatchRequest: batch}
}

func dependencyFetchRegistryRequestForTest(requestID, runID, pkg string) DependencyFetchRegistryRequest {
	dep := dependencyFetchRequestForTest(pkg)
	hash, err := canonicalDependencyRequestIdentity(dep)
	if err != nil {
		panic(err)
	}
	requestHash, err := digestFromIdentity(hash)
	if err != nil {
		panic(err)
	}
	return DependencyFetchRegistryRequest{SchemaID: "runecode.protocol.v0.DependencyFetchRegistryRequest", SchemaVersion: "0.1.0", RequestID: requestID, RunID: runID, DependencyRequest: dep, RequestHash: requestHash}
}

func digestForDependencyTest(identity string) trustpolicy.Digest {
	d, err := digestFromIdentity(identity)
	if err != nil {
		panic(err)
	}
	return d
}

func dependencyFetchRequestForTest(pkg string) DependencyFetchRequestObject {
	return DependencyFetchRequestObject{
		SchemaID:      "runecode.protocol.v0.DependencyFetchRequest",
		SchemaVersion: "0.1.0",
		RequestKind:   "package_version_fetch",
		RegistryIdentity: policyengine.DestinationDescriptor{
			SchemaID:               "runecode.protocol.v0.DestinationDescriptor",
			SchemaVersion:          "0.1.0",
			DescriptorKind:         "package_registry",
			CanonicalHost:          "registry.npmjs.org",
			CanonicalPathPrefix:    "/",
			ProviderOrNamespace:    "npm",
			TLSRequired:            true,
			PrivateRangeBlocking:   "enforced",
			DNSRebindingProtection: "enforced",
		},
		Ecosystem:      "npm",
		PackageName:    "pkg-" + pkg,
		PackageVersion: "1.0.0",
	}
}

type gatedCountingFetcher struct {
	gate    chan struct{}
	started chan struct{}
	once    sync.Once
	entered atomic.Int64
	calls   atomic.Int64
}

func (f *gatedCountingFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("auth lease required")
	}
	f.calls.Add(1)
	f.once.Do(func() { close(f.started) })
	<-f.gate
	payload := "coalesced-payload"
	return io.NopCloser(strings.NewReader(payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: artifacts.DigestBytes([]byte(payload))}, nil
}

type boundedChunkFetcher struct {
	payloadSize int64
	maxReadBuf  int
	maxSeenBuf  atomic.Int64
	readCalls   atomic.Int64
}

func (f *boundedChunkFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("auth lease required")
	}
	if f.payloadSize <= 0 {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("payload size must be positive")
	}
	if f.maxReadBuf <= 0 {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("max read buffer must be positive")
	}
	reader := &boundedChunkReader{remaining: f.payloadSize, byteValue: 'z', maxReadBuf: f.maxReadBuf, maxSeenBuf: &f.maxSeenBuf, readCalls: &f.readCalls}
	return io.NopCloser(reader), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream"}, nil
}

type boundedChunkReader struct {
	remaining  int64
	byteValue  byte
	maxReadBuf int
	maxSeenBuf *atomic.Int64
	readCalls  *atomic.Int64
}

func (r *boundedChunkReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	if len(p) > r.maxReadBuf {
		return 0, errors.New("reader observed oversized read buffer")
	}
	seen := int64(len(p))
	for {
		max := r.maxSeenBuf.Load()
		if seen <= max || r.maxSeenBuf.CompareAndSwap(max, seen) {
			break
		}
	}
	r.readCalls.Add(1)
	n := len(p)
	if int64(n) > r.remaining {
		n = int(r.remaining)
	}
	for i := 0; i < n; i++ {
		p[i] = r.byteValue
	}
	r.remaining -= int64(n)
	return n, nil
}

type concurrencyCountingFetcher struct {
	gate          chan struct{}
	current       atomic.Int64
	maxConcurrent atomic.Int64
}

func (f *concurrencyCountingFetcher) Fetch(_ context.Context, req DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("auth lease required")
	}
	<-f.gate
	cur := f.current.Add(1)
	for {
		max := f.maxConcurrent.Load()
		if cur <= max || f.maxConcurrent.CompareAndSwap(max, cur) {
			break
		}
	}
	defer f.current.Add(-1)
	payload := "payload-" + req.PackageName
	return io.NopCloser(strings.NewReader(payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: artifacts.DigestBytes([]byte(payload))}, nil
}

type streamingFetcher struct {
	payload string
}

func (f streamingFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("auth lease required")
	}
	if f.payload == "" {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("payload required")
	}
	return io.NopCloser(strings.NewReader(f.payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: artifacts.DigestBytes([]byte(f.payload))}, nil
}

type mismatchDigestFetcher struct {
	payload string
}

func (f *mismatchDigestFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("auth lease required")
	}
	return io.NopCloser(strings.NewReader(f.payload)), dependencyRegistryFetchMetadata{
		ContentType:           "application/octet-stream",
		ExpectedPayloadDigest: artifacts.DigestBytes([]byte("different-payload")),
	}, nil
}

func TestDependencyFetchRegistryAuthIsBrokerInternalAndRunnerSurfaceClean(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	s.SetDependencyRegistryAuthSourceForTests(fakeDependencyRegistryAuthSource{leaseID: "lease-sensitive", expiresAt: time.Now().Add(5 * time.Minute).UTC()})
	fetcher := &leaseRecordingFetcher{payload: "leased-payload"}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-auth-guard", "run-deps", "auth"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}
	if got := fetcher.lastLeasePosture.Load(); got != string(dependencyRegistryAuthPosturePublicNoAuth) {
		t.Fatalf("lease posture seen by fetcher = %q, want %q", got, dependencyRegistryAuthPosturePublicNoAuth)
	}
	if got := fetcher.lastLeaseID.Load(); got != "lease-sensitive" {
		t.Fatalf("lease id seen by fetcher = %q, want lease-sensitive", got)
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal response returned error: %v", err)
	}
	respText := string(respJSON)
	for _, forbidden := range []string{"lease-sensitive", "registry_auth", "auth_lease", "credential", "token", "secret"} {
		if strings.Contains(respText, forbidden) {
			t.Fatalf("response unexpectedly contains %q: %s", forbidden, respText)
		}
	}

	events := auditEventsByType(t, s, "dependency_registry_fetch")
	if len(events) == 0 {
		t.Fatal("dependency_registry_fetch audit event not found")
	}
	last := events[len(events)-1]
	if got, _ := last["registry_auth_posture"].(string); got != string(dependencyRegistryAuthPosturePublicNoAuth) {
		t.Fatalf("registry_auth_posture = %q, want %q", got, dependencyRegistryAuthPosturePublicNoAuth)
	}
	if _, ok := last["registry_auth_lease_id"]; ok {
		t.Fatal("registry_auth_lease_id unexpectedly present in audit details")
	}
	if _, ok := last["registry_auth_material"]; ok {
		t.Fatal("registry_auth_material unexpectedly present in audit details")
	}
}

func TestDependencyFetchRunnerFacingTypesDoNotExposeAuthMaterial(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(DependencyFetchRequestObject{}),
		reflect.TypeOf(DependencyFetchBatchRequestObject{}),
		reflect.TypeOf(DependencyCacheEnsureRequest{}),
		reflect.TypeOf(DependencyCacheEnsureResponse{}),
		reflect.TypeOf(DependencyFetchRegistryRequest{}),
		reflect.TypeOf(DependencyFetchRegistryResponse{}),
	}
	for _, typ := range types {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name := strings.ToLower(field.Name + " " + field.Tag.Get("json"))
			for _, forbidden := range []string{"credential", "token", "secret", "password", "auth_material", "auth_lease"} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("type %s unexpectedly exposes forbidden auth field %q", typ.Name(), field.Name)
				}
			}
		}
	}
}

type fakeDependencyRegistryAuthSource struct {
	leaseID   string
	expiresAt time.Time
}

func (f fakeDependencyRegistryAuthSource) AcquireLease(_ context.Context, _ DependencyFetchRequestObject) (dependencyRegistryAuthLease, error) {
	return fakeDependencyRegistryAuthLease{posture: dependencyRegistryAuthPosturePublicNoAuth, leaseID: f.leaseID, expiresAt: f.expiresAt}, nil
}

type fakeDependencyRegistryAuthLease struct {
	posture   dependencyRegistryAuthPosture
	leaseID   string
	expiresAt time.Time
}

func (f fakeDependencyRegistryAuthLease) Posture() dependencyRegistryAuthPosture { return f.posture }
func (f fakeDependencyRegistryAuthLease) LeaseID() string                        { return f.leaseID }
func (f fakeDependencyRegistryAuthLease) ExpiresAt() time.Time                   { return f.expiresAt }

type leaseRecordingFetcher struct {
	payload          string
	lastLeasePosture atomic.Value
	lastLeaseID      atomic.Value
}

func (f *leaseRecordingFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("auth lease required")
	}
	f.lastLeasePosture.Store(string(lease.Posture()))
	f.lastLeaseID.Store(lease.LeaseID())
	payload := f.payload
	if payload == "" {
		payload = "lease-recording-payload"
	}
	return io.NopCloser(strings.NewReader(payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: artifacts.DigestBytes([]byte(payload))}, nil
}
