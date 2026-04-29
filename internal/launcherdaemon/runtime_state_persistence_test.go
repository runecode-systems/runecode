package launcherdaemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestRuntimeAdmissionPersistenceOverwritesExistingFile(t *testing.T) {
	cacheRoot := t.TempDir()
	image := validRuntimeImageForTests()
	record, err := launcherbackend.NewRuntimeAdmissionRecord(image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	record.AuthorityStateDigest = "sha256:" + repeatHex('a')
	record.AuthorityStateRevision = 1
	recordPath, err := runtimeAdmissionRecordPath(cacheRoot, record.DescriptorDigest)
	if err != nil {
		t.Fatalf("runtimeAdmissionRecordPath returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(recordPath), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(recordPath, []byte("{\"stale\":true}"), 0o600); err != nil {
		t.Fatalf("WriteFile(stale) returned error: %v", err)
	}

	if err := persistRuntimeAdmissionRecord(cacheRoot, record); err != nil {
		t.Fatalf("persistRuntimeAdmissionRecord returned error: %v", err)
	}

	persisted, found, err := loadRuntimeAdmissionRecord(cacheRoot, record.DescriptorDigest)
	if err != nil {
		t.Fatalf("loadRuntimeAdmissionRecord returned error: %v", err)
	}
	if !found {
		t.Fatal("expected persisted admission record")
	}
	if persisted.DescriptorDigest != record.DescriptorDigest {
		t.Fatalf("persisted descriptor digest = %q, want %q", persisted.DescriptorDigest, record.DescriptorDigest)
	}
}

func TestRuntimeVerifierAuthorityStatePersistenceOverwritesExistingFile(t *testing.T) {
	cacheRoot := t.TempDir()
	state := importedReplaceRuntimeVerifierAuthorityStateForTests(t)
	statePath := runtimeVerifierAuthorityStatePath(cacheRoot)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("{\"stale\":true}"), 0o600); err != nil {
		t.Fatalf("WriteFile(stale) returned error: %v", err)
	}

	if err := persistImportedRuntimeVerifierAuthorityState(cacheRoot, state); err != nil {
		t.Fatalf("persistImportedRuntimeVerifierAuthorityState returned error: %v", err)
	}

	persisted, found, err := loadImportedRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		t.Fatalf("loadImportedRuntimeVerifierAuthorityState returned error: %v", err)
	}
	if !found {
		t.Fatal("expected imported runtime verifier authority state")
	}
	if persisted.StateDigest != state.StateDigest {
		t.Fatalf("persisted state digest = %q, want %q", persisted.StateDigest, state.StateDigest)
	}
}

func TestRuntimeVerifierAuthorityReceiptPersistenceOverwritesExistingFile(t *testing.T) {
	cacheRoot := t.TempDir()
	receiptPath := seedStaleVerifierAuthorityReceipt(t, cacheRoot)
	receipt := runtimeVerifierAuthorityImportReceiptFixture()

	if err := persistRuntimeVerifierAuthorityImportReceipt(cacheRoot, receipt); err != nil {
		t.Fatalf("persistRuntimeVerifierAuthorityImportReceipt returned error: %v", err)
	}

	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("ReadFile(receipt) returned error: %v", err)
	}
	decoded := RuntimeVerifierAuthorityStateImportReceipt{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal(receipt) returned error: %v", err)
	}
	if decoded.ImportedStateDigest != receipt.ImportedStateDigest {
		t.Fatalf("imported state digest = %q, want %q", decoded.ImportedStateDigest, receipt.ImportedStateDigest)
	}
}

func seedStaleVerifierAuthorityReceipt(t *testing.T, cacheRoot string) string {
	t.Helper()
	receiptPath := filepath.Join(cacheRoot, runtimeVerifierAuthorityStateDirName, runtimeVerifierAuthorityReceiptFileName)
	if err := os.MkdirAll(filepath.Dir(receiptPath), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(receiptPath, []byte("{\"stale\":true}"), 0o600); err != nil {
		t.Fatalf("WriteFile(stale) returned error: %v", err)
	}
	return receiptPath
}

func runtimeVerifierAuthorityImportReceiptFixture() RuntimeVerifierAuthorityStateImportReceipt {
	return RuntimeVerifierAuthorityStateImportReceipt{
		SchemaID:             runtimeVerifierAuthorityReceiptSchemaID,
		SchemaVersion:        runtimeVerifierAuthorityReceiptSchemaVersion,
		Action:               "import",
		ChangedAt:            "2026-05-01T00:00:00Z",
		WorkRootScope:        "runtime-cache",
		ImportedStateDigest:  "sha256:" + repeatHex('b'),
		EffectiveStateDigest: "sha256:" + repeatHex('c'),
		ImportedGeneration: runtimeVerifierAuthorityGeneration{
			Revision: 2,
		},
		EffectiveGeneration: runtimeVerifierAuthorityGeneration{
			Revision: 2,
		},
	}
}
