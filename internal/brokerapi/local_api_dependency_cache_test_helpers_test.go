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
	return io.NopCloser(strings.NewReader(f.payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: artifacts.DigestBytes([]byte("different-payload"))}, nil
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

func requireNoAuthMaterialInResponse(t *testing.T, resp DependencyFetchRegistryResponse) {
	t.Helper()
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
}

func requireNoAuthMaterialInAudit(t *testing.T, events []map[string]interface{}) {
	t.Helper()
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

func requireNoForbiddenAuthFields(t *testing.T, types []reflect.Type) {
	t.Helper()
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

func buildDependencyFetchPolicyDecision(requestHash string) policyengine.PolicyDecision {
	return policyengine.PolicyDecision{
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
}

func requireDependencyFetchPolicyAudit(t *testing.T, events []map[string]interface{}, decision policyengine.PolicyDecision, decisionHash, allowlistDigest string) {
	t.Helper()
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
