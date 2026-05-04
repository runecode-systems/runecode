package brokerapi

import (
	"context"
	"errors"
	"fmt"
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

func TestAuditEvidenceBundleOfflineVerifyRejectsSymlinkLeafBundlePath(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	dir := t.TempDir()
	realPath := filepath.Join(dir, "bundle-real.tar")
	if err := os.WriteFile(realPath, []byte("not-a-tar"), 0o600); err != nil {
		t.Fatalf("WriteFile(realPath) returned error: %v", err)
	}
	symlinkPath := filepath.Join(dir, "bundle-link.tar")
	if err := os.Symlink(realPath, symlinkPath); err != nil {
		t.Skipf("Symlink not supported in this environment: %v", err)
	}
	_, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify-symlink-leaf",
		BundlePath:    symlinkPath,
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleOfflineVerify expected validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
	if got := errResp.Error.Message; got != "bundle_path must not reference a symlink" {
		t.Fatalf("error message = %q, want symlink rejection", got)
	}
}

func TestAuditEvidenceBundleOfflineVerifyMissingPathUsesRedactedValidationError(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	sensitiveToken := "very-sensitive-local-path"
	bundlePath := filepath.Join(t.TempDir(), sensitiveToken, "bundle.tar")
	_, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify-open-failure-redacted",
		BundlePath:    bundlePath,
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleOfflineVerify expected validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
	if got := errResp.Error.Message; got != "bundle_path is not accessible" {
		t.Fatalf("error message = %q, want stable redacted accessibility failure", got)
	}
	if strings.Contains(errResp.Error.Message, sensitiveToken) {
		t.Fatalf("error message leaked sensitive path token %q: %q", sensitiveToken, errResp.Error.Message)
	}
}

func TestAuditEvidenceBundleOfflineVerifyVerifyFailureUsesStableMessage(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	dir := t.TempDir()
	sensitiveToken := "sensitive-bundle-location"
	bundlePath := filepath.Join(dir, sensitiveToken, "bundle.tar")
	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o700); err != nil {
		t.Fatalf("MkdirAll(bundle dir) returned error: %v", err)
	}
	if err := os.WriteFile(bundlePath, []byte("not a tar archive"), 0o600); err != nil {
		t.Fatalf("WriteFile(bundlePath) returned error: %v", err)
	}
	_, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify-verify-failure-redacted",
		BundlePath:    bundlePath,
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleOfflineVerify expected verification error")
	}
	if errResp.Error.Code != "gateway_failure" {
		t.Fatalf("error code = %q, want gateway_failure", errResp.Error.Code)
	}
	if got := errResp.Error.Message; got != "audit evidence bundle offline verify failed" {
		t.Fatalf("error message = %q, want stable redacted verify failure", got)
	}
	if strings.Contains(errResp.Error.Message, sensitiveToken) {
		t.Fatalf("error message leaked sensitive path token %q: %q", sensitiveToken, errResp.Error.Message)
	}
}

func TestOfflineBundleValidationClientMessage(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
		wantOK  bool
	}{
		{name: "required", err: errOfflineBundlePathRequired, wantMsg: "bundle_path is required", wantOK: true},
		{name: "absolute", err: errOfflineBundlePathAbsolute, wantMsg: "bundle_path must be an absolute path", wantOK: true},
		{name: "linked-components", err: errOfflineBundlePathLinkedComponents, wantMsg: "bundle_path must not contain symlink components", wantOK: true},
		{name: "symlink", err: errOfflineBundlePathSymlink, wantMsg: "bundle_path must not reference a symlink", wantOK: true},
		{name: "file", err: errOfflineBundlePathFile, wantMsg: "bundle_path must reference a file", wantOK: true},
		{name: "tar", err: errOfflineBundlePathTar, wantMsg: "bundle_path must reference a .tar archive", wantOK: true},
		{name: "not-accessible", err: fmt.Errorf("details: %w", errOfflineBundlePathNotAccessible), wantMsg: "bundle_path is not accessible", wantOK: true},
		{name: "wrapped-access", err: fmt.Errorf("details: %w", errOfflineBundlePathAccess), wantMsg: "", wantOK: false},
		{name: "other", err: errors.New("boom"), wantMsg: "", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMsg, gotOK := offlineBundleValidationClientMessage(tt.err)
			if gotOK != tt.wantOK {
				t.Fatalf("offlineBundleValidationClientMessage ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotMsg != tt.wantMsg {
				t.Fatalf("offlineBundleValidationClientMessage message = %q, want %q", gotMsg, tt.wantMsg)
			}
		})
	}
}
