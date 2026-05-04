package auditd

import (
	"archive/tar"
	"bytes"
	"strings"
	"testing"
)

func TestOfflineVerifyEvidenceBundleRejectsDuplicateTarPaths(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	archive := duplicateManifestTarArchive(t)
	_, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(archive), "tar")
	if err == nil {
		t.Fatal("OfflineVerifyEvidenceBundle expected duplicate-path error")
	}
}

func TestOfflineVerifyEvidenceBundleRejectsOversizedTarObject(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	archive := oversizedManifestTarArchive(t)
	_, err := ledger.OfflineVerifyEvidenceBundle(bytes.NewReader(archive), "tar")
	if err == nil {
		t.Fatal("OfflineVerifyEvidenceBundle expected oversized-object error")
	}
	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Fatalf("OfflineVerifyEvidenceBundle error = %q, want max-size failure", err)
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

func oversizedManifestTarArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	header := &tar.Header{Name: "manifest.json", Mode: 0o600, Size: offlineBundleTarMaxObjectBytes + 1}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader returned error: %v", err)
	}
	payload := make([]byte, 1024)
	for written := int64(0); written < header.Size; written += int64(len(payload)) {
		chunk := payload
		remaining := header.Size - written
		if remaining < int64(len(payload)) {
			chunk = payload[:remaining]
		}
		if _, err := tw.Write(chunk); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	return buf.Bytes()
}
