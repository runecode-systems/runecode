package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditRecordInclusionGetSuccessProjectsDerivedInclusion(t *testing.T) {
	service, digest := seededAuditRecordTestServiceAndDigest(t)
	resp, errResp := service.HandleAuditRecordInclusionGet(context.Background(), AuditRecordInclusionGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditRecordInclusionGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-record-inclusion-get",
		RecordDigest:  digest,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditRecordInclusionGet error response: %+v", errResp)
	}
	if err := service.validateResponse(resp, auditRecordInclusionGetResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditRecordInclusionGetResponse) returned error: %v", err)
	}
	assertProjectedAuditRecordInclusion(t, resp.Inclusion)
}

func assertProjectedAuditRecordInclusion(t *testing.T, inclusion AuditRecordInclusion) {
	t.Helper()
	if inclusion.SchemaID != "runecode.protocol.v0.AuditRecordInclusion" || inclusion.SchemaVersion != "0.1.0" {
		t.Fatalf("inclusion schema = %q/%q, want runecode.protocol.v0.AuditRecordInclusion/0.1.0", inclusion.SchemaID, inclusion.SchemaVersion)
	}
	if inclusion.SegmentID == "" || inclusion.SegmentRecordCount <= 0 {
		t.Fatalf("inclusion segment projection invalid: %+v", inclusion)
	}
	if inclusion.RecordEnvelopeDigest.Hash == "" {
		t.Fatalf("record_envelope_digest empty: %+v", inclusion.RecordEnvelopeDigest)
	}
	if inclusion.OrderedMerkle.Profile != trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1 {
		t.Fatalf("ordered_merkle.profile = %q, want %q", inclusion.OrderedMerkle.Profile, trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1)
	}
	if len(inclusion.OrderedMerkle.SegmentRecordDigests) > 0 && inclusion.OrderedMerkle.LeafCount != len(inclusion.OrderedMerkle.SegmentRecordDigests) {
		t.Fatalf("ordered_merkle leaf_count=%d segment_record_digests=%d mismatch", inclusion.OrderedMerkle.LeafCount, len(inclusion.OrderedMerkle.SegmentRecordDigests))
	}
	if len(inclusion.OrderedMerkle.SegmentRecordDigests) > 0 && len(inclusion.OrderedMerkle.CompactPath) > 0 {
		t.Fatalf("ordered_merkle exposes both full-list and compact representations: %+v", inclusion.OrderedMerkle)
	}
	assertProjectedSealLinkage(t, inclusion)
	assertInclusionMerkleMaterialRecomputes(t, inclusion)
}

func assertProjectedSealLinkage(t *testing.T, inclusion AuditRecordInclusion) {
	t.Helper()
	if inclusion.SegmentSealDigest != nil && inclusion.SegmentSealChainIndex == nil {
		t.Fatalf("segment_seal_chain_index missing for present segment_seal_digest: %+v", inclusion)
	}
}

func TestAuditRecordInclusionGetNotFoundUsesAuditRecordCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleAuditRecordInclusionGet(context.Background(), AuditRecordInclusionGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditRecordInclusionGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-record-inclusion-missing",
		RecordDigest:  digestChar("f"),
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleAuditRecordInclusionGet expected not-found error")
	}
	if errResp.Error.Code != "broker_not_found_audit_record" {
		t.Fatalf("error code = %q, want broker_not_found_audit_record", errResp.Error.Code)
	}
	if errResp.Error.Message != "audit record not found" {
		t.Fatalf("error message = %q, want generic not-found message", errResp.Error.Message)
	}
}

func TestProjectAuditRecordInclusionRejectsChainIndexWithoutSealDigest(t *testing.T) {
	chainIndex := int64(3)
	_, err := projectAuditRecordInclusion(auditd.AuditRecordInclusion{
		RecordDigest:          "sha256:" + strings.Repeat("1", 64),
		RecordEnvelopeDigest:  "sha256:" + strings.Repeat("2", 64),
		SegmentID:             "segment-000003",
		FrameIndex:            0,
		SegmentRecordCount:    1,
		SegmentSealChainIndex: &chainIndex,
		OrderedMerkle: auditd.AuditRecordInclusionOrderedMerkleLookup{
			Profile:              trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1,
			LeafIndex:            0,
			LeafCount:            1,
			SegmentMerkleRoot:    "sha256:" + strings.Repeat("3", 64),
			SegmentRecordDigests: []string{"sha256:" + strings.Repeat("1", 64)},
		},
	})
	if err == nil {
		t.Fatal("projectAuditRecordInclusion returned nil error, want invariant failure")
	}
}

func assertInclusionMerkleMaterialRecomputes(t *testing.T, inclusion AuditRecordInclusion) {
	t.Helper()
	if inclusion.OrderedMerkle.LeafIndex < 0 || inclusion.OrderedMerkle.LeafIndex >= inclusion.OrderedMerkle.LeafCount {
		t.Fatalf("ordered_merkle leaf index/count invalid: %d/%d", inclusion.OrderedMerkle.LeafIndex, inclusion.OrderedMerkle.LeafCount)
	}
	gotRootID, err := inclusion.OrderedMerkle.SegmentMerkleRoot.Identity()
	if err != nil {
		t.Fatalf("segment_merkle_root invalid digest: %v", err)
	}
	if len(inclusion.OrderedMerkle.SegmentRecordDigests) > 0 {
		assertProjectedFullMerkleMaterial(t, inclusion.OrderedMerkle.SegmentRecordDigests, gotRootID)
		return
	}
	assertProjectedCompactMerkleMaterial(t, inclusion)
}

func assertProjectedFullMerkleMaterial(t *testing.T, recordDigests []trustpolicy.Digest, gotRootID string) {
	t.Helper()
	computedRoot, err := trustpolicy.ComputeOrderedAuditSegmentMerkleRoot(recordDigests)
	if err != nil {
		t.Fatalf("ComputeOrderedAuditSegmentMerkleRoot returned error: %v", err)
	}
	computedRootID, _ := computedRoot.Identity()
	if gotRootID != computedRootID {
		t.Fatalf("segment_merkle_root = %q, want %q", gotRootID, computedRootID)
	}
}

func assertProjectedCompactMerkleMaterial(t *testing.T, inclusion AuditRecordInclusion) {
	t.Helper()
	compactPath := make([]trustpolicy.Digest, 0, len(inclusion.OrderedMerkle.CompactPath))
	for i, digest := range inclusion.OrderedMerkle.CompactPath {
		if _, err := digest.Identity(); err != nil {
			t.Fatalf("compact_path[%d] invalid digest: %v", i, err)
		}
		compactPath = append(compactPath, digest)
	}
	if len(compactPath) == 0 {
		t.Fatal("expected either segment_record_digests or compact_path merkle material")
	}
	if err := trustpolicy.VerifyOrderedAuditSegmentMerkleCompactPath(inclusion.RecordDigest, inclusion.OrderedMerkle.LeafIndex, inclusion.OrderedMerkle.LeafCount, compactPath, inclusion.OrderedMerkle.SegmentMerkleRoot); err != nil {
		t.Fatalf("VerifyOrderedAuditSegmentMerkleCompactPath returned error: %v", err)
	}
}
