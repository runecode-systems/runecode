package auditd

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestOpenSegmentAllowsTrailingPartialFrameBytes(t *testing.T) {
	root, ledger, _ := setupLedgerWithAdmissionFixture(t)
	segmentPath := filepath.Join(root, segmentsDirName, "segment-000001.json")
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := readJSONFile(segmentPath, &segment); err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	segment.TrailingPartialFrameBytes = 7
	if err := writeCanonicalJSONFile(segmentPath, segment); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
	loaded, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	if loaded.TrailingPartialFrameBytes != 7 {
		t.Fatalf("TrailingPartialFrameBytes = %d, want 7", loaded.TrailingPartialFrameBytes)
	}
}

func TestEnsureLayoutCreatesOwnerOnlyDirectories(t *testing.T) {
	skipIfDirectoryModeAssertionsUnavailable(t)
	root := t.TempDir()
	if _, err := Open(root); err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	assertOwnerOnlyMode(t, ledgerLayoutPaths(root))
}

func TestEnsureLayoutNormalizesPermissionsForExistingDirectories(t *testing.T) {
	skipIfDirectoryModeAssertionsUnavailable(t)
	root := t.TempDir()
	createLegacyLayoutFixture(t, ledgerLayoutPaths(root))
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if err := ledger.ensureLayout(); err != nil {
		t.Fatalf("ensureLayout returned error: %v", err)
	}
	assertOwnerOnlyMode(t, ledgerLayoutPaths(root))
}

func ledgerLayoutPaths(root string) []string {
	return []string{
		filepath.Join(root, segmentsDirName),
		filepath.Join(root, sidecarDirName),
		filepath.Join(root, sidecarDirName, sealsDirName),
		filepath.Join(root, sidecarDirName, receiptsDirName),
		filepath.Join(root, sidecarDirName, verificationReportsDirName),
		filepath.Join(root, indexDirName),
	}
}

func skipIfDirectoryModeAssertionsUnavailable(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("permission bits differ on windows")
	}
	umask, err := currentUmask()
	if err != nil {
		t.Fatalf("currentUmask returned error: %v", err)
	}
	if umask != 0 {
		t.Skipf("umask %03o masks owner-only directory mode assertions", umask)
	}
}

func createLegacyLayoutFixture(t *testing.T, paths []string) {
	t.Helper()
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) returned error: %v", path, err)
		}
	}
}

func assertOwnerOnlyMode(t *testing.T, paths []string) {
	t.Helper()
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) returned error: %v", path, err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("%s mode = %o, want 700", path, got)
		}
	}
}

func currentUmask() (int, error) {
	value, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, err
	}
	const prefix = "Umask:\t"
	for _, line := range strings.Split(string(value), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		parsed, err := strconv.ParseInt(strings.TrimPrefix(line, prefix), 8, 32)
		if err != nil {
			return 0, err
		}
		return int(parsed), nil
	}
	return 0, os.ErrNotExist
}

func TestLatestVerificationSummaryAndViewsFailsClosedOnFrameDigestMismatch(t *testing.T) {
	root, ledger, _ := setupLedgerWithAdmissionFixture(t)
	report := validReportFixture("segment-000001")
	_ = mustPersistReport(t, ledger, report)

	segmentPath := filepath.Join(root, segmentsDirName, "segment-000001.json")
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := readJSONFile(segmentPath, &segment); err != nil {
		t.Fatalf("readJSONFile returned error: %v", err)
	}
	segment.Frames[0].RecordDigest.Hash = strings.Repeat("9", 64)
	if err := writeCanonicalJSONFile(segmentPath, segment); err != nil {
		t.Fatalf("writeCanonicalJSONFile returned error: %v", err)
	}
	_, _, _, err := ledger.LatestVerificationSummaryAndViews(10)
	if err == nil {
		t.Fatal("LatestVerificationSummaryAndViews returned nil error, want digest mismatch")
	}
	if !strings.Contains(err.Error(), "frame record_digest mismatch") {
		t.Fatalf("error = %q, want frame record_digest mismatch", err)
	}
}
