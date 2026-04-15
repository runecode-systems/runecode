package brokerapi

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditAnchorPresenceGet(ctx context.Context, req AuditAnchorPresenceGetRequest, meta RequestContext) (AuditAnchorPresenceGetResponse, *ErrorResponse) {
	requestID, requestCtx, release, errResp := s.startAuditAnchorPresenceGet(ctx, req, meta)
	if errResp != nil {
		return AuditAnchorPresenceGetResponse{}, errResp
	}
	defer release()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return AuditAnchorPresenceGetResponse{}, &errOut
	}
	resp, errResp := s.auditAnchorPresenceGetResponse(requestID, req)
	if errResp != nil {
		return AuditAnchorPresenceGetResponse{}, errResp
	}
	return s.validatedAuditAnchorPresenceGetResponse(resp)
}

func (s *Service) startAuditAnchorPresenceGet(ctx context.Context, req AuditAnchorPresenceGetRequest, meta RequestContext) (string, context.Context, func(), *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditAnchorPresenceGetRequestSchemaPath)
	if errResp != nil {
		return "", nil, nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return "", nil, nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	return requestID, requestCtx, func() {
		cancel()
		release()
	}, nil
}

func (s *Service) auditAnchorPresenceGetResponse(requestID string, req AuditAnchorPresenceGetRequest) (AuditAnchorPresenceGetResponse, *ErrorResponse) {
	if s.secretsSvc == nil {
		errOut := s.makeError(requestID, auditAnchorErrorCodeSignerUnavailable, "internal", false, "audit anchor signer unavailable")
		return AuditAnchorPresenceGetResponse{}, &errOut
	}
	if _, err := req.SealDigest.Identity(); err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, fmt.Sprintf("seal_digest: %v", err))
		return AuditAnchorPresenceGetResponse{}, &errOut
	}
	mode := strings.TrimSpace(s.secretsSvc.AuditAnchorPresenceMode())
	if !isAuditAnchorPresenceMode(mode) {
		errOut := s.makeError(requestID, auditAnchorErrorCodePresenceModeUnavailable, "internal", false, "audit anchor presence mode unavailable")
		return AuditAnchorPresenceGetResponse{}, &errOut
	}
	resp := AuditAnchorPresenceGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPresenceGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		SealDigest:    req.SealDigest,
		PresenceMode:  mode,
	}
	attestation, errResp := s.auditAnchorPresenceAttestation(requestID, mode, req.SealDigest)
	if errResp != nil {
		return AuditAnchorPresenceGetResponse{}, errResp
	}
	resp.PresenceAttestation = attestation
	return resp, nil
}

func (s *Service) auditAnchorPresenceAttestation(requestID string, mode string, sealDigest trustpolicy.Digest) (*AuditAnchorPresenceAttestation, *ErrorResponse) {
	if !auditAnchorPresenceAttestationRequired(mode) {
		return nil, nil
	}
	challenge, err := newAuditAnchorPresenceChallenge()
	if err != nil {
		errOut := s.makeError(requestID, auditAnchorErrorCodePresenceChallengeUnavailable, "internal", false, "audit anchor presence challenge unavailable")
		return nil, &errOut
	}
	token, err := s.secretsSvc.ComputeAuditAnchorPresenceAcknowledgmentToken(mode, sealDigest, challenge)
	if err != nil {
		errOut := s.makeError(requestID, auditAnchorErrorCodePresenceTokenUnavailable, "internal", false, "audit anchor presence token unavailable")
		return nil, &errOut
	}
	return &AuditAnchorPresenceAttestation{Challenge: challenge, AcknowledgmentToken: token}, nil
}

func (s *Service) validatedAuditAnchorPresenceGetResponse(resp AuditAnchorPresenceGetResponse) (AuditAnchorPresenceGetResponse, *ErrorResponse) {
	if err := s.validateResponse(resp, auditAnchorPresenceGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(resp.RequestID, err)
		return AuditAnchorPresenceGetResponse{}, &errOut
	}
	return resp, nil
}

func auditAnchorPresenceAttestationRequired(mode string) bool {
	mode = strings.TrimSpace(mode)
	return mode == "os_confirmation" || mode == "hardware_touch"
}

func isAuditAnchorPresenceMode(mode string) bool {
	mode = strings.TrimSpace(mode)
	return mode == "os_confirmation" || mode == "hardware_touch" || mode == "passphrase"
}

func newAuditAnchorPresenceChallenge() (string, error) {
	b := make([]byte, 16)
	if _, err := cryptorand.Read(b); err != nil {
		return "", err
	}
	return "presence-challenge-" + hex.EncodeToString(b), nil
}
