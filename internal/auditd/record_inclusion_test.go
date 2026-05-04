package auditd

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestRecordInclusionByDigestSingleSegmentSealed(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	second := validAdmissionRequestForLedger(t, newAuditFixtureKey(t))
	secondAppend, err := ledger.AppendAdmittedEvent(second)
	if err != nil {
		t.Fatalf("AppendAdmittedEvent(second) returned error: %v", err)
	}
	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment returned error: %v", err)
	}
	sealEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, nil, 0)
	sealResult, err := ledger.SealCurrentSegment(sealEnvelope)
	if err != nil {
		t.Fatalf("SealCurrentSegment returned error: %v", err)
	}
	sealID, _ := sealResult.SealEnvelopeDigest.Identity()
	recordID, _ := secondAppend.RecordDigest.Identity()
	inclusion := mustRecordInclusionByDigest(t, ledger, recordID)
	assertSingleSegmentSealedInclusion(t, inclusion, sealID)
	assertInclusionMerkleRecomputes(t, inclusion)
}

func TestRecordInclusionByDigestOpenSegmentWithoutSeal(t *testing.T) {
	_, ledger, _ := setupLedgerWithAdmissionFixture(t)
	recordID := recordDigestIdentity(t, ledger, 0)
	inclusion := mustRecordInclusionByDigest(t, ledger, recordID)

	if inclusion.SegmentID != "segment-000001" || inclusion.FrameIndex != 0 {
		t.Fatalf("inclusion segment/frame = %q/%d, want segment-000001/0", inclusion.SegmentID, inclusion.FrameIndex)
	}
	if inclusion.SegmentRecordCount != 1 {
		t.Fatalf("SegmentRecordCount = %d, want 1", inclusion.SegmentRecordCount)
	}
	if inclusion.SegmentSealDigest != "" {
		t.Fatalf("SegmentSealDigest = %q, want empty", inclusion.SegmentSealDigest)
	}
	if inclusion.SegmentSealChainIndex != nil {
		t.Fatalf("SegmentSealChainIndex = %v, want nil", inclusion.SegmentSealChainIndex)
	}
	if inclusion.PreviousSealDigest != "" {
		t.Fatalf("PreviousSealDigest = %q, want empty", inclusion.PreviousSealDigest)
	}

	assertInclusionMerkleRecomputes(t, inclusion)
}

func TestRecordInclusionByDigestMultiSegmentPreviousSealLinkage(t *testing.T) {
	_, ledger, fixture := setupLedgerWithAdmissionFixture(t)
	firstRecordID := recordDigestIdentity(t, ledger, 0)
	firstSeal := mustSealSegmentWithChain(t, ledger, fixture, "segment-000001", nil, 0)
	firstSealID, _ := firstSeal.SealEnvelopeDigest.Identity()
	secondAppend := mustAppendAdmissionFixture(t, ledger)
	secondSeal := mustSealSegmentWithChain(t, ledger, fixture, "segment-000002", &firstSeal.SealEnvelopeDigest, 1)
	secondSealID, _ := secondSeal.SealEnvelopeDigest.Identity()
	secondRecordID, _ := secondAppend.RecordDigest.Identity()
	firstInclusion := mustRecordInclusionByDigest(t, ledger, firstRecordID)
	assertInclusionSealLinkage(t, firstInclusion, firstSealID, "", "first")
	secondInclusion := mustRecordInclusionByDigest(t, ledger, secondRecordID)
	assertMultiSegmentSecondInclusion(t, secondInclusion, secondSealID, firstSealID)
	assertInclusionMerkleRecomputes(t, secondInclusion)
}

func mustRecordInclusionByDigest(t *testing.T, ledger *Ledger, recordID string) AuditRecordInclusion {
	t.Helper()
	inclusion, ok, err := ledger.RecordInclusionByDigest(recordID)
	if err != nil {
		t.Fatalf("RecordInclusionByDigest returned error: %v", err)
	}
	if !ok {
		t.Fatal("RecordInclusionByDigest found=false, want true")
	}
	return inclusion
}

func assertSingleSegmentSealedInclusion(t *testing.T, inclusion AuditRecordInclusion, sealID string) {
	t.Helper()
	if inclusion.SegmentID != "segment-000001" || inclusion.FrameIndex != 1 {
		t.Fatalf("inclusion segment/frame = %q/%d, want segment-000001/1", inclusion.SegmentID, inclusion.FrameIndex)
	}
	if inclusion.SegmentRecordCount != 2 {
		t.Fatalf("SegmentRecordCount = %d, want 2", inclusion.SegmentRecordCount)
	}
	if inclusion.OrderedMerkle.LeafCount != 2 || inclusion.OrderedMerkle.LeafIndex != 1 {
		t.Fatalf("ordered merkle leaf_count/index = %d/%d, want 2/1", inclusion.OrderedMerkle.LeafCount, inclusion.OrderedMerkle.LeafIndex)
	}
	assertInclusionSealChainIndex(t, inclusion.SegmentSealChainIndex, 0, "")
	assertInclusionSealLinkage(t, inclusion, sealID, "", "")
}

func assertMultiSegmentSecondInclusion(t *testing.T, inclusion AuditRecordInclusion, sealID string, previousSealID string) {
	t.Helper()
	if inclusion.SegmentID != "segment-000002" || inclusion.FrameIndex != 0 {
		t.Fatalf("second inclusion segment/frame = %q/%d, want segment-000002/0", inclusion.SegmentID, inclusion.FrameIndex)
	}
	assertInclusionSealChainIndex(t, inclusion.SegmentSealChainIndex, 1, "second")
	assertInclusionSealLinkage(t, inclusion, sealID, previousSealID, "second")
}

func assertInclusionSealChainIndex(t *testing.T, got *int64, want int64, label string) {
	t.Helper()
	prefix := label
	if prefix != "" {
		prefix += " inclusion "
	}
	if got == nil || *got != want {
		t.Fatalf("%sSegmentSealChainIndex = %v, want pointer to %d", prefix, got, want)
	}
}

func assertInclusionSealLinkage(t *testing.T, inclusion AuditRecordInclusion, sealID string, previousSealID string, label string) {
	t.Helper()
	prefix := label
	if prefix != "" {
		prefix += " inclusion "
	}
	if inclusion.SegmentSealDigest != sealID {
		t.Fatalf("%sSegmentSealDigest = %q, want %q", prefix, inclusion.SegmentSealDigest, sealID)
	}
	if previousSealID == "" {
		if inclusion.PreviousSealDigest != "" {
			t.Fatalf("%sPreviousSealDigest = %q, want empty", prefix, inclusion.PreviousSealDigest)
		}
		return
	}
	if inclusion.PreviousSealDigest != previousSealID {
		t.Fatalf("%sPreviousSealDigest = %q, want %q", prefix, inclusion.PreviousSealDigest, previousSealID)
	}
}

func mustSealSegmentWithChain(t *testing.T, ledger *Ledger, fixture auditFixtureKey, segmentID string, previous *trustpolicy.Digest, chainIndex int64) SealResult {
	t.Helper()
	segment, err := ledger.loadSegment(segmentID)
	if err != nil {
		t.Fatalf("loadSegment(%s) returned error: %v", segmentID, err)
	}
	sealEnvelope := buildSealEnvelopeForSegment(t, fixture, ledger, segment, previous, chainIndex)
	seal, err := ledger.SealCurrentSegment(sealEnvelope)
	if err != nil {
		t.Fatalf("SealCurrentSegment(%s) returned error: %v", segmentID, err)
	}
	return seal
}

func mustAppendAdmissionFixture(t *testing.T, ledger *Ledger) AppendResult {
	t.Helper()
	req := validAdmissionRequestForLedger(t, newAuditFixtureKey(t))
	appendResult, err := ledger.AppendAdmittedEvent(req)
	if err != nil {
		t.Fatalf("AppendAdmittedEvent returned error: %v", err)
	}
	return appendResult
}

func assertInclusionMerkleRecomputes(t *testing.T, inclusion AuditRecordInclusion) {
	t.Helper()
	if inclusion.OrderedMerkle.Profile != trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1 {
		t.Fatalf("OrderedMerkle.Profile = %q, want %q", inclusion.OrderedMerkle.Profile, trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1)
	}
	recordDigests := make([]trustpolicy.Digest, 0, len(inclusion.OrderedMerkle.SegmentRecordDigests))
	for i, identity := range inclusion.OrderedMerkle.SegmentRecordDigests {
		d, err := digestFromIdentity(identity)
		if err != nil {
			t.Fatalf("segment_record_digests[%d] invalid identity: %v", i, err)
		}
		recordDigests = append(recordDigests, d)
	}
	computedRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot(recordDigests)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	computedRootID, _ := computedRoot.Identity()
	if computedRootID != inclusion.OrderedMerkle.SegmentMerkleRoot {
		t.Fatalf("OrderedMerkle.SegmentMerkleRoot = %q, want %q", inclusion.OrderedMerkle.SegmentMerkleRoot, computedRootID)
	}
}

func recordDigestIdentity(t *testing.T, ledger *Ledger, frameIndex int) string {
	t.Helper()
	segment, err := ledger.loadSegment("segment-000001")
	if err != nil {
		t.Fatalf("loadSegment(segment-000001) returned error: %v", err)
	}
	if frameIndex < 0 || frameIndex >= len(segment.Frames) {
		t.Fatalf("frame index %d out of bounds", frameIndex)
	}
	id, err := segment.Frames[frameIndex].RecordDigest.Identity()
	if err != nil {
		t.Fatalf("RecordDigest.Identity returned error: %v", err)
	}
	return id
}
