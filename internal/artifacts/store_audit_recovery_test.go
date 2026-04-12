package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuotaFailuresPreserveQuotaErrorWhenAuditAlsoFails(t *testing.T) {
	store := newTestStore(t)
	policy := store.Policy()
	policy.PerRoleQuota["workspace"] = Quota{MaxArtifactCount: 1, MaxTotalBytes: 1, MaxSingleArtifactSize: 1}
	if err := store.SetPolicy(policy); err != nil {
		t.Fatalf("SetPolicy error: %v", err)
	}
	if _, err := store.Put(PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("8"), CreatedByRole: "workspace"}); err != nil {
		t.Fatalf("seed Put error: %v", err)
	}
	setBrokenAuditPath(t, store)
	_, err := store.Put(PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("8"), CreatedByRole: "workspace"})
	if err == nil {
		t.Fatal("Put expected joined quota and audit error")
	}
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("Put error = %v, want joined error containing %v", err, ErrQuotaExceeded)
	}
}

func TestAuditFailureIsSurfaced(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("excerpt"), ContentType: "text/plain", DataClass: DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: testDigest("e"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	setBrokenAuditPath(t, store)
	err = store.CheckFlow(FlowCheckRequest{ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: DataClassUnapprovedFileExcerpts, Digest: ref.Digest, IsEgress: true})
	if err == nil {
		t.Fatal("CheckFlow expected audit write error")
	}
	if store.state.LastAuditSequence != 1 {
		t.Fatalf("LastAuditSequence after failed audit append = %d, want 1", store.state.LastAuditSequence)
	}
}

func TestAuditSequencePersistsForBlockedFlow(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("excerpt"), ContentType: "text/plain", DataClass: DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	err = store.CheckFlow(FlowCheckRequest{ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: DataClassUnapprovedFileExcerpts, Digest: ref.Digest, IsEgress: true})
	if err != ErrUnapprovedEgressDenied {
		t.Fatalf("CheckFlow error = %v, want %v", err, ErrUnapprovedEgressDenied)
	}
	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore reload error: %v", err)
	}
	if reloaded.state.LastAuditSequence <= 1 {
		t.Fatalf("reloaded LastAuditSequence = %d, want > 1 after blocked-flow audit", reloaded.state.LastAuditSequence)
	}
}

func TestLoadStateRecoversAuditSequenceWhenStateSaveLagged(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Put(PutRequest{Payload: []byte("seed"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"}); err != nil {
		t.Fatalf("seed Put error: %v", err)
	}
	store.storeIO.statePath = filepath.Join(t.TempDir(), "state-dir")
	if err := os.MkdirAll(store.storeIO.statePath, 0o755); err != nil {
		t.Fatalf("mkdir state dir error: %v", err)
	}
	if _, err := store.Put(PutRequest{Payload: []byte("second"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("2"), CreatedByRole: "workspace"}); err == nil {
		t.Fatal("Put expected state save failure after audit append")
	} else {
		assertPathErrorTarget(t, err, store.storeIO.statePath)
	}
	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore reload error: %v", err)
	}
	events, err := reloaded.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("audit events len = %d, want 2 after save lag", len(events))
	}
	last := events[len(events)-1].Seq
	if reloaded.state.LastAuditSequence != last {
		t.Fatalf("reloaded LastAuditSequence = %d, want %d", reloaded.state.LastAuditSequence, last)
	}
	if len(reloaded.List()) != 1 {
		t.Fatalf("reloaded artifact count = %d, want 1 after rollback", len(reloaded.List()))
	}
}

func TestPutRollsBackArtifactStateAndBlobWhenAuditAppendFails(t *testing.T) {
	store := newTestStore(t)
	store.state.Runs["run-rollback"] = "retained"
	originalAuditPath := setBrokenAuditPath(t, store)
	_, err := store.Put(PutRequest{Payload: []byte("rollback"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("4"), CreatedByRole: "workspace", RunID: "run-rollback"})
	if err == nil {
		t.Fatal("Put expected audit append failure")
	}
	assertArtifactRollbackInMemory(t, store)
	assertArtifactRollbackRunStatus(t, store, "run-rollback", "retained")
	assertArtifactRollbackBlobState(t, store)
	assertArtifactRollbackDigestLookup(t, store, testDigest("4"))
	store.storeIO.auditPath = originalAuditPath
	assertArtifactRollbackPersistedState(t, store)
}

func TestPutRollbackPreservesPreexistingBlobOnAuditFailure(t *testing.T) {
	store := newTestStore(t)
	payload := []byte("preexisting blob")
	digest := DigestBytes(payload)
	blobPath := store.storeIO.blobPath(digest)
	if err := os.WriteFile(blobPath, payload, 0o600); err != nil {
		t.Fatalf("WriteFile preexisting blob returned error: %v", err)
	}
	setBrokenAuditPath(t, store)
	_, err := store.Put(PutRequest{Payload: payload, ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("6"), CreatedByRole: "workspace"})
	if err == nil {
		t.Fatal("Put expected audit append failure")
	}
	if _, statErr := os.Stat(blobPath); statErr != nil {
		t.Fatalf("Stat preexisting blob returned error: %v", statErr)
	}
	if len(store.state.Artifacts) != 0 {
		t.Fatalf("artifacts len = %d, want 0 after rollback", len(store.state.Artifacts))
	}
}

func TestPutRejectsDigestCollisionWhenBlobContentMismatches(t *testing.T) {
	store := newTestStore(t)
	canonicalPayload := []byte("canonical artifact")
	digest := DigestBytes(canonicalPayload)
	blobPath := store.storeIO.blobPath(digest)
	if err := os.WriteFile(blobPath, []byte("tampered existing payload"), 0o600); err != nil {
		t.Fatalf("WriteFile tampered blob returned error: %v", err)
	}
	_, err := store.Put(PutRequest{Payload: canonicalPayload, ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("7"), CreatedByRole: "workspace"})
	if err != ErrInvalidDigest {
		t.Fatalf("Put error = %v, want %v", err, ErrInvalidDigest)
	}
	if len(store.state.Artifacts) != 0 {
		t.Fatalf("artifacts len = %d, want 0 after digest mismatch", len(store.state.Artifacts))
	}
}

func TestLoadStateRejectsRecoveredArtifactWhenBlobDigestMismatches(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte("seed"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("5"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if err := os.WriteFile(store.storeIO.blobPath(ref.Digest), []byte("tampered"), 0o600); err != nil {
		t.Fatalf("WriteFile tampered blob returned error: %v", err)
	}
	store.state.Artifacts = map[string]ArtifactRecord{}
	if err := store.saveStateLocked(); err != nil {
		t.Fatalf("saveStateLocked returned error: %v", err)
	}
	_, err = NewStore(store.rootDir)
	if err == nil {
		t.Fatal("NewStore expected digest mismatch error during recovery")
	}
	if !strings.Contains(err.Error(), "artifact blob digest mismatch") {
		t.Fatalf("NewStore error = %v, want artifact blob digest mismatch", err)
	}
}

func setBrokenAuditPath(t *testing.T, store *Store) string {
	t.Helper()
	originalAuditPath := store.storeIO.auditPath
	badPath := filepath.Join(t.TempDir(), "audit-dir")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("mkdir audit dir error: %v", err)
	}
	store.storeIO.auditPath = badPath
	return originalAuditPath
}

func assertArtifactRollbackInMemory(t *testing.T, store *Store) {
	t.Helper()
	if len(store.state.Artifacts) != 0 {
		t.Fatalf("artifacts len = %d, want 0 after rollback", len(store.state.Artifacts))
	}
	if store.state.LastAuditSequence != 0 {
		t.Fatalf("LastAuditSequence = %d, want 0 after rollback", store.state.LastAuditSequence)
	}
}

func assertArtifactRollbackRunStatus(t *testing.T, store *Store, runID, want string) {
	t.Helper()
	if got := store.state.Runs[runID]; got != want {
		t.Fatalf("%s status = %q, want %q after rollback", runID, got, want)
	}
}

func assertArtifactRollbackBlobState(t *testing.T, store *Store) {
	t.Helper()
	entries, readErr := os.ReadDir(store.storeIO.blobDir)
	if readErr != nil {
		t.Fatalf("ReadDir blobs returned error: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("blob entries len = %d, want 0 after rollback", len(entries))
	}
}

func assertArtifactRollbackDigestLookup(t *testing.T, store *Store, digest string) {
	t.Helper()
	if _, getErr := store.Get(digest); getErr != ErrArtifactNotFound {
		t.Fatalf("Get rolled-back digest error = %v, want %v", getErr, ErrArtifactNotFound)
	}
}

func assertArtifactRollbackPersistedState(t *testing.T, store *Store) {
	t.Helper()
	events, readErr := store.ReadAuditEvents()
	if readErr != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", readErr)
	}
	if len(events) != 0 {
		t.Fatalf("audit events len = %d, want 0 after rollback", len(events))
	}
	reloaded, reloadErr := NewStore(store.rootDir)
	if reloadErr != nil {
		t.Fatalf("NewStore reload error: %v", reloadErr)
	}
	if len(reloaded.List()) != 0 {
		t.Fatalf("reloaded artifact count = %d, want 0 after rollback", len(reloaded.List()))
	}
}

func assertPathErrorTarget(t *testing.T, err error, path string) {
	t.Helper()
	pathErr := &os.PathError{}
	if !errors.As(err, &pathErr) {
		t.Fatalf("error = %v, want wrapped os.PathError", err)
	}
	if pathErr.Path != path {
		t.Fatalf("PathError path = %q, want %q", pathErr.Path, path)
	}
}
