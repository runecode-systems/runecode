package brokerapi

import (
	"context"
	"testing"
)

func TestAuditEvidenceRetentionReviewReturnsSnapshotManifestAndCompleteness(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	resp, errResp := service.HandleAuditEvidenceRetentionReview(context.Background(), AuditEvidenceRetentionReviewRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceRetentionReviewRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-retention-review",
		Scope:         AuditEvidenceBundleScope{ScopeKind: "run", RunID: "run-1"},
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceRetentionReview error response: %+v", errResp)
	}
	if err := service.validateResponse(resp, auditEvidenceRetentionReviewResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditEvidenceRetentionReviewResponse) returned error: %v", err)
	}
	if resp.Snapshot.SchemaID != "runecode.protocol.v0.AuditEvidenceSnapshot" {
		t.Fatalf("snapshot.schema_id = %q, want runecode.protocol.v0.AuditEvidenceSnapshot", resp.Snapshot.SchemaID)
	}
	if resp.Manifest.SchemaID != "runecode.protocol.v0.AuditEvidenceBundleManifest" {
		t.Fatalf("manifest.schema_id = %q, want runecode.protocol.v0.AuditEvidenceBundleManifest", resp.Manifest.SchemaID)
	}
	if resp.Completeness.RequiredIdentityCount < 1 {
		t.Fatalf("required_identity_count = %d, want >=1", resp.Completeness.RequiredIdentityCount)
	}
	if resp.Completeness.FullySatisfied {
		t.Fatalf("fully_satisfied = true, want false while runtime evidence path is not exported in this lane")
	}
	if len(resp.Completeness.Missing) == 0 {
		t.Fatalf("missing = %+v, want explicit completeness gaps", resp.Completeness.Missing)
	}
	if !hasDigestAddressedCompletenessGap(resp.Completeness.Missing) {
		t.Fatalf("missing = %+v, want at least one digest-addressed completeness gap", resp.Completeness.Missing)
	}
}

func hasDigestAddressedCompletenessGap(entries []AuditEvidenceSnapshotIdentity) bool {
	for i := range entries {
		if entries[i].Identity != nil {
			return true
		}
	}
	return false
}
