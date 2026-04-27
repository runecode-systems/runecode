package brokerapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestResolveDependencyRequestFromCachePreservesAllPayloadArtifacts(t *testing.T) {
	s, req, expectedDigests := seedMultiPayloadCacheHitFixture(t)
	requestHash, err := canonicalDependencyRequestIdentity(req)
	if err != nil {
		t.Fatalf("canonicalDependencyRequestIdentity returned error: %v", err)
	}
	resolution, hit, err := s.dependencyFetchService.resolveDependencyRequestFromCache(req, requestHash, time.Now().UTC())
	if err != nil {
		t.Fatalf("resolveDependencyRequestFromCache returned error: %v", err)
	}
	if !hit {
		t.Fatal("resolveDependencyRequestFromCache hit = false, want true")
	}
	assertManifestPayloadArtifactDigests(t, resolution.resolvedUnitManifest, expectedDigests)
}

func seedMultiPayloadCacheHitFixture(t *testing.T) (*Service, DependencyFetchRequestObject, []string) {
	t.Helper()
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	req := dependencyFetchRequestForTest("multi-payload-hit")
	requestHash, err := canonicalDependencyRequestIdentity(req)
	if err != nil {
		t.Fatalf("canonicalDependencyRequestIdentity returned error: %v", err)
	}
	payloadA := putDependencyArtifactForManifestTest(t, s, artifacts.DataClassDependencyPayloadUnit, []byte("payload-a"), "application/octet-stream")
	payloadB := putDependencyArtifactForManifestTest(t, s, artifacts.DataClassDependencyPayloadUnit, []byte("payload-b"), "application/octet-stream")
	manifest := putDependencyArtifactForManifestTest(t, s, artifacts.DataClassDependencyResolvedUnit, []byte(`{"schema_id":"runecode.protocol.v0.DependencyResolvedUnitManifest","schema_version":"0.1.0"}`), "application/json")
	unit := artifacts.DependencyCacheResolvedUnitRecord{ResolvedUnitDigest: artifacts.DigestBytes([]byte("resolved-unit:multi-payload-hit")), RequestDigest: requestHash, ManifestDigest: manifest.Digest, PayloadDigest: []string{payloadA.Digest, payloadB.Digest}, IntegrityState: "verified", MaterializationState: "derived_read_only", CreatedAt: time.Now().UTC()}
	if err := s.RecordDependencyCacheResolvedUnit(unit); err != nil {
		t.Fatalf("RecordDependencyCacheResolvedUnit returned error: %v", err)
	}
	return s, req, []string{payloadA.Digest, payloadB.Digest}
}

func assertManifestPayloadArtifactDigests(t *testing.T, manifest map[string]any, expected []string) {
	t.Helper()
	payloadArtifacts, ok := manifest["payload_artifacts"].([]any)
	if !ok {
		t.Fatalf("payload_artifacts type = %T, want []any", manifest["payload_artifacts"])
	}
	if len(payloadArtifacts) != len(expected) {
		t.Fatalf("payload_artifacts len = %d, want %d", len(payloadArtifacts), len(expected))
	}
	for i := range expected {
		got := manifestArtifactDigestIdentity(t, payloadArtifacts[i])
		if got != expected[i] {
			t.Fatalf("payload_artifacts[%d] digest = %q, want %q", i, got, expected[i])
		}
	}
}

func manifestArtifactDigestIdentity(t *testing.T, value any) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal payload artifact returned error: %v", err)
	}
	var artifactRef struct {
		Digest trustpolicy.Digest `json:"digest"`
	}
	if err := json.Unmarshal(b, &artifactRef); err != nil {
		t.Fatalf("Unmarshal payload artifact returned error: %v", err)
	}
	identity, err := artifactRef.Digest.Identity()
	if err != nil {
		t.Fatalf("Digest.Identity returned error: %v", err)
	}
	return identity
}

func putDependencyArtifactForManifestTest(t *testing.T, s *Service, dataClass artifacts.DataClass, payload []byte, contentType string) artifacts.ArtifactReference {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           contentType,
		DataClass:             dataClass,
		ProvenanceReceiptHash: artifacts.DigestBytes([]byte("manifest-test:" + artifacts.DigestBytes(payload))),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
		StepID:                "dependency_fetch",
	})
	if err != nil {
		t.Fatalf("Put(%s) returned error: %v", dataClass, err)
	}
	return ref
}
