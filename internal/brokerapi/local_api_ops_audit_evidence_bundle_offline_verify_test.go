package brokerapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditEvidenceBundleOfflineVerifyRejectsMissingBundlePath(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify-missing",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleOfflineVerify expected validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestAuditEvidenceBundleOfflineVerifyRejectsNonTarBundlePath(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	bundlePath := t.TempDir() + "/bundle.json"
	if err := os.WriteFile(bundlePath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile(bundlePath) returned error: %v", err)
	}
	_, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify-json",
		BundlePath:    bundlePath,
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleOfflineVerify expected validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestAuditEvidenceBundleOfflineVerifyMissingTarReturnsInternalFailure(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	bundlePath := filepath.Join(t.TempDir(), "missing.tar")
	_, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify-missing-tar",
		BundlePath:    bundlePath,
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleOfflineVerify expected missing-file error")
	}
	if errResp.Error.Code != "gateway_failure" {
		t.Fatalf("error code = %q, want gateway_failure", errResp.Error.Code)
	}
	if errResp.Error.Category != "internal" {
		t.Fatalf("error category = %q, want internal", errResp.Error.Category)
	}
}

func TestOpenValidatedOfflineBundleFileRejectsSymlinkLeaf(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.tar")
	if err := os.WriteFile(target, []byte("tar"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) returned error: %v", err)
	}
	symlink := filepath.Join(dir, "bundle.tar")
	if err := os.Symlink(target, symlink); err != nil {
		t.Skipf("symlink setup unsupported on this platform: %v", err)
	}
	_, err := openValidatedOfflineBundleFile(symlink)
	if err == nil {
		t.Fatal("openValidatedOfflineBundleFile expected symlink validation error")
	}
	if !strings.Contains(err.Error(), "bundle_path must not reference a symlink") {
		t.Fatalf("error = %q, want symlink leaf validation", err)
	}
}

func TestOpenValidatedOfflineBundleFileRejectsPathSwapDuringOpen(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "bundle.tar")
	if err := os.WriteFile(bundlePath, []byte("first"), 0o600); err != nil {
		t.Fatalf("WriteFile(bundlePath) returned error: %v", err)
	}
	replacementPath := filepath.Join(dir, "replacement.tar")
	if err := os.WriteFile(replacementPath, []byte("second"), 0o600); err != nil {
		t.Fatalf("WriteFile(replacementPath) returned error: %v", err)
	}
	_, err := openValidatedOfflineBundleFileWithOpener(bundlePath, func(name string) (*os.File, error) {
		f, err := os.Open(replacementPath)
		if err != nil {
			return nil, err
		}
		return f, nil
	})
	if err == nil {
		t.Fatal("openValidatedOfflineBundleFile expected descriptor mismatch error")
	}
	if !strings.Contains(err.Error(), "bundle_path changed while opening") {
		t.Fatalf("error = %q, want path change detection", err)
	}
}
