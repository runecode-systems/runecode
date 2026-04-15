package brokerapi

import (
	"context"
	"errors"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditAnchorPreflightGet(ctx context.Context, req AuditAnchorPreflightGetRequest, meta RequestContext) (AuditAnchorPreflightGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditAnchorPreflightGetRequestSchemaPath)
	if errResp != nil {
		return AuditAnchorPreflightGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return AuditAnchorPreflightGetResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return AuditAnchorPreflightGetResponse{}, &errOut
	}
	resp, errResp := s.auditAnchorPreflightGetResponse(requestID)
	if errResp != nil {
		return AuditAnchorPreflightGetResponse{}, errResp
	}
	if err := s.validateResponse(resp, auditAnchorPreflightGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditAnchorPreflightGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) auditAnchorPreflightGetResponse(requestID string) (AuditAnchorPreflightGetResponse, *ErrorResponse) {
	if s.auditLedger == nil {
		errOut := s.makeError(requestID, auditAnchorErrorCodeLedgerUnavailable, "internal", false, "audit ledger unavailable")
		return AuditAnchorPreflightGetResponse{}, &errOut
	}
	resp := defaultAuditAnchorPreflightGetResponse(requestID)

	segmentID, sealDigest, sealFound, errResp := s.preflightLatestAnchorableSeal(requestID)
	if errResp != nil {
		return AuditAnchorPreflightGetResponse{}, errResp
	}
	if sealFound {
		resp.LatestAnchorableSeal = &AuditAnchorableSealRef{SegmentID: segmentID, SealDigest: sealDigest}
	}

	resp.SignerReadiness, resp.PresenceRequirements = s.preflightSignerAndPresence(requestID, sealDigest, sealFound)
	resp.VerifierReadiness = s.preflightVerifierReadiness()
	resp.ApprovalRequirements = s.preflightApprovalRequirements(sealDigest, sealFound)
	return resp, nil
}

func defaultAuditAnchorPreflightGetResponse(requestID string) AuditAnchorPreflightGetResponse {
	return AuditAnchorPreflightGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPreflightGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		SignerReadiness: AuditAnchorSignerReadiness{
			Ready:      false,
			ReasonCode: "signer_unavailable",
			Message:    "audit anchor signer is unavailable",
		},
		VerifierReadiness: AuditAnchorVerifierReadiness{
			Ready:      false,
			ReasonCode: "verifier_material_unavailable",
			Message:    "audit verifier material is unavailable",
		},
		PresenceRequirements: AuditAnchorPresenceRequirements{
			Required:   false,
			ReasonCode: "presence_mode_unavailable",
			Message:    "presence mode is unavailable",
		},
		ApprovalRequirements: AuditAnchorApprovalRequirements{
			Required:   false,
			ReasonCode: "no_anchorable_seal",
			Message:    "no latest anchorable seal is available",
		},
	}
}

func (s *Service) preflightLatestAnchorableSeal(requestID string) (string, trustpolicy.Digest, bool, *ErrorResponse) {
	segmentID, digest, err := s.auditLedger.LatestAnchorableSeal()
	if err == nil {
		return segmentID, digest, true, nil
	}
	if errors.Is(err, auditd.ErrNoSealedSegment) {
		return "", trustpolicy.Digest{}, false, nil
	}
	errOut := s.makeError(requestID, auditAnchorErrorCodeAnchorActionUnavailable, "internal", false, "latest anchorable seal lookup failed")
	return "", trustpolicy.Digest{}, false, &errOut
}

func (s *Service) preflightSignerAndPresence(requestID string, sealDigest trustpolicy.Digest, sealFound bool) (AuditAnchorSignerReadiness, AuditAnchorPresenceRequirements) {
	if s.secretsSvc == nil {
		return AuditAnchorSignerReadiness{Ready: false, ReasonCode: "signer_unavailable", Message: "audit anchor signer is unavailable"}, AuditAnchorPresenceRequirements{Required: false, ReasonCode: "signer_unavailable", Message: "presence requirement unavailable while signer is unavailable"}
	}
	mode := strings.TrimSpace(s.secretsSvc.AuditAnchorPresenceMode())
	if !isAuditAnchorPresenceMode(mode) {
		return AuditAnchorSignerReadiness{Ready: false, ReasonCode: "presence_mode_unavailable", Message: "audit anchor presence mode is unavailable"}, AuditAnchorPresenceRequirements{Required: false, ReasonCode: "presence_mode_unavailable", Message: "presence mode is unavailable"}
	}
	signer := AuditAnchorSignerReadiness{Ready: true, PresenceMode: mode, SignerLogicalScope: "node"}
	presence := AuditAnchorPresenceRequirements{Required: auditAnchorPresenceAttestationRequired(mode), AttestationMode: mode, AttestationReady: true}
	if !presence.Required {
		presence.ReasonCode = "presence_not_required"
		presence.Message = "presence attestation is not required for this mode"
		return signer, presence
	}
	if !sealFound {
		presence.AttestationReady = false
		presence.ReasonCode = "no_anchorable_seal"
		presence.Message = "presence attestation requires a latest anchorable seal"
		return signer, presence
	}
	if _, err := s.secretsSvc.ComputeAuditAnchorPresenceAcknowledgmentToken(mode, sealDigest, "presence-preflight-challenge"); err != nil {
		presence.AttestationReady = false
		presence.ReasonCode = "presence_attestation_unavailable"
		presence.Message = "presence attestation token generation is unavailable"
	}
	return signer, presence
}

func (s *Service) preflightVerifierReadiness() AuditAnchorVerifierReadiness {
	readiness, err := s.AuditReadiness()
	if err != nil {
		return AuditAnchorVerifierReadiness{Ready: false, ReasonCode: "readiness_unavailable", Message: "audit readiness is unavailable"}
	}
	if readiness.VerifierMaterialAvailable {
		return AuditAnchorVerifierReadiness{Ready: true}
	}
	return AuditAnchorVerifierReadiness{Ready: false, ReasonCode: "verifier_material_unavailable", Message: "audit verifier material is unavailable"}
}

func (s *Service) preflightApprovalRequirements(sealDigest trustpolicy.Digest, sealFound bool) AuditAnchorApprovalRequirements {
	if !sealFound {
		return AuditAnchorApprovalRequirements{Required: false, ReasonCode: "no_anchorable_seal", Message: "approval requirement depends on latest anchorable seal"}
	}
	required, err := s.anchorApprovalRequirement(sealDigest)
	if err != nil {
		if errors.Is(err, errAuditAnchorDeniedByPolicy) {
			return AuditAnchorApprovalRequirements{Required: false, ReasonCode: "anchor_denied_by_policy", Message: "anchoring denied by latest policy decision"}
		}
		return AuditAnchorApprovalRequirements{Required: false, ReasonCode: "approval_requirement_unavailable", Message: "approval requirement lookup unavailable"}
	}
	if required.Required {
		return AuditAnchorApprovalRequirements{Required: true, RequiredAssuranceLevel: strings.TrimSpace(required.RequiredAssurance), PolicyDecisionRef: strings.TrimSpace(required.PolicyDecisionRef), ReasonCode: "approval_required", Message: "policy requires approval for anchoring"}
	}
	if strings.TrimSpace(required.PolicyDecisionRef) != "" {
		return AuditAnchorApprovalRequirements{Required: false, PolicyDecisionRef: strings.TrimSpace(required.PolicyDecisionRef), ReasonCode: "approval_not_required", Message: "latest policy allows anchoring without approval"}
	}
	return AuditAnchorApprovalRequirements{Required: false, ReasonCode: "approval_not_required", Message: "no approval requirement declared"}
}
