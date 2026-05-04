package auditd

import (
	"testing"
)

func TestLatestAnchorableSealReturnsLatestSealedSegment(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	first := mustSealFixtureSegment(t, ledger, fixture)
	if _, err := ledger.AppendAdmittedEvent(validAdmissionRequestForLedger(t, newAuditFixtureKey(t))); err != nil {
		t.Fatalf("AppendAdmittedEvent(second segment) returned error: %v", err)
	}
	segment, err := ledger.loadSegment("segment-000002")
	if err != nil {
		t.Fatalf("loadSegment(segment-000002) returned error: %v", err)
	}
	firstDigest := first.SealEnvelopeDigest
	secondEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, &firstDigest, 1)
	second, err := ledger.SealCurrentSegment(secondEnvelope)
	if err != nil {
		t.Fatalf("SealCurrentSegment(second segment) returned error: %v", err)
	}

	segmentID, digest, err := ledger.LatestAnchorableSeal()
	if err != nil {
		t.Fatalf("LatestAnchorableSeal returned error: %v", err)
	}
	if segmentID != "segment-000002" {
		t.Fatalf("segment_id = %q, want segment-000002", segmentID)
	}
	got, _ := digest.Identity()
	want, _ := second.SealEnvelopeDigest.Identity()
	if got != want {
		t.Fatalf("seal_digest = %q, want %q", got, want)
	}
	if gotFirst, _ := first.SealEnvelopeDigest.Identity(); got == gotFirst {
		t.Fatalf("latest seal digest should not equal first seal digest %q", gotFirst)
	}
}

func TestLatestAnchorableSealFailsWhenNoSealedSegment(t *testing.T) {
	root := t.TempDir()
	ledger, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if _, _, err := ledger.LatestAnchorableSeal(); err == nil {
		t.Fatal("LatestAnchorableSeal expected error when no sealed segment exists")
	}
}
