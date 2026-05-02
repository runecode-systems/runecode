package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditEvidenceBundleManifestGet(ctx context.Context, req AuditEvidenceBundleManifestGetRequest, meta RequestContext) (AuditEvidenceBundleManifestGetResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareAuditEvidenceRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditEvidenceBundleManifestGetRequestSchemaPath, meta, "audit evidence bundle manifest service unavailable")
	if errResp != nil {
		return AuditEvidenceBundleManifestGetResponse{}, errResp
	}
	defer cleanup()
	if errResp := s.requireAuditEvidenceLedger(requestID); errResp != nil {
		return AuditEvidenceBundleManifestGetResponse{}, errResp
	}
	if errResp := s.validateBundleManifestSharingPreconditions(requestID, req.ExternalSharingIntended); errResp != nil {
		return AuditEvidenceBundleManifestGetResponse{}, errResp
	}
	manifest, errResp := s.buildProjectedAuditEvidenceBundleManifest(requestID, req)
	if errResp != nil {
		return AuditEvidenceBundleManifestGetResponse{}, errResp
	}
	resp, errResp := s.buildAuditEvidenceBundleManifestGetResponse(requestID, manifest, req.ExternalSharingIntended)
	if errResp != nil {
		return AuditEvidenceBundleManifestGetResponse{}, errResp
	}
	if err := s.validateResponse(resp, auditEvidenceBundleManifestGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditEvidenceBundleManifestGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) buildAuditEvidenceBundleManifestGetResponse(requestID string, manifest AuditEvidenceBundleManifest, externalSharingIntended bool) (AuditEvidenceBundleManifestGetResponse, *ErrorResponse) {
	if errResp := s.validateBundleManifestForSharing(requestID, manifest, externalSharingIntended); errResp != nil {
		return AuditEvidenceBundleManifestGetResponse{}, errResp
	}
	resp := AuditEvidenceBundleManifestGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleManifestGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Manifest:      manifest,
	}
	if !externalSharingIntended {
		return resp, nil
	}
	envelope, errResp := s.signAndPersistAuditEvidenceBundleManifest(requestID, manifest)
	if errResp != nil {
		return AuditEvidenceBundleManifestGetResponse{}, errResp
	}
	resp.SignedManifest = envelope
	return resp, nil
}

func (s *Service) validateBundleManifestSharingPreconditions(requestID string, externalSharingIntended bool) *ErrorResponse {
	if !externalSharingIntended {
		return nil
	}
	if _, _, err := s.auditLedger.LatestAnchorableSeal(); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "external sharing requires a sealed segment")
		return &errOut
	}
	if s.secretsSvc == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "manifest signing unavailable")
		return &errOut
	}
	return nil
}

func (s *Service) buildProjectedAuditEvidenceBundleManifest(requestID string, req AuditEvidenceBundleManifestGetRequest) (AuditEvidenceBundleManifest, *ErrorResponse) {
	trustedManifest, err := s.auditLedger.BuildEvidenceBundleManifest(auditd.AuditEvidenceBundleManifestRequest{
		Scope:             projectAuditEvidenceBundleScopeToTrusted(req.Scope),
		ExportProfile:     req.ExportProfile,
		CreatedByTool:     projectAuditEvidenceBundleToolIdentityToTrusted(req.CreatedByTool),
		DisclosurePosture: projectAuditEvidenceBundleDisclosurePostureToTrusted(req.DisclosurePosture),
		Redactions:        projectAuditEvidenceBundleRedactionsToTrusted(req.Redactions),
	})
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditEvidenceBundleManifest{}, &errOut
	}
	manifest, err := projectAuditEvidenceBundleManifest(trustedManifest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle manifest projection failed")
		return AuditEvidenceBundleManifest{}, &errOut
	}
	if err := s.validateResponse(manifest, auditEvidenceBundleManifestSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditEvidenceBundleManifest{}, &errOut
	}
	return manifest, nil
}

func (s *Service) validateBundleManifestForSharing(requestID string, manifest AuditEvidenceBundleManifest, externalSharingIntended bool) *ErrorResponse {
	if !externalSharingIntended {
		return nil
	}
	if strings.TrimSpace(manifest.VerifierIdentity.KeyIDValue) == "" {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle manifest requires verifier identity for external sharing")
		return &errOut
	}
	if len(manifest.TrustRootDigests) == 0 {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle manifest requires trust roots for external sharing")
		return &errOut
	}
	return nil
}

func (s *Service) signAndPersistAuditEvidenceBundleManifest(requestID string, manifest AuditEvidenceBundleManifest) (*trustpolicy.SignedObjectEnvelope, *ErrorResponse) {
	envelope, err := s.signAuditEvidenceBundleManifestEnvelope(manifest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle manifest signing failed")
		return nil, &errOut
	}
	if err := s.validateResponse(envelope, "objects/SignedObjectEnvelope.schema.json"); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return nil, &errOut
	}
	if _, err := s.auditLedger.PersistBundleManifestEnvelope(envelope); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle manifest persistence failed")
		return nil, &errOut
	}
	return &envelope, nil
}
