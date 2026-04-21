package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditRecordGetSuccessProjectsTypedDetail(t *testing.T) {
	service, digest := seededAuditRecordTestServiceAndDigest(t)
	resp, errResp := service.HandleAuditRecordGet(context.Background(), AuditRecordGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditRecordGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-record-get",
		RecordDigest:  digest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditRecordGet error response: %+v", errResp)
	}
	assertProjectedAuditRecordDetail(t, resp)
	if err := service.validateResponse(resp, auditRecordGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditRecordGetResponse) returned error: %v", err)
	}
}

func seededAuditRecordTestServiceAndDigest(t *testing.T) (*Service, trustpolicy.Digest) {
	t.Helper()
	storeRoot := t.TempDir()
	ledgerRoot := t.TempDir()
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	surface, err := service.LatestAuditVerificationSurface(1)
	if err != nil {
		t.Fatalf("LatestAuditVerificationSurface returned error: %v", err)
	}
	if len(surface.Views) != 1 {
		t.Fatalf("views len = %d, want 1", len(surface.Views))
	}
	return service, surface.Views[0].RecordDigest
}

func assertProjectedAuditRecordDetail(t *testing.T, resp AuditRecordGetResponse) {
	t.Helper()
	if resp.Record.RecordFamily != "audit_event" {
		t.Fatalf("record_family = %q, want audit_event", resp.Record.RecordFamily)
	}
	if resp.Record.EventType == "" {
		t.Fatal("event_type empty, want projected event type")
	}
	if !strings.Contains(resp.Record.Summary, "Audit event") {
		t.Fatalf("summary = %q, want broker-owned event summary", resp.Record.Summary)
	}
	if len(resp.Record.LinkedReferences) == 0 {
		t.Fatal("linked_references empty, want projected audit links")
	}
	if resp.Record.ProjectContextID == "" {
		t.Fatal("record.project_context_identity_digest empty, want validated digest")
	}
	if resp.Record.Scope == nil || resp.Record.Scope.RunID == "" {
		t.Fatalf("scope = %+v, want derived run scope", resp.Record.Scope)
	}
	if resp.Record.Correlation == nil || resp.Record.Correlation.SessionID == "" {
		t.Fatalf("correlation = %+v, want derived correlation", resp.Record.Correlation)
	}
}

func TestAuditRecordGetNotFoundUsesAuditRecordCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleAuditRecordGet(context.Background(), AuditRecordGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditRecordGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-record-missing",
		RecordDigest:  digestChar("f"),
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditRecordGet expected not-found error")
	}
	if errResp.Error.Code != "broker_not_found_audit_record" {
		t.Fatalf("error code = %q, want broker_not_found_audit_record", errResp.Error.Code)
	}
}

func TestAuditRecordGetSchemaShapePreventsPrivateLeak(t *testing.T) {
	record := AuditRecordGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditRecordGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-shape",
		Record: AuditRecordDetail{
			SchemaID:         "runecode.protocol.v0.AuditRecordDetail",
			SchemaVersion:    "0.1.0",
			RecordDigest:     digestChar("a"),
			RecordFamily:     "audit_event",
			OccurredAt:       "2026-04-11T10:00:00Z",
			EventType:        "approval_requested",
			Summary:          "Approval request was emitted for a policy-gated stage transition.",
			LinkedReferences: []AuditRecordLinkedReference{},
		},
	}
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if err := s.validateResponse(record, auditRecordGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(valid projected record) returned error: %v", err)
	}
}

func TestAuditRecordPostureFromLatestReportTreatsInfoAsOK(t *testing.T) {
	service, digest := seededAuditRecordTestServiceAndDigest(t)
	digestID := mustTestDigestIdentity(t, digest)
	assertAuditRecordPosture(t, service, digestID, "ok", nil)
	persistInfoOnlyReportForDigest(t, service, digest)
	assertAuditRecordPosture(t, service, digestID, "ok", []string{trustpolicy.AuditVerificationReasonEventContractMissing})
}

func TestAuditRecordPostureDefaultsToOKWhenReportHasNoFindingForRecord(t *testing.T) {
	service, digest := seededAuditRecordTestServiceAndDigest(t)
	digestID := mustTestDigestIdentity(t, digest)
	assertAuditRecordPosture(t, service, digestID, "ok", nil)
}

func mustTestDigestIdentity(t *testing.T, digest trustpolicy.Digest) string {
	t.Helper()
	digestID, err := digest.Identity()
	if err != nil {
		t.Fatalf("digest.Identity() returned error: %v", err)
	}
	return digestID
}

func persistInfoOnlyReportForDigest(t *testing.T, service *Service, digest trustpolicy.Digest) {
	t.Helper()
	report, err := service.auditLedger.LatestVerificationReport()
	if err != nil {
		t.Fatalf("LatestVerificationReport returned error: %v", err)
	}
	report.Findings = append(report.Findings, trustpolicy.AuditVerificationFinding{
		Code:                trustpolicy.AuditVerificationReasonEventContractMissing,
		Dimension:           trustpolicy.AuditVerificationDimensionIntegrity,
		Severity:            trustpolicy.AuditVerificationSeverityInfo,
		Message:             "informational coverage finding",
		SubjectRecordDigest: &digest,
		RelatedRecordDigests: []trustpolicy.Digest{
			digest,
		},
	})
	report.VerifiedAt = "2099-01-01T00:00:00Z"
	if _, err := service.auditLedger.PersistVerificationReport(report); err != nil {
		t.Fatalf("PersistVerificationReport returned error: %v", err)
	}
}

func assertAuditRecordPosture(t *testing.T, service *Service, digestID string, wantStatus string, wantReasons []string) {
	t.Helper()
	reasons, posture := service.deriveRecordVerificationPosture(digestID)
	if posture == nil {
		t.Fatal("posture = nil, want explicit posture")
	}
	if posture.Status != wantStatus {
		t.Fatalf("posture.status = %q, want %q", posture.Status, wantStatus)
	}
	if !equalStrings(reasons, wantReasons) {
		t.Fatalf("reasons = %v, want %v", reasons, wantReasons)
	}
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func digestChar(ch string) trustpolicy.Digest {
	return trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat(ch, 64)}
}
