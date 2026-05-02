package brokerapi

import (
	"context"
	"os"
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
