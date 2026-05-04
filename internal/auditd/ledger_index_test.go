package auditd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestLookupRecordDigestRefreshesOnIndexDrift(t *testing.T) {
	root, ledger, result := appendFixtureAndBuildIndex(t)
	recordID, _ := result.RecordDigest.Identity()
	tamperDerivedIndexForTest(t, root, func(index *derivedIndex) {
		index.RecordDigestLookup[recordID] = RecordLookup{SegmentID: "segment-999999", FrameIndex: 0}
	})

	lookup, ok, err := ledger.LookupRecordDigest(recordID)
	if err != nil {
		t.Fatalf("LookupRecordDigest returned error: %v", err)
	}
	if !ok {
		t.Fatal("LookupRecordDigest found=false, want true")
	}
	if lookup.SegmentID != "segment-000001" || lookup.FrameIndex != 0 {
		t.Fatalf("LookupRecordDigest = %+v, want segment-000001 frame 0", lookup)
	}

	repaired := mustReadDerivedIndex(t, root)
	if repairedLookup := repaired.RecordDigestLookup[recordID]; repairedLookup.SegmentID != "segment-000001" || repairedLookup.FrameIndex != 0 {
		t.Fatalf("repaired index record lookup = %+v, want segment-000001 frame 0", repairedLookup)
	}
}

func TestLookupRecordDigestFailsClosedWhenCanonicalFrameDigestCannotBeValidated(t *testing.T) {
	root, ledger, result := appendFixtureAndBuildIndex(t)
	recordID, _ := result.RecordDigest.Identity()

	segmentPath := filepath.Join(root, segmentsDirName, "segment-000001.json")
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := readJSONFile(segmentPath, &segment); err != nil {
		t.Fatalf("readJSONFile(segment) returned error: %v", err)
	}
	segment.Frames[0].CanonicalSignedEnvelopeBytes = "not-base64"
	if err := writeCanonicalJSONFile(segmentPath, segment); err != nil {
		t.Fatalf("writeCanonicalJSONFile(segment) returned error: %v", err)
	}

	if _, _, err := ledger.LookupRecordDigest(recordID); err == nil {
		t.Fatal("LookupRecordDigest returned nil error, want fail-closed error when canonical frame cannot be validated")
	}
}

func TestLookupSealDigestByChainIndexRefreshesOnIndexDrift(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seal := mustSealFixtureSegment(t, ledger, fixture)
	realSealID, _ := seal.SealEnvelopeDigest.Identity()
	if _, err := ledger.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	tamperDerivedIndexForTest(t, root, func(index *derivedIndex) {
		index.SealChainIndexLookup["0"] = "sha256:" + strings.Repeat("a", 64)
	})

	sealID, ok, err := ledger.LookupSealDigestByChainIndex(0)
	if err != nil {
		t.Fatalf("LookupSealDigestByChainIndex returned error: %v", err)
	}
	if !ok {
		t.Fatal("LookupSealDigestByChainIndex found=false, want true")
	}
	if sealID != realSealID {
		t.Fatalf("LookupSealDigestByChainIndex digest=%q, want %q", sealID, realSealID)
	}

	repaired := mustReadDerivedIndex(t, root)
	if repaired.SealChainIndexLookup["0"] != realSealID {
		t.Fatalf("repaired index chain lookup = %q, want %q", repaired.SealChainIndexLookup["0"], realSealID)
	}
}

func TestLookupSealDigestByChainIndexFailsClosedOnCanonicalConflict(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	_ = mustSealFixtureSegment(t, ledger, fixture)

	conflictEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, nil, 0)
	conflictPayload, err := decodeAndValidateSealEnvelope(conflictEnvelope)
	if err != nil {
		t.Fatalf("decodeAndValidateSealEnvelope returned error: %v", err)
	}
	conflictPayload.SegmentID = "segment-999999"
	conflictBytes := mustJSON(t, conflictPayload)
	conflictEnvelope.Payload = conflictBytes
	conflictDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(conflictEnvelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	conflictID, err := conflictDigest.Identity()
	if err != nil {
		t.Fatalf("conflictDigest.Identity() returned error: %v", err)
	}
	conflictPath := filepath.Join(root, sidecarDirName, sealsDirName, strings.TrimPrefix(conflictID, "sha256:")+".json")
	if err := writeCanonicalJSONFile(conflictPath, conflictEnvelope); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
	removeDerivedIndexArtifactsForTest(t, root)

	if _, _, err := ledger.LookupSealDigestByChainIndex(0); err == nil || !strings.Contains(err.Error(), "multiple seals share chain index") {
		t.Fatalf("LookupSealDigestByChainIndex error=%v, want canonical conflict", err)
	}
}

func TestOpenFailsClosedWhenLatestSealIndexBuildDetectsCanonicalConflict(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	_ = mustSealFixtureSegment(t, ledger, fixture)

	conflictEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, nil, 0)
	conflictPayload, err := decodeAndValidateSealEnvelope(conflictEnvelope)
	if err != nil {
		t.Fatalf("decodeAndValidateSealEnvelope returned error: %v", err)
	}
	conflictPayload.SegmentID = "segment-999999"
	conflictBytes := mustJSON(t, conflictPayload)
	conflictEnvelope.Payload = conflictBytes
	conflictDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(conflictEnvelope)
	if err != nil {
		t.Fatalf("ComputeSignedEnvelopeAuditRecordDigest returned error: %v", err)
	}
	conflictID, err := conflictDigest.Identity()
	if err != nil {
		t.Fatalf("conflictDigest.Identity() returned error: %v", err)
	}
	conflictPath := filepath.Join(root, sidecarDirName, sealsDirName, strings.TrimPrefix(conflictID, "sha256:")+".json")
	if err := writeCanonicalJSONFile(conflictPath, conflictEnvelope); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
	if err := os.Remove(filepath.Join(root, indexDirName, auditEvidenceIndexFileName)); err != nil {
		t.Fatalf("Remove(index) returned error: %v", err)
	}

	if _, err := Open(root); err == nil || !strings.Contains(err.Error(), "multiple seals share chain index") {
		t.Fatalf("Open error=%v, want canonical conflict", err)
	}
}

func TestLatestVerificationReportRecoversWhenIndexDigestDrifts(t *testing.T) {
	root, ledger, _ := setupLedgerWithAdmissionFixture(t)
	report := validReportFixture("segment-000001")
	digest := mustPersistReport(t, ledger, report)
	if _, err := ledger.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	tamperDerivedIndexForTest(t, root, func(index *derivedIndex) {
		index.LatestVerificationReportDigest = "sha256:" + strings.Repeat("c", 64)
	})

	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopened) returned error: %v", err)
	}
	loaded, err := reopened.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	loadedDigest, err := canonicalDigest(loaded)
	if err != nil {
		t.Fatalf("canonicalDigest returned error: %v", err)
	}
	loadedID, _ := loadedDigest.Identity()
	expectedID, _ := digest.Identity()
	if loadedID != expectedID {
		t.Fatalf("loaded report digest = %q, want %q", loadedID, expectedID)
	}

	repaired := mustReadDerivedIndex(t, root)
	if repaired.LatestVerificationReportDigest != expectedID {
		t.Fatalf("repaired latest report digest = %q, want %q", repaired.LatestVerificationReportDigest, expectedID)
	}
}

func TestLatestVerificationReportFailsClosedWhenReportSidecarIsTampered(t *testing.T) {
	root, ledger, _ := setupLedgerWithAdmissionFixture(t)
	report := validReportFixture("segment-000001")
	digest := mustPersistReport(t, ledger, report)
	if _, err := ledger.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	identity, _ := digest.Identity()
	reportPath := filepath.Join(root, sidecarDirName, verificationReportsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	tampered := validReportFixture("segment-000001")
	tampered.Summary = "tampered report"
	if err := writeCanonicalJSONFile(reportPath, tampered); err != nil {
		t.Fatalf("writeCanonicalJSONFile(report) returned error: %v", err)
	}

	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopened) returned error: %v", err)
	}
	if _, err := reopened.LatestVerificationReport(); err == nil || !strings.Contains(err.Error(), "verification report sidecar digest mismatch") {
		t.Fatalf("LatestVerificationReport error=%v, want digest mismatch fail-closed", err)
	}
}

func TestDerivedIndexRebuildMatchesIncrementalState(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	firstRecordID := recordDigestIdentity(t, ledger, 0)
	secondAppend := appendAndSealSecondSegment(t, ledger, fixture)
	reportID := persistIndexedLatestReport(t, ledger, "segment-000002")

	incremental := mustReadDerivedIndex(t, root)
	assertIncrementalIndexState(t, incremental, firstRecordID, secondAppend, reportID)

	rebuilt := mustBuildIndex(t, ledger)
	if !derivedIndexEqualIgnoringBuiltAt(incremental, rebuilt) {
		t.Fatalf("rebuilt index differs from incremental\nincremental=%+v\nrebuilt=%+v", incremental, rebuilt)
	}
}

func TestLookupRecordDigestFailsClosedWhenLegacyDerivedIndexOverwritesShardedIndex(t *testing.T) {
	root, ledger, result := appendFixtureAndBuildIndex(t)
	recordID, _ := result.RecordDigest.Identity()
	stale := mustReadDerivedIndex(t, root)
	stale.RecordDigestLookup = map[string]RecordLookup{recordID: {SegmentID: "segment-stale", FrameIndex: 99}}
	if err := writeCanonicalJSONFile(filepath.Join(root, indexDirName, auditEvidenceIndexFileName), stale); err != nil {
		t.Fatalf("writeCanonicalJSONFile(legacy index) returned error: %v", err)
	}
	legacyPath := filepath.Join(root, indexDirName, auditEvidenceIndexFileName)
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(legacyPath, future, future); err != nil {
		t.Fatalf("Chtimes(legacy index) returned error: %v", err)
	}
	if _, _, err := ledger.LookupRecordDigest(recordID); err == nil || !strings.Contains(err.Error(), "legacy representation is newer") {
		t.Fatalf("LookupRecordDigest error=%v, want fail-closed stale legacy representation", err)
	}
}

func TestOpenUsesIndexBackedLatestSealDiscoveryWithoutRescanningMalformedOldSeals(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	firstSealDigest := sealFirstSegmentForOpenRecovery(t, ledger, fixture)
	buildSecondSealedSegmentForOpenRecovery(t, ledger, fixture, &firstSealDigest)
	if _, err := ledger.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}
	tamperOldSealSidecar(t, root, firstSealDigest)
	assertReopenedLastSealedSegmentID(t, root, "segment-000002")
}

func TestOpenUsesIndexBackedLatestReportDiscoveryWithoutRescanningMalformedOlderReports(t *testing.T) {
	root, ledger, _ := setupLedgerWithAdmissionFixture(t)
	older := validReportFixture("segment-000001")
	older.VerifiedAt = "2026-03-13T12:30:00Z"
	olderDigest := mustPersistReport(t, ledger, older)
	latest := validReportFixture("segment-000001")
	latest.VerifiedAt = "2026-03-13T12:45:00Z"
	latestDigest := mustPersistReport(t, ledger, latest)
	if _, err := ledger.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	olderID, _ := olderDigest.Identity()
	olderPath := filepath.Join(root, sidecarDirName, verificationReportsDirName, strings.TrimPrefix(olderID, "sha256:")+".json")
	if err := os.WriteFile(olderPath, []byte(`{"bad":`), 0o600); err != nil {
		t.Fatalf("WriteFile(olderPath) returned error: %v", err)
	}

	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopened) returned error: %v", err)
	}
	report, err := reopened.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	gotDigest, err := canonicalDigest(report)
	if err != nil {
		t.Fatalf("canonicalDigest returned error: %v", err)
	}
	gotID, _ := gotDigest.Identity()
	wantID, _ := latestDigest.Identity()
	if gotID != wantID {
		t.Fatalf("latest report digest = %q, want %q", gotID, wantID)
	}
}

func derivedIndexEqualIgnoringBuiltAt(left, right derivedIndex) bool {
	left.BuiltAt = ""
	right.BuiltAt = ""
	return reflect.DeepEqual(left, right)
}

func tamperDerivedIndexForTest(t *testing.T, root string, mutate func(*derivedIndex)) {
	t.Helper()
	index := mustReadDerivedIndex(t, root)
	mutate(&index)
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open(root) returned error: %v", err)
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if err := ledger.saveDerivedIndexLocked(index); err != nil {
		t.Fatalf("saveDerivedIndexLocked returned error: %v", err)
	}
}

func mustReadDerivedIndex(t *testing.T, root string) derivedIndex {
	t.Helper()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open(root) returned error: %v", err)
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	index, exists, err := ledger.loadDerivedIndexLocked()
	if err != nil {
		t.Fatalf("loadDerivedIndexLocked returned error: %v", err)
	}
	if !exists {
		t.Fatal("loadDerivedIndexLocked found=false, want true")
	}
	return index
}

func removeDerivedIndexArtifactsForTest(t *testing.T, root string) {
	t.Helper()
	paths := []string{
		filepath.Join(root, indexDirName, auditEvidenceIndexFileName),
		filepath.Join(root, indexDirName, indexMetaFileName),
		filepath.Join(root, indexDirName, indexRecordLookupDirName),
		filepath.Join(root, indexDirName, indexSegmentSealDirName),
		filepath.Join(root, indexDirName, indexSealChainDirName),
		filepath.Join(root, indexDirName, indexRunTimelineDirName),
	}
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("RemoveAll(%q) returned error: %v", path, err)
		}
	}
}

func appendAndSealSecondSegment(t *testing.T, ledger *Ledger, fixture auditFixtureKey) AppendResult {
	t.Helper()
	firstSeal := sealLoadedSegment(t, ledger, fixture, "segment-000001", nil, 0)
	secondReq := validAdmissionRequestForLedger(t, newAuditFixtureKey(t))
	secondAppend, err := ledger.AppendAdmittedEvent(secondReq)
	if err != nil {
		t.Fatalf("AppendAdmittedEvent(second) returned error: %v", err)
	}
	sealLoadedSegment(t, ledger, fixture, "segment-000002", &firstSeal.SealEnvelopeDigest, 1)
	return secondAppend
}

func sealFirstSegmentForOpenRecovery(t *testing.T, ledger *Ledger, fixture auditFixtureKey) trustpolicy.Digest {
	t.Helper()
	firstSeal := sealLoadedSegment(t, ledger, fixture, "segment-000001", nil, 0)
	return firstSeal.SealEnvelopeDigest
}

func buildSecondSealedSegmentForOpenRecovery(t *testing.T, ledger *Ledger, fixture auditFixtureKey, previous *trustpolicy.Digest) {
	t.Helper()
	secondReq := validAdmissionRequestForLedger(t, newAuditFixtureKey(t))
	if _, err := ledger.AppendAdmittedEvent(secondReq); err != nil {
		t.Fatalf("AppendAdmittedEvent(second) returned error: %v", err)
	}
	sealLoadedSegment(t, ledger, fixture, "segment-000002", previous, 1)
}

func sealLoadedSegment(t *testing.T, ledger *Ledger, fixture auditFixtureKey, segmentID string, previous *trustpolicy.Digest, chainIndex int64) SealResult {
	t.Helper()
	segment, err := ledger.loadSegment(segmentID)
	if err != nil {
		t.Fatalf("loadSegment(%s) returned error: %v", segmentID, err)
	}
	sealEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, previous, chainIndex)
	seal, err := ledger.SealCurrentSegment(sealEnvelope)
	if err != nil {
		t.Fatalf("SealCurrentSegment(%s) returned error: %v", segmentID, err)
	}
	return seal
}

func persistIndexedLatestReport(t *testing.T, ledger *Ledger, segmentID string) string {
	t.Helper()
	reportDigest := mustPersistReport(t, ledger, validReportFixture(segmentID))
	reportID, _ := reportDigest.Identity()
	return reportID
}

func assertIncrementalIndexState(t *testing.T, incremental derivedIndex, firstRecordID string, secondAppend AppendResult, reportID string) {
	t.Helper()
	if incremental.TotalRecords != 2 {
		t.Fatalf("incremental TotalRecords = %d, want 2", incremental.TotalRecords)
	}
	if incremental.RecordDigestLookup[firstRecordID].SegmentID != "segment-000001" {
		t.Fatalf("incremental first record lookup = %+v, want segment-000001", incremental.RecordDigestLookup[firstRecordID])
	}
	secondRecordID, _ := secondAppend.RecordDigest.Identity()
	if incremental.RecordDigestLookup[secondRecordID].SegmentID != "segment-000002" {
		t.Fatalf("incremental second record lookup = %+v, want segment-000002", incremental.RecordDigestLookup[secondRecordID])
	}
	if incremental.LatestVerificationReportDigest != reportID {
		t.Fatalf("incremental latest report digest = %q, want %q", incremental.LatestVerificationReportDigest, reportID)
	}
}

func tamperOldSealSidecar(t *testing.T, root string, firstSealDigest trustpolicy.Digest) {
	t.Helper()
	firstSealID, _ := firstSealDigest.Identity()
	firstSealPath := filepath.Join(root, sidecarDirName, sealsDirName, strings.TrimPrefix(firstSealID, "sha256:")+".json")
	if err := os.WriteFile(firstSealPath, []byte(`{"bad":`), 0o600); err != nil {
		t.Fatalf("WriteFile(firstSealPath) returned error: %v", err)
	}
}

func assertReopenedLastSealedSegmentID(t *testing.T, root string, want string) {
	t.Helper()
	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopened) returned error: %v", err)
	}
	state, err := reopened.loadState()
	if err != nil {
		t.Fatalf("loadState returned error: %v", err)
	}
	if state.LastSealedSegmentID != want {
		t.Fatalf("LastSealedSegmentID = %q, want %q", state.LastSealedSegmentID, want)
	}
}
