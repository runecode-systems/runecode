package brokerapi

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditEvidenceBundleExportStreamsManifestAndTarChunks(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	_, sealDigest, err := service.auditLedger.LatestAnchorableSeal()
	if err != nil {
		t.Fatalf("LatestAnchorableSeal returned error: %v", err)
	}
	events, errResp := service.HandleAuditEvidenceBundleExport(context.Background(), validAuditEvidenceBundleExportRequest("req-audit-bundle-export"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleExport error response: %+v", errResp)
	}
	assertAuditEvidenceBundleExportStartEvent(t, events)
	archiveBytes := gatherAuditBundleExportBytes(t, events)
	entries := readAuditBundleTarEntries(t, archiveBytes)
	if _, ok := entries["manifest.json"]; !ok {
		t.Fatal("manifest.json missing from streamed archive")
	}
	assertAuditEvidenceBundleExportChunking(t, events)
	assertAuditEvidenceBundleExportTerminal(t, events)
	assertAuditEvidenceBundleExportSchemaValidation(t, service, events)
	assertMetaAuditReceiptPresent(t, service, sealDigest, auditReceiptKindEvidenceBundleExport)
}

func validAuditEvidenceBundleExportRequest(requestID string) AuditEvidenceBundleExportRequest {
	return AuditEvidenceBundleExportRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleExportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Scope:         AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile: "operator_private_full",
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    "operator_private",
			SelectiveDisclosureApplied: false,
		},
		ArchiveFormat: "tar",
	}
}

func assertAuditEvidenceBundleExportStartEvent(t *testing.T, events []AuditEvidenceBundleExportEvent) {
	t.Helper()
	if len(events) < 3 {
		t.Fatalf("events len = %d, want >= 3", len(events))
	}
	if events[0].EventType != "audit_evidence_bundle_export_start" {
		t.Fatalf("start event type = %q, want audit_evidence_bundle_export_start", events[0].EventType)
	}
	if events[0].Manifest == nil || events[0].Manifest.Scope.RunID != "run-1" {
		t.Fatalf("start manifest = %+v, want run scope run-1", events[0].Manifest)
	}
	if events[0].ManifestDigest == nil {
		t.Fatal("start event missing manifest_digest")
	}
}

func assertAuditEvidenceBundleExportChunking(t *testing.T, events []AuditEvidenceBundleExportEvent) {
	t.Helper()
	for i := range events {
		if events[i].EventType == "audit_evidence_bundle_export_chunk" {
			return
		}
	}
	t.Fatal("missing chunk event in streamed export")
}

func assertAuditEvidenceBundleExportTerminal(t *testing.T, events []AuditEvidenceBundleExportEvent) {
	t.Helper()
	if len(events[len(events)-1].TerminalStatus) == 0 || events[len(events)-1].TerminalStatus != "completed" {
		t.Fatalf("terminal status = %q, want completed", events[len(events)-1].TerminalStatus)
	}
}

func assertAuditEvidenceBundleExportSchemaValidation(t *testing.T, service *Service, events []AuditEvidenceBundleExportEvent) {
	t.Helper()
	for i := range events {
		if err := service.validateResponse(events[i], auditEvidenceBundleExportEventSchemaPath); err != nil {
			t.Fatalf("validateResponse(event[%d]) returned error: %v", i, err)
		}
	}
}

func TestAuditEvidenceBundleExportRejectsInvalidScopeShape(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	_, errResp := service.HandleAuditEvidenceBundleExport(context.Background(), AuditEvidenceBundleExportRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleExportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-export-invalid",
		Scope:         AuditEvidenceBundleScope{ScopeKind: "artifact"},
		ExportProfile: "external_relying_party_minimal",
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    "digest_metadata_only",
			SelectiveDisclosureApplied: true,
		},
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleExport expected validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestAuditEvidenceBundleOfflineVerifyRejectsRelativeBundlePath(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify-relative",
		BundlePath:    "relative/bundle.tar",
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditEvidenceBundleOfflineVerify expected validation error")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestAuditEvidenceBundleExportStreamsAcrossManyChunksWhenChunkLimitSmall(t *testing.T) {
	service := newChunkedAuditEvidenceBundleExportService(t, 8)
	events, errResp := service.HandleAuditEvidenceBundleExport(context.Background(), validAuditEvidenceBundleExportRequest("req-audit-bundle-export-chunked"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleExport error response: %+v", errResp)
	}
	chunkEvents := countAuditEvidenceBundleChunkEvents(t, events, 8)
	if chunkEvents < 2 {
		t.Fatalf("chunk event count = %d, want multiple chunks when MaxStreamChunkBytes is small", chunkEvents)
	}
}

func newChunkedAuditEvidenceBundleExportService(t *testing.T, chunkBytes int) *Service {
	t.Helper()
	storeRoot := t.TempDir()
	ledgerRoot := t.TempDir()
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{
		RepositoryRoot: repositoryRootForProjectSubstrateTests(t),
		Limits:         Limits{MaxStreamChunkBytes: chunkBytes, MaxResponseStreamBytes: 1 << 20},
	})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	return service
}

func countAuditEvidenceBundleChunkEvents(t *testing.T, events []AuditEvidenceBundleExportEvent, maxChunkBytes int) int {
	t.Helper()
	chunkEvents := 0
	for i := range events {
		if events[i].EventType != "audit_evidence_bundle_export_chunk" {
			continue
		}
		chunkEvents++
		if events[i].ChunkBytes > maxChunkBytes {
			t.Fatalf("chunk bytes = %d, want <= %d", events[i].ChunkBytes, maxChunkBytes)
		}
	}
	return chunkEvents
}

func TestAuditEvidenceBundleOfflineVerifySurfacesDegradedPostureFromBundle(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	path, cleanup := exportAuditBundleFileForOfflineVerifyTest(t, service)
	defer cleanup()
	resp, errResp := service.HandleAuditEvidenceBundleOfflineVerify(context.Background(), AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-offline-verify",
		BundlePath:    path,
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleOfflineVerify error response: %+v", errResp)
	}
	if err := service.validateResponse(resp, auditEvidenceBundleOfflineVerifyResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditEvidenceBundleOfflineVerifyResponse) returned error: %v", err)
	}
	if resp.Verification.VerifierIdentity.KeyIDValue == "" {
		t.Fatal("verification.verifier_identity.key_id_value empty, want preserved verifier identity")
	}
	if len(resp.Verification.TrustRootDigests) == 0 {
		t.Fatal("verification.trust_root_digests empty, want preserved trust-root identity")
	}
	if resp.Verification.VerificationStatus == "" {
		t.Fatal("verification_status empty, want explicit offline status")
	}
	if len(resp.Verification.VerificationReports) == 0 {
		t.Fatal("verification_reports empty, want projected bundle report posture")
	}
}

func exportAuditBundleFileForOfflineVerifyTest(t *testing.T, service *Service) (string, func()) {
	t.Helper()
	events, errResp := service.HandleAuditEvidenceBundleExport(context.Background(), AuditEvidenceBundleExportRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleExportRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-bundle-export-offline-verify",
		Scope:         AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
		ExportProfile: "external_relying_party_minimal",
		CreatedByTool: AuditEvidenceBundleToolIdentity{ToolName: "runecode-broker", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{
			Posture:                    "digest_metadata_only",
			SelectiveDisclosureApplied: true,
		},
		ArchiveFormat: "tar",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceBundleExport error response: %+v", errResp)
	}
	archiveBytes := gatherAuditBundleExportBytes(t, events)
	dir := t.TempDir()
	path := filepath.Join(dir, "offline-verify-bundle.tar")
	if err := os.WriteFile(path, archiveBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(bundle) returned error: %v", err)
	}
	return path, func() {}
}

func gatherAuditBundleExportBytes(t *testing.T, events []AuditEvidenceBundleExportEvent) []byte {
	t.Helper()
	var out bytes.Buffer
	for i := range events {
		e := events[i]
		if e.EventType != "audit_evidence_bundle_export_chunk" {
			continue
		}
		chunk, err := base64.StdEncoding.DecodeString(e.ChunkBase64)
		if err != nil {
			t.Fatalf("DecodeString(chunk[%d]) returned error: %v", i, err)
		}
		if _, err := out.Write(chunk); err != nil {
			t.Fatalf("Write(chunk[%d]) returned error: %v", i, err)
		}
	}
	return out.Bytes()
}

func readAuditBundleTarEntries(t *testing.T, archive []byte) map[string][]byte {
	t.Helper()
	entries := map[string][]byte{}
	tr := tar.NewReader(bytes.NewReader(archive))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return entries
		}
		if err != nil {
			t.Fatalf("tar.Next returned error: %v", err)
		}
		payload, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("ReadAll(%q) returned error: %v", header.Name, err)
		}
		entries[strings.TrimSpace(header.Name)] = payload
	}
}

func assertMetaAuditReceiptPresent(t *testing.T, service *Service, sealDigest trustpolicy.Digest, kind string) {
	t.Helper()
	receipts, err := service.auditLedger.ReceiptsForSealDigest(sealDigest)
	if err != nil {
		t.Fatalf("ReceiptsForSealDigest returned error: %v", err)
	}
	for i := range receipts {
		payload := map[string]any{}
		if err := json.Unmarshal(receipts[i].Payload, &payload); err != nil {
			continue
		}
		if receiptKind, _ := payload["audit_receipt_kind"].(string); receiptKind == kind {
			return
		}
	}
	t.Fatalf("meta-audit receipt kind %q not found", kind)
}
