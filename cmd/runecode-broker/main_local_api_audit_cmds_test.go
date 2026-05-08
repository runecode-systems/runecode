package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestAuditAnchorFailureReasonPrefersFailureCode(t *testing.T) {
	resp := brokerapi.AuditAnchorSegmentResponse{
		FailureCode:    "external_anchor_deferred_or_unavailable",
		FailureMessage: "external anchor confirmation is deferred",
	}
	if got := auditAnchorFailureReason(resp); got != "external_anchor_deferred_or_unavailable" {
		t.Fatalf("auditAnchorFailureReason() = %q, want external_anchor_deferred_or_unavailable", got)
	}
}

func TestAuditAnchorFailureReasonFallsBackToFailureMessage(t *testing.T) {
	resp := brokerapi.AuditAnchorSegmentResponse{FailureMessage: "external anchor confirmation is deferred"}
	if got := auditAnchorFailureReason(resp); got != "external anchor confirmation is deferred" {
		t.Fatalf("auditAnchorFailureReason() = %q, want external anchor confirmation is deferred", got)
	}
}

func TestAuditEvidenceBundleCommandsSmokePath(t *testing.T) {
	root := setBrokerServiceForTest(t)
	if err := seedLedgerForBrokerCommandTest(filepath.Join(root, "audit-ledger")); err != nil {
		t.Fatalf("seedLedgerForBrokerCommandTest returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if err := run([]string{"audit-evidence-snapshot-get"}, stdout, stderr); err != nil {
		t.Fatalf("audit-evidence-snapshot-get returned error: %v", err)
	}
	snapshot := brokerapi.AuditEvidenceSnapshot{}
	if err := json.Unmarshal(stdout.Bytes(), &snapshot); err != nil {
		t.Fatalf("audit-evidence-snapshot-get output parse error: %v", err)
	}
	if len(snapshot.SegmentSealDigests) == 0 {
		t.Fatal("snapshot.segment_seal_digests empty, want evidence snapshot material")
	}

	stdout.Reset()
	requestPath, outPath := writeAuditEvidenceBundleExportFixtures(t)
	if err := run([]string{"audit-evidence-bundle-export", "--request-file", requestPath, "--out", outPath}, stdout, stderr); err != nil {
		t.Fatalf("audit-evidence-bundle-export returned error: %v", err)
	}
	exportResp := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &exportResp); err != nil {
		t.Fatalf("audit-evidence-bundle-export output parse error: %v", err)
	}
	if got := strings.TrimSpace(exportResp["out"].(string)); got != outPath {
		t.Fatalf("export out path = %q, want %q", got, outPath)
	}
	if info, err := os.Stat(outPath); err != nil {
		t.Fatalf("Stat(export out) returned error: %v", err)
	} else if info.Size() == 0 {
		t.Fatal("exported bundle size = 0, want tar archive bytes")
	}

	stdout.Reset()
	if err := run([]string{"audit-evidence-bundle-offline-verify", "--bundle", outPath, "--archive-format", "tar"}, stdout, stderr); err != nil {
		t.Fatalf("audit-evidence-bundle-offline-verify returned error: %v", err)
	}
	verification := brokerapi.AuditEvidenceBundleOfflineVerification{}
	if err := json.Unmarshal(stdout.Bytes(), &verification); err != nil {
		t.Fatalf("audit-evidence-bundle-offline-verify output parse error: %v", err)
	}
	if verification.BundleID == "" || verification.VerificationStatus == "" {
		t.Fatalf("offline verification missing core fields: %+v", verification)
	}
	if len(verification.VerificationReports) == 0 {
		t.Fatal("offline verification reports empty, want projected report posture")
	}
}

func writeAuditEvidenceBundleExportFixtures(t *testing.T) (string, string) {
	t.Helper()
	tempRoot := canonicalTempDir(t)
	requestPath := filepath.Join(tempRoot, "audit-evidence-bundle-export.request.json")
	outPath := filepath.Join(tempRoot, "audit-evidence-bundle-export.tar")
	writeJSONFixtureFile(t, requestPath, map[string]any{
		"scope":              map[string]any{"scope_kind": "run", "run_id": "run-1"},
		"export_profile":     "external_relying_party_minimal",
		"created_by_tool":    map[string]any{"tool_name": "runecode-broker", "tool_version": "0.0.0-dev"},
		"disclosure_posture": map[string]any{"posture": "digest_metadata_only", "selective_disclosure_applied": true},
		"archive_format":     "tar",
	})
	return requestPath, outPath
}
