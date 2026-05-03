package brokerapi

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditEvidenceSnapshotGetIncludesIdentityFoundationFields(t *testing.T) {
	service, _ := seededAuditRecordTestServiceAndDigest(t)
	resp, errResp := service.HandleAuditEvidenceSnapshotGet(context.Background(), AuditEvidenceSnapshotGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceSnapshotGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-snapshot-identities",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditEvidenceSnapshotGet error response: %+v", errResp)
	}
	if err := service.validateResponse(resp, auditEvidenceSnapshotGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditEvidenceSnapshotGetResponse) returned error: %v", err)
	}
	if resp.Snapshot.RepositoryIdentityDigest != nil {
		t.Fatal("repository_identity_digest present, want empty until canonical cross-machine repository identity exists")
	}
	if strings.TrimSpace(resp.Snapshot.ProductInstanceID) == "" {
		t.Fatal("product_instance_id empty, want repo-scoped product identity")
	}
	if strings.TrimSpace(resp.Snapshot.LedgerIdentity) == "" {
		t.Fatal("ledger_identity empty, want persistent ledger identity")
	}
	if resp.Snapshot.ProjectContextIdentityDigests != nil {
		t.Fatalf("project_context_identity_digests = %+v, want nil for seeded fixture without sidecar", resp.Snapshot.ProjectContextIdentityDigests)
	}
}

func TestProjectAuditEvidenceSnapshotProjectsAllDigestFamilies(t *testing.T) {
	snapshot := fullAuditEvidenceSnapshotFixture()

	projected, err := projectAuditEvidenceSnapshot(snapshot)
	if err != nil {
		t.Fatalf("projectAuditEvidenceSnapshot error = %v", err)
	}
	assertProjectedDigestFamilies(t, projected)
	if !reflect.DeepEqual(projected.RequiredApprovalIDs, snapshot.RequiredApprovalIDs) {
		t.Fatalf("RequiredApprovalIDs = %v, want %v", projected.RequiredApprovalIDs, snapshot.RequiredApprovalIDs)
	}
}

func fullAuditEvidenceSnapshotFixture() auditd.AuditEvidenceSnapshot {
	makeDigests := func(ch string) []string {
		return []string{"sha256:" + strings.Repeat(ch, 64)}
	}
	return auditd.AuditEvidenceSnapshot{
		SchemaID:                      "runecode.protocol.v0.AuditEvidenceSnapshot",
		SchemaVersion:                 "0.1.0",
		CreatedAt:                     "2026-05-03T00:00:00Z",
		SegmentSealDigests:            makeDigests("1"),
		AuditReceiptDigests:           makeDigests("2"),
		VerificationReportDigests:     makeDigests("3"),
		RuntimeEvidenceDigests:        makeDigests("4"),
		VerifierRecordDigests:         makeDigests("5"),
		EventContractCatalogDigests:   makeDigests("6"),
		SignerEvidenceDigests:         makeDigests("7"),
		StoragePostureDigests:         makeDigests("8"),
		TypedRequestDigests:           makeDigests("9"),
		ActionRequestDigests:          makeDigests("a"),
		ControlPlaneDigests:           makeDigests("b"),
		AttestationEvidenceDigests:    makeDigests("c"),
		ProjectContextIdentityDigests: makeDigests("d"),
		PolicyEvidenceDigests:         makeDigests("e"),
		RequiredApprovalIDs:           []string{"approval-1"},
		ApprovalEvidenceDigests:       makeDigests("f"),
		AnchorEvidenceDigests:         makeDigests("0"),
		ProviderInvocationDigests:     makeDigests("1"),
		SecretLeaseDigests:            makeDigests("2"),
	}
}

func assertProjectedDigestFamilies(t *testing.T, projected AuditEvidenceSnapshot) {
	t.Helper()
	for _, family := range []struct {
		name string
		got  []trustpolicy.Digest
	}{
		{name: "VerifierRecordDigests", got: projected.VerifierRecordDigests},
		{name: "EventContractCatalogDigests", got: projected.EventContractCatalogDigests},
		{name: "SignerEvidenceDigests", got: projected.SignerEvidenceDigests},
		{name: "StoragePostureDigests", got: projected.StoragePostureDigests},
		{name: "TypedRequestDigests", got: projected.TypedRequestDigests},
		{name: "ActionRequestDigests", got: projected.ActionRequestDigests},
		{name: "ControlPlaneDigests", got: projected.ControlPlaneDigests},
		{name: "AttestationEvidenceDigests", got: projected.AttestationEvidenceDigests},
		{name: "ProjectContextIdentityDigests", got: projected.ProjectContextIdentityDigests},
		{name: "PolicyEvidenceDigests", got: projected.PolicyEvidenceDigests},
		{name: "ApprovalEvidenceDigests", got: projected.ApprovalEvidenceDigests},
		{name: "AnchorEvidenceDigests", got: projected.AnchorEvidenceDigests},
		{name: "ProviderInvocationDigests", got: projected.ProviderInvocationDigests},
		{name: "SecretLeaseDigests", got: projected.SecretLeaseDigests},
	} {
		if len(family.got) != 1 {
			t.Fatalf("%s len = %d, want 1", family.name, len(family.got))
		}
	}
}
