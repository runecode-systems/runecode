package brokerapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunListStoreSyncFailureMessageIsSanitized(t *testing.T) {
	root := t.TempDir()
	storeRoot := filepath.Join(root, "store")
	ledgerRoot := filepath.Join(root, "ledger")
	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	statePath := filepath.Join(storeRoot, "state.json")
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove state.json returned error: %v", err)
	}
	if err := os.MkdirAll(statePath, 0o700); err != nil {
		t.Fatalf("MkdirAll(state.json as dir) returned error: %v", err)
	}

	_, errResp := service.HandleRunList(context.Background(), RunListRequest{
		SchemaID:      "runecode.protocol.v0.RunListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-run-list-sync-fail",
		Limit:         10,
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleRunList error response = nil, want storage failure")
	}
	if errResp.Error.Code != "broker_storage_write_failed" {
		t.Fatalf("error.code = %q, want broker_storage_write_failed", errResp.Error.Code)
	}
	if errResp.Error.Message != "store synchronization failed" {
		t.Fatalf("error.message = %q, want sanitized sync failure message", errResp.Error.Message)
	}
	if strings.Contains(errResp.Error.Message, statePath) {
		t.Fatalf("error.message leaked internal path: %q", errResp.Error.Message)
	}
}
