package artifacts

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPutGetHeadAndList(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{
		Payload:               []byte("hello"),
		ContentType:           "text/plain",
		DataClass:             DataClassSpecText,
		ProvenanceReceiptHash: testDigest("1"),
		CreatedByRole:         "workspace",
		RunID:                 "run-a",
		StepID:                "step-a",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	record, err := store.Head(ref.Digest)
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}
	if record.Reference.DataClass != DataClassSpecText {
		t.Fatalf("Head data class = %q, want %q", record.Reference.DataClass, DataClassSpecText)
	}
	r, err := store.Get(ref.Digest)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	b, readErr := ioReadAllAndClose(r)
	if readErr != nil {
		t.Fatalf("read payload error: %v", readErr)
	}
	if string(b) != "hello" {
		t.Fatalf("Get payload = %q, want hello", string(b))
	}
	if len(store.List()) != 1 {
		t.Fatalf("List count = %d, want 1", len(store.List()))
	}
}

func TestCanonicalJSONDigestDeterministic(t *testing.T) {
	store := newTestStore(t)
	ref1, err := store.Put(PutRequest{
		Payload:               []byte(`{"b":2,"a":1}`),
		ContentType:           "application/json",
		DataClass:             DataClassSpecText,
		ProvenanceReceiptHash: testDigest("2"),
		CreatedByRole:         "workspace",
	})
	if err != nil {
		t.Fatalf("first Put returned error: %v", err)
	}
	ref2, err := store.Put(PutRequest{
		Payload:               []byte(`{"a":1,"b":2}`),
		ContentType:           "application/json",
		DataClass:             DataClassSpecText,
		ProvenanceReceiptHash: testDigest("2"),
		CreatedByRole:         "workspace",
	})
	if err != nil {
		t.Fatalf("second Put returned error: %v", err)
	}
	if ref1.Digest != ref2.Digest {
		t.Fatalf("digests differ: %s vs %s", ref1.Digest, ref2.Digest)
	}
}

func TestDataClassMutationDenied(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Put(PutRequest{
		Payload:               []byte("same-bytes"),
		ContentType:           "text/plain",
		DataClass:             DataClassSpecText,
		ProvenanceReceiptHash: testDigest("3"),
		CreatedByRole:         "workspace",
	})
	if err != nil {
		t.Fatalf("first Put returned error: %v", err)
	}
	_, err = store.Put(PutRequest{
		Payload:               []byte("same-bytes"),
		ContentType:           "text/plain",
		DataClass:             DataClassDiffs,
		ProvenanceReceiptHash: testDigest("3"),
		CreatedByRole:         "workspace",
	})
	if err != ErrDataClassMutationDenied {
		t.Fatalf("Put error = %v, want %v", err, ErrDataClassMutationDenied)
	}
}

func TestPutCanonicalizesUntrustedCreatedByRole(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{
		Payload:               []byte("payload"),
		ContentType:           "text/plain",
		DataClass:             DataClassSpecText,
		ProvenanceReceiptHash: testDigest("a"),
		CreatedByRole:         " admin ",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	record, err := store.Head(ref.Digest)
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}
	if record.CreatedByRole != "untrusted_client" {
		t.Fatalf("created_by_role = %q, want untrusted_client", record.CreatedByRole)
	}
}

func TestFlowChecksFailClosedAndEgressRules(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{
		Payload:               []byte("excerpt"),
		ContentType:           "text/plain",
		DataClass:             DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: testDigest("4"),
		CreatedByRole:         "workspace",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	err = store.CheckFlow(FlowCheckRequest{ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: DataClassUnapprovedFileExcerpts, Digest: ref.Digest, IsEgress: true})
	if err != ErrUnapprovedEgressDenied {
		t.Fatalf("CheckFlow error = %v, want %v", err, ErrUnapprovedEgressDenied)
	}

	err = store.CheckFlow(FlowCheckRequest{ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: DataClassApprovedFileExcerpts, Digest: ref.Digest, IsEgress: true, ManifestOptIn: false})
	if err != ErrFlowDenied {
		t.Fatalf("CheckFlow data class mismatch error = %v, want %v", err, ErrFlowDenied)
	}

	err = store.CheckFlow(FlowCheckRequest{ProducerRole: "workspace", ConsumerRole: "unknown", DataClass: DataClassSpecText, Digest: ref.Digest})
	if err != ErrFlowDenied {
		t.Fatalf("CheckFlow unknown lane error = %v, want %v", err, ErrFlowDenied)
	}

	err = store.CheckFlow(FlowCheckRequest{ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: DataClassSpecText, Digest: testDigest("f")})
	if err != ErrArtifactNotFound {
		t.Fatalf("CheckFlow unknown digest error = %v, want %v", err, ErrArtifactNotFound)
	}
}

func TestGetForFlowEnforcesProducerRoleAndManifestOptIn(t *testing.T) {
	store, unapproved := setupPromotionSourceForTests(t)
	request := ArtifactReadRequest{
		Digest:       unapproved.Digest,
		ProducerRole: "workspace",
		ConsumerRole: "model_gateway",
		DataClass:    DataClassUnapprovedFileExcerpts,
		IsEgress:     true,
	}
	_, _, err := store.GetForFlow(request)
	if err != ErrUnapprovedEgressDenied {
		t.Fatalf("GetForFlow unapproved egress error = %v, want %v", err, ErrUnapprovedEgressDenied)
	}

	approved := promoteApprovedExcerptForTests(t, store, unapproved.Digest, "human")
	_, _, err = store.GetForFlow(ArtifactReadRequest{
		Digest:       approved.Digest,
		ProducerRole: "workspace",
		ConsumerRole: "model_gateway",
		DataClass:    DataClassApprovedFileExcerpts,
		IsEgress:     true,
	})
	if err != ErrApprovedEgressRequiresManifest {
		t.Fatalf("GetForFlow approved no-opt-in error = %v, want %v", err, ErrApprovedEgressRequiresManifest)
	}

	r, rec, err := store.GetForFlow(ArtifactReadRequest{
		Digest:        approved.Digest,
		ProducerRole:  "workspace",
		ConsumerRole:  "model_gateway",
		DataClass:     DataClassApprovedFileExcerpts,
		IsEgress:      true,
		ManifestOptIn: true,
	})
	if err != nil {
		t.Fatalf("GetForFlow approved with opt-in error: %v", err)
	}
	b, readErr := ioReadAllAndClose(r)
	if readErr != nil {
		t.Fatalf("GetForFlow read error: %v", readErr)
	}
	if string(b) != "approved:\nsensitive excerpt" {
		t.Fatalf("GetForFlow payload = %q, want approved payload", string(b))
	}
	if rec.Reference.Digest != approved.Digest {
		t.Fatalf("GetForFlow record digest = %q, want %q", rec.Reference.Digest, approved.Digest)
	}

	_, _, err = store.GetForFlow(ArtifactReadRequest{
		Digest:        approved.Digest,
		ProducerRole:  "auditd",
		ConsumerRole:  "model_gateway",
		DataClass:     DataClassApprovedFileExcerpts,
		IsEgress:      true,
		ManifestOptIn: true,
	})
	if err != ErrFlowProducerRoleMismatch {
		t.Fatalf("GetForFlow mismatched producer error = %v, want %v", err, ErrFlowProducerRoleMismatch)
	}
}

func TestPromotionRequiresApprovalAndMintsNewReference(t *testing.T) {
	store, unapproved := setupPromotionSourceForTests(t)
	assertPromotionRequiresApprover(t, store, unapproved.Digest)
	approved := promoteApprovedExcerptForTests(t, store, unapproved.Digest, "human-1")
	if approved.Digest == unapproved.Digest {
		t.Fatalf("approved digest must differ from unapproved digest")
	}
	if approved.DataClass != DataClassApprovedFileExcerpts {
		t.Fatalf("approved data class = %q", approved.DataClass)
	}
	oldRecord, _ := store.Head(unapproved.Digest)
	if oldRecord.Reference.DataClass != DataClassUnapprovedFileExcerpts {
		t.Fatalf("source artifact mutated: %q", oldRecord.Reference.DataClass)
	}
}

func TestPromotionRateLimitAndBulkGate(t *testing.T) {
	store := newTestStore(t)
	policy := store.Policy()
	policy.MaxPromotionRequestsPerMinute = 1
	if err := store.SetPolicy(policy); err != nil {
		t.Fatalf("SetPolicy error: %v", err)
	}

	first, err := store.Put(PutRequest{Payload: []byte("1"), ContentType: "text/plain", DataClass: DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: testDigest("6"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put first error: %v", err)
	}
	_, err = promoteApprovedExcerptWithFlagsForTests(t, store, first.Digest, "human", true, false)
	if err != ErrApprovalBulkConfirmationNeeded {
		t.Fatalf("bulk promotion error = %v, want %v", err, ErrApprovalBulkConfirmationNeeded)
	}
	_, err = promoteApprovedExcerptWithFlagsForTests(t, store, first.Digest, "human", true, true)
	if err != nil {
		t.Fatalf("bulk promotion confirmed error: %v", err)
	}
	second, err := store.Put(PutRequest{Payload: []byte("2"), ContentType: "text/plain", DataClass: DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: testDigest("7"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put second error: %v", err)
	}
	_, err = promoteApprovedExcerptWithFlagsForTests(t, store, second.Digest, "human", false, false)
	if err != ErrPromotionRateLimited {
		t.Fatalf("second promotion error = %v, want %v", err, ErrPromotionRateLimited)
	}
}

func TestQuotasEnforcedAndAudited(t *testing.T) {
	store := newTestStore(t)
	policy := store.Policy()
	policy.PerRoleQuota["workspace"] = Quota{MaxArtifactCount: 1, MaxTotalBytes: 5, MaxSingleArtifactSize: 5}
	if err := store.SetPolicy(policy); err != nil {
		t.Fatalf("SetPolicy error: %v", err)
	}
	_, err := store.Put(PutRequest{Payload: []byte("12345"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("8"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("first Put error: %v", err)
	}
	_, err = store.Put(PutRequest{Payload: []byte("x"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("8"), CreatedByRole: "workspace"})
	if err != ErrQuotaExceeded {
		t.Fatalf("second Put error = %v, want %v", err, ErrQuotaExceeded)
	}
	audit, err := store.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents error: %v", err)
	}
	if !containsAuditType(audit, "artifact_quota_violation") {
		t.Fatalf("expected artifact_quota_violation in audit")
	}
}

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
	badPath := filepath.Join(t.TempDir(), "audit-dir")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("mkdir audit dir error: %v", err)
	}
	store.storeIO.auditPath = badPath
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
	badPath := filepath.Join(t.TempDir(), "audit-dir")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("mkdir audit dir error: %v", err)
	}
	store.storeIO.auditPath = badPath
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
	}
	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore reload error: %v", err)
	}
	events, err := reloaded.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected audit events after simulated state save failure")
	}
	last := events[len(events)-1].Seq
	if reloaded.state.LastAuditSequence != last {
		t.Fatalf("reloaded LastAuditSequence = %d, want %d", reloaded.state.LastAuditSequence, last)
	}
}

func TestStateAndAuditFilesArePrivateByDefault(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Put(PutRequest{Payload: []byte("hello"), ContentType: "text/plain", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"}); err != nil {
		t.Fatalf("Put error: %v", err)
	}
	assertMode(t, store.storeIO.statePath, 0o600)
	assertMode(t, store.storeIO.auditPath, 0o600)
}

func TestCanonicalJSONSupportsFullJCSNumberAndUnicodeBehavior(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.Put(PutRequest{Payload: []byte(`{"😀":"value","b":1e2,"a":-0,"c":1.25}`), ContentType: "application/json", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	r, err := store.Get(ref.Digest)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	b, err := ioReadAllAndClose(r)
	if err != nil {
		t.Fatalf("read payload error: %v", err)
	}
	want := `{"a":0,"b":100,"c":1.25,"😀":"value"}`
	if string(b) != want {
		t.Fatalf("canonical payload = %q, want %q", string(b), want)
	}
}

func TestCanonicalJSONRejectsTopLevelScalarRoots(t *testing.T) {
	store := newTestStore(t)
	for _, payload := range []string{"1", `"text"`, "true", "null"} {
		_, err := store.Put(PutRequest{Payload: []byte(payload), ContentType: "application/json", DataClass: DataClassSpecText, ProvenanceReceiptHash: testDigest("1"), CreatedByRole: "workspace"})
		if err == nil {
			t.Fatalf("Put(%q) expected canonicalization error", payload)
		}
		if !strings.Contains(err.Error(), "top-level JSON value must be an object or array") {
			t.Fatalf("Put(%q) error = %v, want object-or-array root error", payload, err)
		}
	}
}

func TestReservedDataClassesFailClosedByDefault(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Put(PutRequest{Payload: []byte("web"), ContentType: "text/plain", DataClass: DataClassWebQuery, ProvenanceReceiptHash: testDigest("b"), CreatedByRole: "workspace"})
	if err != ErrReservedDataClassDisabled {
		t.Fatalf("reserved class Put error = %v, want %v", err, ErrReservedDataClassDisabled)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv(backupHMACKeyEnv, "test-backup-key")
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	store.nowFn = func() time.Time { return time.Now().UTC() }
	return store
}

func testDigest(seed string) string {
	base := strings.Repeat(seed, 64)
	if len(base) > 64 {
		base = base[:64]
	}
	for len(base) < 64 {
		base += "0"
	}
	return "sha256:" + base
}

func containsAuditType(events []AuditEvent, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func ioReadAllAndClose(r io.ReadCloser) ([]byte, error) {
	b, err := io.ReadAll(r)
	_ = r.Close()
	return b, err
}

func copyBlobsToStore(t *testing.T, dst *Store, records []ArtifactRecord) {
	t.Helper()
	for _, rec := range records {
		copyBlobFile(t, rec.BlobPath, dst.storeIO.blobPath(rec.Reference.Digest))
	}
}

func copyBlobFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read blob %s error: %v", src, err)
	}
	if err := os.WriteFile(dst, b, 0o600); err != nil {
		t.Fatalf("write blob %s error: %v", dst, err)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not reliable on Windows")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error: %v", path, err)
	}
	got := info.Mode().Perm()
	if got != want {
		t.Fatalf("mode for %s = %#o, want %#o", path, got, want)
	}
}
