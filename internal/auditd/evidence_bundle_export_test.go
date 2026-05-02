package auditd

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestExportEvidenceBundleRunScopeStreamsTarWithoutFullAssembly(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	secondReportDigest := seedRunScopedBundleFixture(t, ledger, fixture)
	exported := exportRunScopedEvidenceBundle(t, ledger)
	defer exported.Reader.Close()
	assertRunScopedBundleManifest(t, exported.Manifest)
	archiveBytes := mustReadExportedBundle(t, exported.Reader)
	entries := readTarEntries(t, archiveBytes)
	assertRunScopedBundleArchive(t, entries)
	assertIncludedBundleObjectsPresent(t, exported.Manifest.IncludedObjects, entries)
	secondReportIdentity, _ := secondReportDigest.Identity()
	if includesDigest(exported.Manifest.IncludedObjects, secondReportIdentity) {
		t.Fatal("run-scoped manifest unexpectedly includes report for unrelated segment")
	}
}

func seedRunScopedBundleFixture(t *testing.T, ledger *Ledger, fixture auditFixtureKey) trustpolicy.Digest {
	t.Helper()
	firstSeal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, firstSeal.SealEnvelopeDigest))
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))
	request := validAdmissionRequestForLedger(t, fixture)
	request.Envelope = signedEnvelopeForRunAndSession(t, fixture, request.Envelope, "run-2", "session-2", 2)
	if _, err := ledger.AppendAdmittedEvent(request); err != nil {
		t.Fatalf("AppendAdmittedEvent(second run) returned error: %v", err)
	}
	secondSeal := mustSealSegmentForExportWithChain(t, ledger, fixture, "segment-000002", &firstSeal.SealEnvelopeDigest, 1)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, secondSeal.SealEnvelopeDigest))
	return mustPersistReport(t, ledger, validReportFixture("segment-000002"))
}

func exportRunScopedEvidenceBundle(t *testing.T, ledger *Ledger) AuditEvidenceBundleExport {
	t.Helper()
	exported, err := ledger.ExportEvidenceBundle(AuditEvidenceBundleExportRequest{ManifestRequest: AuditEvidenceBundleManifestRequest{Scope: AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"}, ExportProfile: "operator_private_full", CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"}, DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false}}, ArchiveFormat: "tar"})
	if err != nil {
		t.Fatalf("ExportEvidenceBundle returned error: %v", err)
	}
	return exported
}

func assertRunScopedBundleManifest(t *testing.T, manifest AuditEvidenceBundleManifest) {
	t.Helper()
	if manifest.Scope.ScopeKind != "run" || manifest.Scope.RunID != "run-1" {
		t.Fatalf("manifest.scope = %+v, want run scope run-1", manifest.Scope)
	}
	if len(manifest.SealReferences) != 1 || manifest.SealReferences[0].SegmentID != "segment-000001" {
		t.Fatalf("manifest.seal_references = %+v, want only segment-000001", manifest.SealReferences)
	}
}

func mustReadExportedBundle(t *testing.T, reader io.Reader) []byte {
	t.Helper()
	archiveBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll(exported.Reader) returned error: %v", err)
	}
	return archiveBytes
}

func assertRunScopedBundleArchive(t *testing.T, entries map[string][]byte) {
	t.Helper()
	if len(entries) == 0 {
		t.Fatal("tar entries empty, want streamed bundle archive")
	}
	if _, ok := entries["manifest.json"]; !ok {
		t.Fatal("manifest.json missing from archive")
	}
	if _, ok := entries["segments/segment-000001.json"]; !ok {
		t.Fatal("run-scoped archive missing selected segment")
	}
	if _, ok := entries["segments/segment-000002.json"]; ok {
		t.Fatal("run-scoped archive unexpectedly contains unrelated segment")
	}
}

func assertIncludedBundleObjectsPresent(t *testing.T, objects []AuditEvidenceBundleIncludedObject, entries map[string][]byte) {
	t.Helper()
	for _, object := range objects {
		if object.Path == "" {
			t.Fatalf("included object has empty path: %+v", object)
		}
		content, ok := entries[object.Path]
		if !ok {
			t.Fatalf("archive missing included object path %q", object.Path)
		}
		if int64(len(content)) != object.ByteLength {
			t.Fatalf("archive object %q size = %d, want %d", object.Path, len(content), object.ByteLength)
		}
	}
}

func TestBuildEvidenceBundleManifestFailsClosedForUnresolvedArtifactAndIncidentScopes(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))

	artifactDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("a", 64)}
	artifactIdentity, _ := artifactDigest.Identity()
	_, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{Scope: AuditEvidenceBundleScope{ScopeKind: "artifact", ArtifactDigests: []string{artifactIdentity}}, ExportProfile: "external_relying_party_minimal", CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"}, DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "digest_metadata_only", SelectiveDisclosureApplied: true}})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest(artifact) expected fail-closed error")
	}

	_, err = ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{Scope: AuditEvidenceBundleScope{ScopeKind: "incident", IncidentID: "incident-42"}, ExportProfile: "incident_response_scope", CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"}, DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: true}})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest(incident) expected fail-closed error")
	}
}

func TestOfflineVerifyEvidenceBundleRejectsDuplicateTarPaths(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	archive := duplicateManifestTarArchive(t)
	_, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(archive), "tar")
	if err == nil {
		t.Fatal("OfflineVerifyEvidenceBundle expected duplicate-path error")
	}
}

func TestExportEvidenceBundleStreamsLargeEvidenceSet(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealedSegments := 80

	seedLargeEvidenceRun(t, ledger, fixture, sealedSegments)

	exported, err := ledger.ExportEvidenceBundle(AuditEvidenceBundleExportRequest{
		ManifestRequest: AuditEvidenceBundleManifestRequest{
			Scope:             AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
			ExportProfile:     "operator_private_full",
			CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
			DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
		},
		ArchiveFormat: "tar",
	})
	if err != nil {
		t.Fatalf("ExportEvidenceBundle returned error: %v", err)
	}
	defer exported.Reader.Close()

	assertLargeEvidenceManifest(t, exported.Manifest, sealedSegments)
	archive, readOps, totalBytes := readBundleStreamInChunks(t, exported.Reader, 257)
	assertLargeEvidenceStreaming(t, totalBytes, readOps)
	entries := readTarEntries(t, archive.Bytes())
	if _, ok := entries["manifest.json"]; !ok {
		t.Fatal("manifest.json missing from large streamed archive")
	}
	if _, ok := entries[fmt.Sprintf("segments/segment-%06d.json", sealedSegments)]; !ok {
		t.Fatalf("archive missing highest segment path for large evidence set")
	}
}

func seedLargeEvidenceRun(t *testing.T, ledger *Ledger, fixture auditFixtureKey, sealedSegments int) {
	t.Helper()
	firstSeal := mustSealFixtureSegment(t, ledger, fixture)
	previousDigest := firstSeal.SealEnvelopeDigest
	for i := 2; i <= sealedSegments; i++ {
		appendRunSegmentForExportTest(t, ledger, fixture, i, &previousDigest)
	}
}

func appendRunSegmentForExportTest(t *testing.T, ledger *Ledger, fixture auditFixtureKey, segmentNumber int, previousDigest *trustpolicy.Digest) {
	t.Helper()
	request := validAdmissionRequestForLedger(t, fixture)
	request.Envelope = signedEnvelopeForRunAndSession(t, fixture, request.Envelope, "run-1", fmt.Sprintf("session-%d", segmentNumber), int64(segmentNumber))
	if _, err := ledger.AppendAdmittedEvent(request); err != nil {
		t.Fatalf("AppendAdmittedEvent(segment=%d) returned error: %v", segmentNumber, err)
	}
	segmentID := fmt.Sprintf("segment-%06d", segmentNumber)
	seal := mustSealSegmentForExportWithChain(t, ledger, fixture, segmentID, previousDigest, int64(segmentNumber-1))
	*previousDigest = seal.SealEnvelopeDigest
}

func assertLargeEvidenceManifest(t *testing.T, manifest AuditEvidenceBundleManifest, sealedSegments int) {
	t.Helper()
	if got := len(manifest.SealReferences); got != sealedSegments {
		t.Fatalf("seal_references len = %d, want %d", got, sealedSegments)
	}
	segmentObjects := 0
	for i := range manifest.IncludedObjects {
		if manifest.IncludedObjects[i].ObjectFamily == "audit_segment" {
			segmentObjects++
		}
	}
	if segmentObjects != sealedSegments {
		t.Fatalf("audit_segment object count = %d, want %d", segmentObjects, sealedSegments)
	}
}

func readBundleStreamInChunks(t *testing.T, reader io.Reader, chunkSize int) (bytes.Buffer, int, int) {
	t.Helper()
	buf := make([]byte, chunkSize)
	readOps := 0
	totalBytes := 0
	var archive bytes.Buffer
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			readOps++
			totalBytes += n
			if _, err := archive.Write(buf[:n]); err != nil {
				t.Fatalf("archive.Write returned error: %v", err)
			}
		}
		if readErr == io.EOF {
			return archive, readOps, totalBytes
		}
		if readErr != nil {
			t.Fatalf("stream read returned error: %v", readErr)
		}
	}
}

func assertLargeEvidenceStreaming(t *testing.T, totalBytes int, readOps int) {
	t.Helper()
	if readOps < 20 {
		t.Fatalf("stream read operations = %d, want many chunked reads for large bundle", readOps)
	}
	if totalBytes <= 0 {
		t.Fatal("stream total bytes = 0, want non-empty archive")
	}
}

func signedEnvelopeForRunAndSession(t *testing.T, fixture auditFixtureKey, envelope trustpolicy.SignedObjectEnvelope, runID string, sessionID string, seq int64) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal(envelope.Payload) returned error: %v", err)
	}
	payload["seq"] = seq
	payload["scope"] = map[string]any{"workspace_id": "workspace-1", "run_id": runID, "stage_id": "stage-1"}
	payload["correlation"] = map[string]any{"session_id": sessionID, "operation_id": "op-1"}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal(payload) returned error: %v", err)
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(rawPayload)
	if err != nil {
		t.Fatalf("Transform(payload) returned error: %v", err)
	}
	signature := ed25519.Sign(fixture.privateKey, canonicalPayload)
	envelope.Payload = rawPayload
	envelope.Signature = trustpolicy.SignatureBlock{
		Alg:        "ed25519",
		KeyID:      trustpolicy.KeyIDProfile,
		KeyIDValue: fixture.keyIDValue,
		Signature:  base64.StdEncoding.EncodeToString(signature),
	}
	return envelope
}

func readTarEntries(t *testing.T, archive []byte) map[string][]byte {
	t.Helper()
	entries := map[string][]byte{}
	tr := tar.NewReader(bytes.NewReader(archive))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return entries
		}
		if err != nil {
			t.Fatalf("tar.Next returned error: %v", err)
		}
		payload, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("ReadAll(%q) returned error: %v", header.Name, err)
		}
		entries[header.Name] = payload
	}
}

func duplicateManifestTarArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte(`{"schema_id":"runecode.protocol.v0.AuditEvidenceBundleManifest","schema_version":"0.1.0","bundle_id":"bundle-test","created_at":"2026-01-01T00:00:00Z","created_by_tool":{"tool_name":"runecode-broker","tool_version":"0.0.0-dev"},"export_profile":"operator_private_full","scope":{"scope_kind":"operator_private"},"verifier_identity":{"key_id":"key_sha256","key_id_value":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","logical_purpose":"audit_verifier","logical_scope":"node"},"disclosure_posture":{"posture":"operator_private","selective_disclosure_applied":false}}`)
	for i := 0; i < 2; i++ {
		header := &tar.Header{Name: "manifest.json", Mode: 0o600, Size: int64(len(content))}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader returned error: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	return buf.Bytes()
}

func includesDigest(objects []AuditEvidenceBundleIncludedObject, digest string) bool {
	for i := range objects {
		if objects[i].Digest == digest {
			return true
		}
	}
	return false
}

func mustSealSegmentForExportWithChain(t *testing.T, ledger *Ledger, fixture auditFixtureKey, segmentID string, previous *trustpolicy.Digest, chainIndex int64) SealResult {
	t.Helper()
	segment, err := ledger.loadSegment(segmentID)
	if err != nil {
		t.Fatalf("loadSegment(%s) returned error: %v", segmentID, err)
	}
	sealEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, previous, chainIndex)
	result, err := ledger.SealCurrentSegment(sealEnvelope)
	if err != nil {
		t.Fatalf("SealCurrentSegment(%s) returned error: %v", segmentID, err)
	}
	return result
}
