package launcherdaemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestLoadEffectiveRuntimeVerifierAuthorityStateUsesBuiltinWhenNoImportedState(t *testing.T) {
	cacheRoot := t.TempDir()
	state, err := loadEffectiveRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		t.Fatalf("loadEffectiveRuntimeVerifierAuthorityState returned error: %v", err)
	}
	if state.SchemaID != runtimeVerifierAuthorityStateSchemaID {
		t.Fatalf("schema_id = %q", state.SchemaID)
	}
	if len(state.AuthoritiesByKind[runtimeVerifierKindImage]) == 0 {
		t.Fatal("expected builtin image verifier authority")
	}
}

func TestLoadEffectiveRuntimeVerifierAuthorityStateFailClosedOnMalformedPersistedState(t *testing.T) {
	cacheRoot := t.TempDir()
	path := runtimeVerifierAuthorityStatePath(cacheRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if _, err := loadEffectiveRuntimeVerifierAuthorityState(cacheRoot); err == nil {
		t.Fatal("expected malformed imported authority state error")
	}
}

func TestImportRuntimeVerifierAuthorityStatePersistsAndLoads(t *testing.T) {
	cacheRoot := t.TempDir()
	raw, err := json.Marshal(importedExtendRuntimeVerifierAuthorityStateForTests(t))
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := cacheRoot + "/import.json"
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile(input) returned error: %v", err)
	}
	if err := ImportRuntimeVerifierAuthorityState(cacheRoot, input); err != nil {
		t.Fatalf("ImportRuntimeVerifierAuthorityState returned error: %v", err)
	}
	effective, err := loadEffectiveRuntimeVerifierAuthorityState(cacheRoot)
	if err != nil {
		t.Fatalf("loadEffectiveRuntimeVerifierAuthorityState returned error: %v", err)
	}
	if len(effective.AuthoritiesByKind[runtimeVerifierKindImage]) < 2 {
		t.Fatalf("expected extend mode to include builtin and imported authorities, got %d", len(effective.AuthoritiesByKind[runtimeVerifierKindImage]))
	}
}

func TestImportRuntimeVerifierAuthorityStateWithReceiptPersistsReceipt(t *testing.T) {
	cacheRoot := t.TempDir()
	raw, err := json.Marshal(importedReplaceRuntimeVerifierAuthorityStateForTests(t))
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := filepath.Join(cacheRoot, "import-with-receipt.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	receipt, err := ImportRuntimeVerifierAuthorityStateWithReceipt(cacheRoot, input)
	if err != nil {
		t.Fatalf("ImportRuntimeVerifierAuthorityStateWithReceipt returned error: %v", err)
	}
	if receipt.SchemaID != runtimeVerifierAuthorityReceiptSchemaID {
		t.Fatalf("receipt schema_id = %q", receipt.SchemaID)
	}
	receiptPath := filepath.Join(cacheRoot, runtimeVerifierAuthorityStateDirName, runtimeVerifierAuthorityReceiptFileName)
	b, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("ReadFile(receipt) returned error: %v", err)
	}
	decoded := RuntimeVerifierAuthorityStateImportReceipt{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal(receipt) returned error: %v", err)
	}
	if decoded.ImportedStateDigest == "" || decoded.EffectiveStateDigest == "" {
		t.Fatal("expected non-empty digest fields in receipt")
	}
}

func importedExtendRuntimeVerifierAuthorityStateForTests(t *testing.T) runtimeVerifierAuthorityState {
	t.Helper()
	builtin := builtInRuntimeVerifierAuthorityState()
	newRecord := builtInRuntimeToolchainVerifierRecord()
	return normalizeRuntimeVerifierAuthorityState(runtimeVerifierAuthorityState{
		SchemaID:      runtimeVerifierAuthorityStateSchemaID,
		SchemaVersion: runtimeVerifierAuthorityStateSchemaVersion,
		Generation: runtimeVerifierAuthorityGeneration{
			Revision:         2,
			PreviousRevision: builtin.Generation.Revision,
			ChangedAt:        time.Now().UTC().Format(time.RFC3339),
			Reason:           "add approved verifier",
		},
		MergeMode: runtimeVerifierAuthorityMergeModeExtend,
		AuthoritiesByKind: map[string][]runtimeVerifierAuthorityEntry{
			runtimeVerifierKindImage: {{
				VerifierSetRef: mustRuntimeVerifierSetDigest([]trustpolicy.VerifierRecord{newRecord}),
				Records:        []trustpolicy.VerifierRecord{newRecord},
				Status:         runtimeVerifierAuthorityStatusActive,
				Source:         runtimeVerifierAuthoritySourceImported,
				ChangedAt:      time.Now().UTC().Format(time.RFC3339),
				Reason:         "rotation",
			}},
		},
	})
}

func importedReplaceRuntimeVerifierAuthorityStateForTests(t *testing.T) runtimeVerifierAuthorityState {
	t.Helper()
	builtin := builtInRuntimeVerifierAuthorityState()
	return normalizeRuntimeVerifierAuthorityState(runtimeVerifierAuthorityState{
		SchemaID:      runtimeVerifierAuthorityStateSchemaID,
		SchemaVersion: runtimeVerifierAuthorityStateSchemaVersion,
		Generation: runtimeVerifierAuthorityGeneration{
			Revision:         2,
			PreviousRevision: builtin.Generation.Revision,
			ChangedAt:        time.Now().UTC().Format(time.RFC3339),
			Reason:           "rotate verifier",
		},
		MergeMode: runtimeVerifierAuthorityMergeModeReplace,
		AuthoritiesByKind: map[string][]runtimeVerifierAuthorityEntry{
			runtimeVerifierKindImage: builtInRuntimeVerifierAuthorityState().AuthoritiesByKind[runtimeVerifierKindImage],
		},
	})
}

func TestImportRuntimeVerifierAuthorityStateRejectsDuplicateVerifierSetRefs(t *testing.T) {
	cacheRoot := t.TempDir()
	builtin := builtInRuntimeVerifierAuthorityState()
	records := []trustpolicy.VerifierRecord{builtInRuntimeImageVerifierRecord()}
	state := runtimeVerifierAuthorityState{
		SchemaID:      runtimeVerifierAuthorityStateSchemaID,
		SchemaVersion: runtimeVerifierAuthorityStateSchemaVersion,
		Generation: runtimeVerifierAuthorityGeneration{
			Revision:         builtin.Generation.Revision + 1,
			PreviousRevision: builtin.Generation.Revision,
			ChangedAt:        time.Now().UTC().Format(time.RFC3339),
			Reason:           "duplicate entry regression",
		},
		MergeMode: runtimeVerifierAuthorityMergeModeReplace,
		AuthoritiesByKind: map[string][]runtimeVerifierAuthorityEntry{
			runtimeVerifierKindImage: {
				{VerifierSetRef: mustRuntimeVerifierSetDigest(records), Records: records, Status: runtimeVerifierAuthorityStatusActive, Source: runtimeVerifierAuthoritySourceImported, ChangedAt: time.Now().UTC().Format(time.RFC3339)},
				{VerifierSetRef: mustRuntimeVerifierSetDigest(records), Records: records, Status: runtimeVerifierAuthorityStatusRevoked, Source: runtimeVerifierAuthoritySourceImported, ChangedAt: time.Now().UTC().Format(time.RFC3339)},
			},
		},
	}
	state = normalizeRuntimeVerifierAuthorityState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := filepath.Join(cacheRoot, "duplicate-authority-state.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := ImportRuntimeVerifierAuthorityState(cacheRoot, input); err == nil {
		t.Fatal("expected duplicate verifier_set_ref rejection")
	}
}

func TestImportRuntimeVerifierAuthorityStateRejectsNonMonotonicRevision(t *testing.T) {
	cacheRoot := t.TempDir()
	builtin := builtInRuntimeVerifierAuthorityState()
	state := runtimeVerifierAuthorityState{
		SchemaID:      runtimeVerifierAuthorityStateSchemaID,
		SchemaVersion: runtimeVerifierAuthorityStateSchemaVersion,
		Generation: runtimeVerifierAuthorityGeneration{
			Revision:         builtin.Generation.Revision,
			PreviousRevision: builtin.Generation.Revision,
			ChangedAt:        time.Now().UTC().Format(time.RFC3339),
			Reason:           "rollback attempt",
		},
		MergeMode: runtimeVerifierAuthorityMergeModeReplace,
		AuthoritiesByKind: map[string][]runtimeVerifierAuthorityEntry{
			runtimeVerifierKindImage: builtInRuntimeVerifierAuthorityState().AuthoritiesByKind[runtimeVerifierKindImage],
		},
	}
	state = normalizeRuntimeVerifierAuthorityState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := filepath.Join(cacheRoot, "rollback-authority-state.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := ImportRuntimeVerifierAuthorityState(cacheRoot, input); err == nil {
		t.Fatal("expected non-monotonic revision rejection")
	}
}

func TestImportRuntimeVerifierAuthorityStateRejectsZeroRevision(t *testing.T) {
	cacheRoot := t.TempDir()
	state := runtimeVerifierAuthorityState{
		SchemaID:      runtimeVerifierAuthorityStateSchemaID,
		SchemaVersion: runtimeVerifierAuthorityStateSchemaVersion,
		Generation: runtimeVerifierAuthorityGeneration{
			Revision:  0,
			ChangedAt: time.Now().UTC().Format(time.RFC3339),
			Reason:    "invalid revision",
		},
		MergeMode: runtimeVerifierAuthorityMergeModeReplace,
		AuthoritiesByKind: map[string][]runtimeVerifierAuthorityEntry{
			runtimeVerifierKindImage: builtInRuntimeVerifierAuthorityState().AuthoritiesByKind[runtimeVerifierKindImage],
		},
	}
	state = normalizeRuntimeVerifierAuthorityState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := filepath.Join(cacheRoot, "zero-revision-authority-state.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := ImportRuntimeVerifierAuthorityState(cacheRoot, input); err == nil {
		t.Fatal("expected zero revision rejection")
	}
}

func TestImportRuntimeVerifierAuthorityStateRejectsDigestMismatch(t *testing.T) {
	cacheRoot := t.TempDir()
	state := importedExtendRuntimeVerifierAuthorityStateForTests(t)
	state.StateDigest = "sha256:" + repeatHex('f')
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := filepath.Join(cacheRoot, "digest-mismatch-authority-state.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := ImportRuntimeVerifierAuthorityState(cacheRoot, input); err == nil {
		t.Fatal("expected digest mismatch rejection")
	}
}

func TestImportRuntimeVerifierAuthorityStateWithReceiptReturnsErrorWhenReceiptPersistenceFails(t *testing.T) {
	cacheRoot := t.TempDir()
	raw, err := json.Marshal(importedReplaceRuntimeVerifierAuthorityStateForTests(t))
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := filepath.Join(cacheRoot, "import-with-receipt-conflict.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	receiptPath := filepath.Join(cacheRoot, runtimeVerifierAuthorityStateDirName, runtimeVerifierAuthorityReceiptFileName)
	if err := os.MkdirAll(receiptPath, 0o700); err != nil {
		t.Fatalf("MkdirAll(receiptPath) returned error: %v", err)
	}
	receipt, err := ImportRuntimeVerifierAuthorityStateWithReceipt(cacheRoot, input)
	if err == nil {
		t.Fatal("expected receipt persistence error")
	}
	if receipt.ImportedStateDigest == "" || receipt.EffectiveStateDigest == "" {
		t.Fatal("expected non-empty receipt digests when receipt persistence fails")
	}
	if _, found, err := loadImportedRuntimeVerifierAuthorityState(cacheRoot); err != nil || !found {
		t.Fatalf("expected imported state persisted despite receipt write failure: found=%v err=%v", found, err)
	}
}

func TestImportRuntimeVerifierAuthorityStateWithReceiptAllowsIdempotentRetryAfterReceiptFailure(t *testing.T) {
	cacheRoot := t.TempDir()
	input := writeRuntimeVerifierAuthorityStateFixture(t, cacheRoot, "retry-authority-state.json", importedReplaceRuntimeVerifierAuthorityStateForTests(t))
	receiptPath := filepath.Join(cacheRoot, runtimeVerifierAuthorityStateDirName, runtimeVerifierAuthorityReceiptFileName)
	if err := os.MkdirAll(receiptPath, 0o700); err != nil {
		t.Fatalf("MkdirAll(receiptPath) returned error: %v", err)
	}
	if _, err := ImportRuntimeVerifierAuthorityStateWithReceipt(cacheRoot, input); err == nil {
		t.Fatal("expected receipt persistence error on first import")
	}
	if err := os.RemoveAll(receiptPath); err != nil {
		t.Fatalf("RemoveAll(receiptPath) returned error: %v", err)
	}
	receipt, err := ImportRuntimeVerifierAuthorityStateWithReceipt(cacheRoot, input)
	if err != nil {
		t.Fatalf("ImportRuntimeVerifierAuthorityStateWithReceipt retry returned error: %v", err)
	}
	if receipt.ImportedStateDigest == "" || receipt.EffectiveStateDigest == "" {
		t.Fatal("expected non-empty receipt digests after idempotent retry")
	}
}

func writeRuntimeVerifierAuthorityStateFixture(t *testing.T, dir string, name string, state runtimeVerifierAuthorityState) string {
	t.Helper()
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return path
}

func TestAuthorizeRuntimeVerifierSetRejectsRevokedEntry(t *testing.T) {
	cacheRoot := t.TempDir()
	records := []trustpolicy.VerifierRecord{builtInRuntimeImageVerifierRecord()}
	importRevokedRuntimeVerifierAuthorityState(t, cacheRoot, records)
	err := authorizeRuntimeVerifierSet(cacheRoot, runtimeVerifierKindImage, mustRuntimeVerifierSetDigest(records), records)
	if err == nil || err.Error() != "runtime verifier set is revoked" {
		t.Fatalf("expected revoked error, got %v", err)
	}
}

func importRevokedRuntimeVerifierAuthorityState(t *testing.T, cacheRoot string, records []trustpolicy.VerifierRecord) {
	t.Helper()
	revokedState := runtimeVerifierAuthorityState{
		SchemaID:      runtimeVerifierAuthorityStateSchemaID,
		SchemaVersion: runtimeVerifierAuthorityStateSchemaVersion,
		Generation: runtimeVerifierAuthorityGeneration{
			Revision:         2,
			PreviousRevision: builtInRuntimeVerifierAuthorityState().Generation.Revision,
			ChangedAt:        time.Now().UTC().Format(time.RFC3339),
			Reason:           "revoke compromised key",
		},
		MergeMode: runtimeVerifierAuthorityMergeModeReplace,
		AuthoritiesByKind: map[string][]runtimeVerifierAuthorityEntry{
			runtimeVerifierKindImage: {
				{
					VerifierSetRef: mustRuntimeVerifierSetDigest(records),
					Records:        records,
					Status:         runtimeVerifierAuthorityStatusRevoked,
					Source:         runtimeVerifierAuthoritySourceImported,
					ChangedAt:      time.Now().UTC().Format(time.RFC3339),
					Reason:         "revoked",
				},
			},
		},
	}
	revokedState = normalizeRuntimeVerifierAuthorityState(revokedState)
	raw, err := json.Marshal(revokedState)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	input := filepath.Join(cacheRoot, "revoked-authority-state.json")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := ImportRuntimeVerifierAuthorityState(cacheRoot, input); err != nil {
		t.Fatalf("ImportRuntimeVerifierAuthorityState returned error: %v", err)
	}
}
