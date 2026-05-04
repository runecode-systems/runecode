package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAppendReloadRecoveryAndIndex(t *testing.T) {
	root, ledger, result := appendFixtureAndBuildIndex(t)
	if result.FrameCount != 1 {
		t.Fatalf("FrameCount = %d, want 1", result.FrameCount)
	}
	index := mustBuildIndex(t, ledger)
	if index.TotalRecords != 1 {
		t.Fatalf("TotalRecords = %d, want 1", index.TotalRecords)
	}
	assertRecoveredOpenState(t, root, 1)
}

func TestSidecarEvidencePersistenceByDigest(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	sealResult := mustSealFixtureSegment(t, ledger, fixture)
	sealID, _ := sealResult.SealEnvelopeDigest.Identity()
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, sealsDirName), sealID)
	receiptEnvelope := buildAnchorReceiptEnvelope(t, fixture, sealResult.SealEnvelopeDigest)
	receiptDigest := mustPersistReceipt(t, ledger, receiptEnvelope)
	receiptID, _ := receiptDigest.Identity()
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, receiptsDirName), receiptID)

	report := validReportFixture("segment-000001")
	reportDigest := mustPersistReport(t, ledger, report)
	reportID, _ := reportDigest.Identity()
	assertDigestSidecarExists(t, filepath.Join(root, sidecarDirName, verificationReportsDirName), reportID)
}

func TestReadinessSemantics(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	readiness, err := ledger.Readiness()
	if err != nil {
		t.Fatalf("Readiness returned error: %v", err)
	}
	if readiness.Ready {
		t.Fatal("Readiness.Ready = true, want false before verification inputs")
	}
	if readiness.VerifierMaterialAvailable {
		t.Fatal("VerifierMaterialAvailable = true, want false")
	}

	fixture := newAuditFixtureKey(t)
	request := validAdmissionRequestForLedger(t, fixture)
	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{VerifierRecords: request.VerifierRecords, EventContractCatalog: request.EventContractCatalog, SignerEvidence: request.SignerEvidence}); err != nil {
		t.Fatalf("ConfigureVerificationInputs returned error: %v", err)
	}
	if _, err := ledger.AppendAdmittedEvent(request); err != nil {
		t.Fatalf("AppendAdmittedEvent returned error: %v", err)
	}
	if _, err := ledger.BuildIndex(); err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	readiness, err = ledger.Readiness()
	if err != nil {
		t.Fatalf("Readiness(after append) returned error: %v", err)
	}
	if !readiness.Ready {
		t.Fatal("Readiness.Ready = false, want true")
	}
}

func TestAppendAdmittedEventRejectsDuplicateRecordDigestAndPreservesCounts(t *testing.T) {
	root, ledger, fixture := setupLedgerWithAdmissionFixture(t)

	before := mustReadDerivedIndex(t, root)
	if before.TotalRecords != 1 {
		t.Fatalf("index TotalRecords(before duplicate) = %d, want 1", before.TotalRecords)
	}

	duplicate := validAdmissionRequestForLedger(t, fixture)
	if _, err := ledger.AppendAdmittedEvent(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate record digest") {
		t.Fatalf("AppendAdmittedEvent(duplicate) error = %v, want duplicate record digest rejection", err)
	}

	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	if len(segment.Frames) != 1 {
		t.Fatalf("segment frame count = %d, want 1 after duplicate rejection", len(segment.Frames))
	}

	after := mustReadDerivedIndex(t, root)
	if after.TotalRecords != 1 {
		t.Fatalf("index TotalRecords(after duplicate) = %d, want 1", after.TotalRecords)
	}
	rebuilt := mustBuildIndex(t, ledger)
	if rebuilt.TotalRecords != 1 {
		t.Fatalf("rebuilt index TotalRecords = %d, want 1", rebuilt.TotalRecords)
	}

	state, err := ledger.loadState()
	if err != nil {
		t.Fatalf("loadState returned error: %v", err)
	}
	if state.LastIndexedRecordCount != 1 {
		t.Fatalf("state LastIndexedRecordCount = %d, want 1", state.LastIndexedRecordCount)
	}
	if state.OpenFrameCount != 1 {
		t.Fatalf("state OpenFrameCount = %d, want 1", state.OpenFrameCount)
	}
}

func TestAppendAdmittedEventRejectsConcurrentDuplicateRecordDigest(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	duplicate := validAdmissionRequestForLedger(t, fixture)

	const attempts = 8
	errCh := make(chan error, attempts)
	var start sync.WaitGroup
	start.Add(1)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start.Wait()
			_, err := ledger.AppendAdmittedEvent(duplicate)
			errCh <- err
		}()
	}
	start.Done()
	wg.Wait()
	close(errCh)

	duplicateErrs := 0
	for err := range errCh {
		if err == nil {
			continue
		}
		if !strings.Contains(err.Error(), "duplicate record digest") {
			t.Fatalf("AppendAdmittedEvent(concurrent duplicate) error = %v, want duplicate record digest rejection", err)
		}
		duplicateErrs++
	}
	if duplicateErrs != attempts {
		t.Fatalf("duplicate rejections = %d, want %d", duplicateErrs, attempts)
	}

	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	if len(segment.Frames) != 1 {
		t.Fatalf("segment frame count = %d, want 1 after concurrent duplicate rejection", len(segment.Frames))
	}
	state, err := ledger.loadState()
	if err != nil {
		t.Fatalf("loadState returned error: %v", err)
	}
	if state.LastIndexedRecordCount != 1 || state.OpenFrameCount != 1 {
		t.Fatalf("state after concurrent duplicate rejection = %+v, want frame/index counts of 1", state)
	}
}

func TestReadinessFailsClosedWhenVerificationInputsMalformed(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, "event-contract-catalog.json"), []byte(`{"bad":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, "verifier-records.json"), []byte(`[]`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	readiness, err := ledger.Readiness()
	if err != nil {
		t.Fatalf("Readiness returned error: %v", err)
	}
	if readiness.VerifierMaterialAvailable {
		t.Fatal("VerifierMaterialAvailable = true, want false for malformed contracts")
	}
	if readiness.Ready {
		t.Fatal("Readiness.Ready = true, want false for malformed contracts")
	}
}

func TestLatestVerificationReportRecoversFromStatePointerLoss(t *testing.T) {
	root, ledger, _ := setupLedgerWithAdmissionFixture(t)
	report := validReportFixture("segment-000001")
	digest := mustPersistReport(t, ledger, report)
	statePath := filepath.Join(root, stateFileName)
	state := ledgerState{}
	if err := readJSONFile(statePath, &state); err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	state.LastVerificationReportDigest = ""
	if err := writeCanonicalJSONFile(statePath, state); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
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
}

func TestWriteCanonicalJSONFileReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"stale":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	state := ledgerState{SchemaVersion: stateSchemaVersion, LastSealedSegmentID: "segment-000123"}
	if err := writeCanonicalJSONFile(path, state); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
	loaded := ledgerState{}
	if err := readJSONFile(path, &loaded); err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	if loaded.LastSealedSegmentID != "segment-000123" {
		t.Fatalf("LastSealedSegmentID = %q, want segment-000123", loaded.LastSealedSegmentID)
	}
}

func TestWriteCanonicalJSONFileConcurrentWritersSameTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	const writers = 24

	var start sync.WaitGroup
	start.Add(1)
	var wg sync.WaitGroup
	errCh := make(chan error, writers)

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start.Wait()
			state := ledgerState{SchemaVersion: stateSchemaVersion, LastSealedSegmentID: nextSegmentID(int64(i + 1))}
			if err := writeCanonicalJSONFile(path, state); err != nil {
				errCh <- err
			}
		}(i)
	}

	start.Done()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("writeCanonicalJSONFile returned concurrent error: %v", err)
	}

	loaded := ledgerState{}
	if err := readJSONFile(path, &loaded); err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	if loaded.SchemaVersion != stateSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", loaded.SchemaVersion, stateSchemaVersion)
	}
}

func TestReplaceFileRestoresDestinationWhenSecondRenameFails(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "state.json.tmp")
	dst := filepath.Join(dir, "state.json")
	if err := os.WriteFile(src, []byte(`{"next":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(src) returned error: %v", err)
	}
	if err := os.WriteFile(dst, []byte(`{"current":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(dst) returned error: %v", err)
	}
	originalRename := renameFile
	renameFile = func(srcPath, dstPath string) error {
		if srcPath == src && dstPath == dst {
			return fmt.Errorf("forced rename failure")
		}
		return originalRename(srcPath, dstPath)
	}
	t.Cleanup(func() {
		renameFile = originalRename
	})

	err := replaceFile(src, dst)
	if err == nil {
		t.Fatal("replaceFile expected rename failure")
	}
	b, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("ReadFile(dst) returned error: %v", readErr)
	}
	if string(b) != `{"current":true}` {
		t.Fatalf("dst contents = %q, want original contents", string(b))
	}
}

func TestReplaceFileFallbackPromotesSourceAndRemovesBackup(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "state.json.tmp")
	dst := filepath.Join(dir, "state.json")
	if err := os.WriteFile(src, []byte(`{"next":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(src) returned error: %v", err)
	}
	if err := os.WriteFile(dst, []byte(`{"current":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile(dst) returned error: %v", err)
	}
	originalRename := renameFile
	firstPromote := true
	renameFile = func(srcPath, dstPath string) error {
		if srcPath == src && dstPath == dst && firstPromote {
			firstPromote = false
			return fmt.Errorf("forced first promote failure")
		}
		return originalRename(srcPath, dstPath)
	}
	t.Cleanup(func() {
		renameFile = originalRename
	})

	if err := replaceFile(src, dst); err != nil {
		t.Fatalf("replaceFile returned error: %v", err)
	}
	b, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("ReadFile(dst) returned error: %v", readErr)
	}
	if string(b) != `{"next":true}` {
		t.Fatalf("dst contents = %q, want promoted contents", string(b))
	}
	if _, statErr := os.Stat(dst + ".bak"); !os.IsNotExist(statErr) {
		t.Fatalf("backup presence err = %v, want not exist", statErr)
	}
}

func TestOpenConcurrentCleanStartAndRestart(t *testing.T) {
	root := t.TempDir()
	const starters = 16
	assertConcurrentOpenSucceeds(t, root, starters)

	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(restart) returned error: %v", err)
	}
	state, err := reopened.loadState()
	if err != nil {
		t.Fatalf("loadState returned error: %v", err)
	}
	if state.CurrentOpenSegmentID == "" || !state.RecoveryComplete {
		t.Fatalf("unexpected persisted state after restart: %+v", state)
	}
	if state.LedgerIdentity == "" {
		t.Fatalf("ledger_identity empty after restart: %+v", state)
	}
}

func assertConcurrentOpenSucceeds(t *testing.T, root string, starters int) {
	t.Helper()
	var start sync.WaitGroup
	start.Add(1)
	var wg sync.WaitGroup
	errCh := make(chan error, starters)
	for i := 0; i < starters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start.Wait()
			if _, err := Open(root); err != nil {
				errCh <- err
			}
		}()
	}
	start.Done()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("Open(clean-start) returned error: %v", err)
	}
}

func TestLedgerIdentityGeneratedAndPersistsAcrossReopen(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	state, err := ledger.loadState()
	if err != nil {
		t.Fatalf("loadState(first) returned error: %v", err)
	}
	if state.LedgerIdentity == "" {
		t.Fatalf("ledger_identity empty on first open: %+v", state)
	}
	first := state.LedgerIdentity
	reopened, err := Open(root)
	if err != nil {
		t.Fatalf("Open(reopened) returned error: %v", err)
	}
	reloaded, err := reopened.loadState()
	if err != nil {
		t.Fatalf("loadState(reopened) returned error: %v", err)
	}
	if reloaded.LedgerIdentity != first {
		t.Fatalf("ledger_identity after reopen = %q, want %q", reloaded.LedgerIdentity, first)
	}
}

func TestConfigureVerificationInputsClearsOmittedOptionalFiles(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	fixture := newAuditFixtureKey(t)
	request := validAdmissionRequestForLedger(t, fixture)
	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{
		VerifierRecords:      request.VerifierRecords,
		EventContractCatalog: request.EventContractCatalog,
		SignerEvidence:       request.SignerEvidence,
		StoragePosture:       fullStoragePostureFixture(),
	}); err != nil {
		t.Fatalf("ConfigureVerificationInputs(initial) returned error: %v", err)
	}
	contractsDir := filepath.Join(root, "contracts")
	assertPathPresent(t, filepath.Join(contractsDir, "signer-evidence.json"), "signer-evidence.json missing after initial configure")
	assertPathPresent(t, filepath.Join(contractsDir, "storage-posture.json"), "storage-posture.json missing after initial configure")

	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{
		VerifierRecords:      request.VerifierRecords,
		EventContractCatalog: request.EventContractCatalog,
	}); err != nil {
		t.Fatalf("ConfigureVerificationInputs(update) returned error: %v", err)
	}
	assertPathMissing(t, filepath.Join(contractsDir, "signer-evidence.json"), "signer-evidence.json should be removed when omitted")
	assertPathMissing(t, filepath.Join(contractsDir, "storage-posture.json"), "storage-posture.json should be removed when omitted")
}

func TestConfigureVerificationInputsPersistsMetaAuditReceiptsWhenSealedSegmentExists(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	seal := mustSealFixtureSegment(t, ledger, fixture)
	request := validAdmissionRequestForLedger(t, fixture)
	if err := ledger.ConfigureVerificationInputs(VerificationConfiguration{VerifierRecords: request.VerifierRecords, EventContractCatalog: request.EventContractCatalog, SignerEvidence: request.SignerEvidence}); err != nil {
		t.Fatalf("ConfigureVerificationInputs returned error: %v", err)
	}
	receipts, err := ledger.ReceiptsForSealDigest(seal.SealEnvelopeDigest)
	if err != nil {
		t.Fatalf("ReceiptsForSealDigest returned error: %v", err)
	}
	if !hasReceiptKind(t, receipts, "verifier_configuration_changed") {
		t.Fatal("missing verifier_configuration_changed receipt")
	}
	if !hasReceiptKind(t, receipts, "trust_root_updated") {
		t.Fatal("missing trust_root_updated receipt")
	}
	if !hasReceiptKind(t, receipts, "evidence_import") {
		t.Fatal("missing evidence_import receipt")
	}
}

func hasReceiptKind(t *testing.T, receipts []trustpolicy.SignedObjectEnvelope, kind string) bool {
	t.Helper()
	for i := range receipts {
		payload := map[string]any{}
		if err := json.Unmarshal(receipts[i].Payload, &payload); err != nil {
			continue
		}
		if receiptKind, _ := payload["audit_receipt_kind"].(string); receiptKind == kind {
			return true
		}
	}
	return false
}

func fullStoragePostureFixture() *trustpolicy.AuditStoragePostureEvidence {
	return &trustpolicy.AuditStoragePostureEvidence{EncryptedAtRestDefault: true, EncryptedAtRestEffective: true, DevPlaintextOverrideActive: false, SurfacedToOperator: true}
}

func assertPathPresent(t *testing.T, path string, msg string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func assertPathMissing(t *testing.T, path string, msg string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s, stat err = %v", msg, err)
	}
}
