package brokerapi

import (
	"context"

	"github.com/runecode-ai/runecode/internal/auditd"
)

func (s *Service) HandleAuditEvidenceRetentionReview(ctx context.Context, req AuditEvidenceRetentionReviewRequest, meta RequestContext) (AuditEvidenceRetentionReviewResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareAuditEvidenceRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditEvidenceRetentionReviewRequestSchemaPath, meta, "audit evidence retention review service unavailable")
	if errResp != nil {
		return AuditEvidenceRetentionReviewResponse{}, errResp
	}
	defer cleanup()
	if errResp := s.requireAuditEvidenceLedger(requestID); errResp != nil {
		return AuditEvidenceRetentionReviewResponse{}, errResp
	}
	snapshot, manifest, projectedCompleteness, errResp := s.buildProjectedAuditEvidenceRetentionReview(requestID, req.Scope)
	if errResp != nil {
		return AuditEvidenceRetentionReviewResponse{}, errResp
	}
	resp := AuditEvidenceRetentionReviewResponse{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceRetentionReviewResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Snapshot:      snapshot,
		Manifest:      manifest,
		Completeness:  projectedCompleteness,
	}
	if err := s.validateResponse(resp, auditEvidenceRetentionReviewResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditEvidenceRetentionReviewResponse{}, &errOut
	}
	manifestDigest, err := canonicalDigest(resp.Manifest)
	if err == nil {
		s.persistMetaAuditReceipt(auditReceiptKindSensitiveEvidenceView, "audit_evidence_retention_review", manifestDigestRefOrNil(manifestDigest), nil, manifestDigestRefOrNil(manifestDigest), "retention_review")
	}
	return resp, nil
}

func (s *Service) buildProjectedAuditEvidenceRetentionReview(requestID string, scope AuditEvidenceBundleScope) (AuditEvidenceSnapshot, AuditEvidenceBundleManifest, AuditEvidenceSnapshotCompleteness, *ErrorResponse) {
	trustedScope, err := projectAuditEvidenceBundleScopeToTrusted(scope)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditEvidenceSnapshot{}, AuditEvidenceBundleManifest{}, AuditEvidenceSnapshotCompleteness{}, &errOut
	}
	trustedSnapshot, trustedManifest, completeness, err := s.auditLedger.BuildEvidenceRetentionReview(trustedScope, s.auditEvidenceIdentityContext())
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditEvidenceSnapshot{}, AuditEvidenceBundleManifest{}, AuditEvidenceSnapshotCompleteness{}, &errOut
	}
	snapshot, err := projectAuditEvidenceSnapshot(trustedSnapshot)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence retention snapshot projection failed")
		return AuditEvidenceSnapshot{}, AuditEvidenceBundleManifest{}, AuditEvidenceSnapshotCompleteness{}, &errOut
	}
	manifest, err := projectAuditEvidenceBundleManifest(trustedManifest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence retention manifest projection failed")
		return AuditEvidenceSnapshot{}, AuditEvidenceBundleManifest{}, AuditEvidenceSnapshotCompleteness{}, &errOut
	}
	projectedCompleteness, err := projectAuditEvidenceSnapshotCompleteness(completeness)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence retention completeness projection failed")
		return AuditEvidenceSnapshot{}, AuditEvidenceBundleManifest{}, AuditEvidenceSnapshotCompleteness{}, &errOut
	}
	return snapshot, manifest, projectedCompleteness, nil
}

func projectAuditEvidenceSnapshotCompleteness(review auditd.AuditEvidenceSnapshotCompletenessReview) (AuditEvidenceSnapshotCompleteness, error) {
	missing, err := projectAuditEvidenceSnapshotIdentityEntries(review.Missing)
	if err != nil {
		return AuditEvidenceSnapshotCompleteness{}, err
	}
	declared, err := projectAuditEvidenceSnapshotIdentityEntries(review.DeclaredRedactions)
	if err != nil {
		return AuditEvidenceSnapshotCompleteness{}, err
	}
	transitive, err := projectAuditEvidenceSnapshotIdentityEntries(review.TransitiveEmbedded)
	if err != nil {
		return AuditEvidenceSnapshotCompleteness{}, err
	}
	unsupported, err := projectAuditEvidenceSnapshotIdentityEntries(review.UnsupportedDirectCompleteness)
	if err != nil {
		return AuditEvidenceSnapshotCompleteness{}, err
	}
	return AuditEvidenceSnapshotCompleteness{
		FullySatisfied:                  review.FullySatisfied,
		RequiredIdentityCount:           review.RequiredIdentityCount,
		Missing:                         missing,
		DeclaredRedactions:              declared,
		TransitiveEmbedded:              transitive,
		UnsupportedDirectCompleteness:   unsupported,
		TransitiveEmbeddedIdentityCount: review.TransitiveEmbeddedIdentityCount,
		UnsupportedDirectIdentityCount:  review.UnsupportedDirectIdentityCount,
	}, nil
}

func projectAuditEvidenceSnapshotIdentityEntries(entries []auditd.AuditEvidenceSnapshotCompleteness) ([]AuditEvidenceSnapshotIdentity, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]AuditEvidenceSnapshotIdentity, 0, len(entries))
	for i := range entries {
		if entries[i].Identity == "" {
			out = append(out, AuditEvidenceSnapshotIdentity{Family: entries[i].Family})
			continue
		}
		d, err := digestFromIdentity(entries[i].Identity)
		if err != nil {
			return nil, err
		}
		digest := d
		out = append(out, AuditEvidenceSnapshotIdentity{Family: entries[i].Family, Identity: &digest})
	}
	return out, nil
}
