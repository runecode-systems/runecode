package brokerapi

import (
	"context"
	"fmt"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditRecordInclusionGet(ctx context.Context, req AuditRecordInclusionGetRequest, meta RequestContext) (AuditRecordInclusionGetResponse, *ErrorResponse) {
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if s == nil {
		errOut := toErrorResponse(requestID, "gateway_failure", "internal", false, "audit record inclusion service unavailable")
		return AuditRecordInclusionGetResponse{}, &errOut
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditRecordInclusionGetRequestSchemaPath)
	if errResp != nil {
		return AuditRecordInclusionGetResponse{}, errResp
	}
	recordIdentity, errResp := s.validatedAuditRecordInclusionGetRequest(ctx, req, requestID, meta)
	if errResp != nil {
		return AuditRecordInclusionGetResponse{}, errResp
	}
	inclusion, errResp := s.lookupAuditRecordInclusion(recordIdentity, requestID)
	if errResp != nil {
		return AuditRecordInclusionGetResponse{}, errResp
	}
	return s.validatedAuditRecordInclusionGetResponse(requestID, inclusion)
}

func (s *Service) validatedAuditRecordInclusionGetRequest(ctx context.Context, req AuditRecordInclusionGetRequest, requestID string, meta RequestContext) (string, *ErrorResponse) {
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return "", &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return "", &errOut
	}
	recordIdentity, err := req.RecordDigest.Identity()
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, fmt.Sprintf("record_digest: %v", err))
		return "", &errOut
	}
	return recordIdentity, nil
}

func (s *Service) lookupAuditRecordInclusion(recordIdentity string, requestID string) (AuditRecordInclusion, *ErrorResponse) {
	if s.auditLedger == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit ledger unavailable")
		return AuditRecordInclusion{}, &errOut
	}
	trustedInclusion, found, err := s.auditLedger.RecordInclusionByDigest(recordIdentity)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit record inclusion lookup failed")
		return AuditRecordInclusion{}, &errOut
	}
	if !found {
		errOut := s.makeError(requestID, "broker_not_found_audit_record", "storage", false, "audit record not found")
		return AuditRecordInclusion{}, &errOut
	}
	inclusion, err := projectAuditRecordInclusion(trustedInclusion)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit record inclusion projection failed")
		return AuditRecordInclusion{}, &errOut
	}
	return inclusion, nil
}

func projectAuditRecordInclusion(inclusion auditd.AuditRecordInclusion) (AuditRecordInclusion, error) {
	projected, err := newProjectedAuditRecordInclusion(inclusion)
	if err != nil {
		return AuditRecordInclusion{}, err
	}
	if err := attachProjectedSealMaterial(&projected, inclusion); err != nil {
		return AuditRecordInclusion{}, err
	}
	return projected, nil
}

func newProjectedAuditRecordInclusion(inclusion auditd.AuditRecordInclusion) (AuditRecordInclusion, error) {
	recordDigest, err := digestFromIdentity(inclusion.RecordDigest)
	if err != nil {
		return AuditRecordInclusion{}, err
	}
	recordEnvelopeDigest, err := digestFromIdentity(inclusion.RecordEnvelopeDigest)
	if err != nil {
		return AuditRecordInclusion{}, err
	}
	orderedMerkle, err := projectAuditRecordInclusionOrderedMerkle(inclusion.OrderedMerkle)
	if err != nil {
		return AuditRecordInclusion{}, err
	}
	return AuditRecordInclusion{
		SchemaID:             "runecode.protocol.v0.AuditRecordInclusion",
		SchemaVersion:        "0.1.0",
		RecordDigest:         recordDigest,
		RecordEnvelopeDigest: recordEnvelopeDigest,
		SegmentID:            inclusion.SegmentID,
		FrameIndex:           inclusion.FrameIndex,
		SegmentRecordCount:   inclusion.SegmentRecordCount,
		OrderedMerkle:        orderedMerkle,
	}, nil
}

func projectAuditRecordInclusionOrderedMerkle(lookup auditd.AuditRecordInclusionOrderedMerkleLookup) (AuditRecordInclusionOrderedMerkle, error) {
	segmentMerkleRoot, err := digestFromIdentity(lookup.SegmentMerkleRoot)
	if err != nil {
		return AuditRecordInclusionOrderedMerkle{}, err
	}
	segmentRecordDigests, err := projectAuditRecordDigestIdentities(lookup.SegmentRecordDigests)
	if err != nil {
		return AuditRecordInclusionOrderedMerkle{}, err
	}
	return AuditRecordInclusionOrderedMerkle{
		Profile:              lookup.Profile,
		LeafIndex:            lookup.LeafIndex,
		LeafCount:            lookup.LeafCount,
		SegmentMerkleRoot:    segmentMerkleRoot,
		SegmentRecordDigests: segmentRecordDigests,
	}, nil
}

func attachProjectedSealMaterial(projected *AuditRecordInclusion, inclusion auditd.AuditRecordInclusion) error {
	if err := attachProjectedSegmentSeal(projected, inclusion); err != nil {
		return err
	}
	if inclusion.PreviousSealDigest == "" {
		return nil
	}
	if projected.SegmentSealDigest == nil {
		return fmt.Errorf("previous_seal_digest present without segment_seal_digest")
	}
	previousSealDigest, err := digestFromIdentity(inclusion.PreviousSealDigest)
	if err != nil {
		return err
	}
	projected.PreviousSealDigest = &previousSealDigest
	return nil
}

func attachProjectedSegmentSeal(projected *AuditRecordInclusion, inclusion auditd.AuditRecordInclusion) error {
	if inclusion.SegmentSealDigest == "" {
		if inclusion.SegmentSealChainIndex != nil {
			return fmt.Errorf("segment_seal_chain_index present without segment_seal_digest")
		}
		return nil
	}
	segmentSealDigest, err := digestFromIdentity(inclusion.SegmentSealDigest)
	if err != nil {
		return err
	}
	projected.SegmentSealDigest = &segmentSealDigest
	if inclusion.SegmentSealChainIndex == nil {
		return fmt.Errorf("segment_seal_chain_index missing for sealed segment")
	}
	chainIndex := *inclusion.SegmentSealChainIndex
	projected.SegmentSealChainIndex = &chainIndex
	return nil
}

func projectAuditRecordDigestIdentities(identities []string) ([]trustpolicy.Digest, error) {
	out := make([]trustpolicy.Digest, 0, len(identities))
	for _, identity := range identities {
		digest, err := digestFromIdentity(identity)
		if err != nil {
			return nil, err
		}
		out = append(out, digest)
	}
	return out, nil
}

func (s *Service) validatedAuditRecordInclusionGetResponse(requestID string, inclusion AuditRecordInclusion) (AuditRecordInclusionGetResponse, *ErrorResponse) {
	if err := s.validateResponse(inclusion, auditRecordInclusionSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditRecordInclusionGetResponse{}, &errOut
	}
	resp := AuditRecordInclusionGetResponse{SchemaID: "runecode.protocol.v0.AuditRecordInclusionGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Inclusion: inclusion}
	if err := s.validateResponse(resp, auditRecordInclusionGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditRecordInclusionGetResponse{}, &errOut
	}
	return resp, nil
}
