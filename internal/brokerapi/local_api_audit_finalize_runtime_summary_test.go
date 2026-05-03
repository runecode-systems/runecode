package brokerapi

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestHandleAuditFinalizeVerifyPersistsRuntimeSummaryReceipt(t *testing.T) {
	service := setupFinalizeRuntimeSummaryService(t)
	finalizeRuntimeSummaryForTest(t, service)
	receipts := runtimeSummaryReceiptsForTest(t, service)
	assertRuntimeSummaryReceiptsPresent(t, receipts)
}

func setupFinalizeRuntimeSummaryService(t *testing.T) *Service {
	t.Helper()
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	return service
}

func finalizeRuntimeSummaryForTest(t *testing.T, service *Service) AuditFinalizeVerifyResponse {
	t.Helper()
	resp, errResp := service.HandleAuditFinalizeVerify(context.Background(), AuditFinalizeVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditFinalizeVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-finalize-runtime-summary",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditFinalizeVerify returned error: %+v", errResp)
	}
	if resp.ActionStatus != "ok" {
		t.Fatalf("action_status = %q, want ok", resp.ActionStatus)
	}
	return resp
}

func runtimeSummaryReceiptsForTest(t *testing.T, service *Service) []trustpolicy.SignedObjectEnvelope {
	t.Helper()
	_, sealDigest, err := service.auditLedger.LatestAnchorableSeal()
	if err != nil {
		t.Fatalf("LatestAnchorableSeal returned error: %v", err)
	}
	receipts, err := service.auditLedger.ReceiptsForSealDigest(sealDigest)
	if err != nil {
		t.Fatalf("ReceiptsForSealDigest returned error: %v", err)
	}
	return receipts
}

func assertRuntimeSummaryReceiptsPresent(t *testing.T, receipts []trustpolicy.SignedObjectEnvelope) {
	t.Helper()
	kinds := map[string]map[string]any{}
	for i := range receipts {
		payload, ok := decodeRuntimeSummaryTestReceipt(receipts[i])
		if !ok {
			continue
		}
		kind, _ := payload["audit_receipt_kind"].(string)
		kinds[kind] = payload
	}
	assertRuntimeSummaryPayloadFields(t, kinds["runtime_summary"])
	assertRuntimeSummaryKindPresent(t, kinds, "degraded_posture_summary")
	assertRuntimeSummaryKindPresent(t, kinds, "negative_capability_summary")
}

func decodeRuntimeSummaryTestReceipt(envelope trustpolicy.SignedObjectEnvelope) (map[string]any, bool) {
	payload := map[string]any{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func assertRuntimeSummaryPayloadFields(t *testing.T, payload map[string]any) {
	t.Helper()
	if payload == nil {
		t.Fatal("runtime_summary receipt not found")
	}
	rp, _ := payload["receipt_payload"].(map[string]any)
	for _, field := range []string{
		"no_provider_invocation",
		"no_secret_lease_issued",
		"no_approval_consumed",
		"no_artifact_crossed_boundary",
	} {
		if _, ok := rp[field]; !ok {
			t.Fatalf("runtime_summary missing %s", field)
		}
	}
}

func assertRuntimeSummaryKindPresent(t *testing.T, kinds map[string]map[string]any, kind string) {
	t.Helper()
	if kinds[kind] == nil {
		t.Fatalf("%s receipt not found", kind)
	}
}
