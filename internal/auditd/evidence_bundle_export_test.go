package auditd

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func TestBuildEvidenceBundleManifestSupportsDeterministicArtifactAndIncidentScopes(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	setIncidentCorrelationOnCurrentOpenSegmentFrame(t, ledger, "incident-42")
	seal := mustSealFixtureSegment(t, ledger, fixture)
	_ = mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))
	_ = mustPersistReport(t, ledger, validReportFixture("segment-000001"))
	inclusion := mustRecordInclusionForRun1(t, ledger)
	artifactManifest, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{Scope: AuditEvidenceBundleScope{ScopeKind: "artifact", ArtifactDigests: []string{inclusion.RecordDigest}}, ExportProfile: "operator_private_full", CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"}, DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false}})
	if err != nil {
		t.Fatalf("BuildEvidenceBundleManifest(artifact) returned error: %v", err)
	}
	if len(artifactManifest.SealReferences) != 1 || artifactManifest.SealReferences[0].SegmentID != inclusion.SegmentID {
		t.Fatalf("artifact scope selected unexpected seal references: %+v", artifactManifest.SealReferences)
	}
	request := validAdmissionRequestForLedger(t, fixture)
	request.Envelope = signedEnvelopeForRunAndSession(t, fixture, request.Envelope, "run-2", "session-2", 2)
	if _, err := ledger.AppendAdmittedEvent(request); err != nil {
		t.Fatalf("AppendAdmittedEvent(run-2) returned error: %v", err)
	}
	if _, err := ledger.SealCurrentSegment(newSealEnvelopeForCurrentSegment(t, ledger, fixture)); err != nil {
		t.Fatalf("SealCurrentSegment(run-2) returned error: %v", err)
	}
	incidentManifest, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{Scope: AuditEvidenceBundleScope{ScopeKind: "incident", IncidentID: "incident-42"}, ExportProfile: "incident_response_scope", CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"}, DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: true}})
	if err != nil {
		t.Fatalf("BuildEvidenceBundleManifest(incident) returned error: %v", err)
	}
	if len(incidentManifest.SealReferences) != 1 || incidentManifest.SealReferences[0].SegmentID != inclusion.SegmentID {
		t.Fatalf("incident scope selected unexpected seal references: %+v", incidentManifest.SealReferences)
	}
}

func TestBuildEvidenceBundleManifestArtifactScopeFailsClosedOnCorruptReceiptSidecar(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seal := mustSealFixtureSegment(t, ledger, fixture)
	receiptDigest := mustPersistReceipt(t, ledger, buildAnchorReceiptEnvelope(t, fixture, seal.SealEnvelopeDigest))

	receiptID, _ := receiptDigest.Identity()
	receiptPath := filepath.Join(root, sidecarDirName, receiptsDirName, strings.TrimPrefix(receiptID, "sha256:")+".json")
	if err := os.WriteFile(receiptPath, []byte(`{"bad":`), 0o600); err != nil {
		t.Fatalf("WriteFile(receiptPath) returned error: %v", err)
	}

	_, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "artifact", ArtifactDigests: []string{receiptID}},
		ExportProfile:     "operator_private_full",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
	})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest returned nil error, want fail-closed error for corrupt receipt sidecar")
	}
}

func TestBuildEvidenceBundleManifestArtifactScopeFailsClosedOnCorruptReportSidecar(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	_ = mustSealFixtureSegment(t, ledger, fixture)
	reportDigest := mustPersistReport(t, ledger, validReportFixture("segment-000001"))

	reportID, _ := reportDigest.Identity()
	reportPath := filepath.Join(root, sidecarDirName, verificationReportsDirName, strings.TrimPrefix(reportID, "sha256:")+".json")
	if err := os.WriteFile(reportPath, []byte(`{"bad":`), 0o600); err != nil {
		t.Fatalf("WriteFile(reportPath) returned error: %v", err)
	}

	_, err := ledger.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             AuditEvidenceBundleScope{ScopeKind: "artifact", ArtifactDigests: []string{reportID}},
		ExportProfile:     "operator_private_full",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
	})
	if err == nil {
		t.Fatal("BuildEvidenceBundleManifest returned nil error, want fail-closed error for corrupt report sidecar")
	}
}

func mustRecordInclusionForRun1(t *testing.T, ledger *Ledger) AuditRecordInclusion {
	t.Helper()
	index, err := ledger.BuildIndex()
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	for i := range index.RunTimeline {
		if index.RunTimeline[i].RunID != "run-1" {
			continue
		}
		inc, ok, err := ledger.RecordInclusionByDigest(index.RunTimeline[i].RecordDigest)
		if err != nil {
			t.Fatalf("RecordInclusionByDigest returned error: %v", err)
		}
		if ok {
			return inc
		}
	}
	t.Fatal("no record inclusion found for run-1")
	return AuditRecordInclusion{}
}

func setIncidentCorrelationOnCurrentOpenSegmentFrame(t *testing.T, ledger *Ledger, incidentID string) {
	t.Helper()
	segment, idx := mustLoadCurrentOpenSegmentAndFrameIndex(t, ledger)
	envelope := mustDecodeFrameEnvelope(t, segment.Frames[idx])
	payload := mustUnmarshalEnvelopePayloadMap(t, envelope.Payload)
	payload["correlation"] = incidentCorrelationMap(payload["correlation"], incidentID)
	mutated, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal(mutated payload) returned error: %v", err)
	}
	envelope.Payload = mutated
	canonBytes, digest, err := canonicalEnvelopeAndDigest(envelope)
	if err != nil {
		t.Fatalf("canonicalEnvelopeAndDigest returned error: %v", err)
	}
	segment.Frames[idx].CanonicalSignedEnvelopeBytes = base64.StdEncoding.EncodeToString(canonBytes)
	segment.Frames[idx].RecordDigest = digest
	segment.Frames[idx].ByteLength = int64(len(canonBytes))
	if err := writeCanonicalJSONFile(filepath.Join(ledger.rootDir, segmentsDirName, segment.Header.SegmentID+".json"), segment); err != nil {
		t.Fatalf("writeCanonicalJSONFile(segment) returned error: %v", err)
	}
}

func mustLoadCurrentOpenSegmentAndFrameIndex(t *testing.T, ledger *Ledger) (trustpolicy.AuditSegmentFilePayload, int) {
	t.Helper()
	state, err := ledger.loadState()
	if err != nil {
		t.Fatalf("loadState returned error: %v", err)
	}
	segment, err := ledger.loadSegment(state.CurrentOpenSegmentID)
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	if len(segment.Frames) == 0 {
		t.Fatal("current segment has no frames")
	}
	return segment, len(segment.Frames) - 1
}

func mustDecodeFrameEnvelope(t *testing.T, frame trustpolicy.AuditSegmentRecordFrame) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		t.Fatalf("decodeFrameEnvelope returned error: %v", err)
	}
	return envelope
}

func mustUnmarshalEnvelopePayloadMap(t *testing.T, payloadBytes []byte) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("Unmarshal(event payload) returned error: %v", err)
	}
	return payload
}

func incidentCorrelationMap(existing any, incidentID string) map[string]any {
	corr := map[string]any{}
	existingMap, ok := existing.(map[string]any)
	if ok {
		for key, value := range existingMap {
			corr[key] = value
		}
	}
	corr["incident_id"] = incidentID
	return corr
}

func newSealEnvelopeForCurrentSegment(t *testing.T, ledger *Ledger, fixture auditFixtureKey) trustpolicy.SignedObjectEnvelope {
	t.Helper()
	state, err := ledger.loadState()
	if err != nil {
		t.Fatalf("loadState returned error: %v", err)
	}
	segment, err := ledger.loadSegment(state.CurrentOpenSegmentID)
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	var previous *trustpolicy.Digest
	if state.LastSealEnvelopeDigest != "" {
		d, err := digestFromIdentity(state.LastSealEnvelopeDigest)
		if err != nil {
			t.Fatalf("digestFromIdentity(last seal) returned error: %v", err)
		}
		previous = &d
	}
	chainIndex := int64(0)
	if previous != nil {
		if sealedCount, err := countSealedSegments(ledger); err == nil {
			chainIndex = int64(sealedCount)
		}
	}
	return buildSealEnvelopeForSegment(t, fixture, ledger, segment, previous, chainIndex)
}

func countSealedSegments(ledger *Ledger) (int, error) {
	index, err := ledger.BuildIndex()
	if err != nil {
		return 0, err
	}
	return len(index.SegmentSealLookup), nil
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
