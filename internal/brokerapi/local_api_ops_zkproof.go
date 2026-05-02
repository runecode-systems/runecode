package brokerapi

import (
	"context"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

const (
	zkProofStatementFamilyV0      = zkproof.StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0
	zkProofStatementVersionV0     = zkproof.StatementVersionV0
	zkProofNormalizationProfileV0 = zkproof.NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0
	zkProofSchemeAdapterIDV0      = zkproof.SchemeAdapterGnarkGroth16IsolateSessionBoundV0
	zkProofCircuitIDV0            = "runecode.zk.circuit.audit.isolate_session_bound.attested_runtime_membership.v0"

	zkProofEvaluationGatePass = "first_narrow_proof_family_ready_for_check_in"
	zkProofUserCheckInNote    = "first narrow proof family evaluated; review persisted evaluation details before expanding proof scope"

	zkProofVerifierImplementationID = "runecode.trusted.zk.verifier.gnark.v0"
)

func (s *Service) HandleZKProofGenerate(ctx context.Context, req ZKProofGenerateRequest, meta RequestContext) (ZKProofGenerateResponse, *ErrorResponse) {
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if s == nil {
		errOut := toErrorResponse(requestID, "gateway_failure", "internal", false, "zk proof service unavailable")
		return ZKProofGenerateResponse{}, &errOut
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, zkProofGenerateRequestSchemaPath)
	if errResp != nil {
		return ZKProofGenerateResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ZKProofGenerateResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return ZKProofGenerateResponse{}, &errOut
	}

	resp, err := s.zkProofGenerateValidated(requestID, req.RecordDigest)
	if err != nil {
		errOut := s.zkProofGenerateErrorResponse(requestID, err)
		return ZKProofGenerateResponse{}, &errOut
	}
	if err := s.validateResponse(resp, zkProofGenerateResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ZKProofGenerateResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleZKProofVerify(ctx context.Context, req ZKProofVerifyRequest, meta RequestContext) (ZKProofVerifyResponse, *ErrorResponse) {
	requestID := resolveRequestID(req.RequestID, meta.RequestID)
	if s == nil {
		errOut := toErrorResponse(requestID, "gateway_failure", "internal", false, "zk proof service unavailable")
		return ZKProofVerifyResponse{}, &errOut
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, zkProofVerifyRequestSchemaPath)
	if errResp != nil {
		return ZKProofVerifyResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ZKProofVerifyResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return ZKProofVerifyResponse{}, &errOut
	}

	resp, err := s.zkProofVerifyValidated(requestID, req.ZKProofArtifactDigest)
	if err != nil {
		errOut := s.zkProofVerifyErrorResponse(requestID, err)
		return ZKProofVerifyResponse{}, &errOut
	}
	if err := s.validateResponse(resp, zkProofVerifyResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ZKProofVerifyResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) zkProofGenerateValidated(requestID string, recordDigest trustpolicy.Digest) (ZKProofGenerateResponse, error) {
	if err := s.requireZKProofBackend("generation"); err != nil {
		return ZKProofGenerateResponse{}, err
	}
	recordIdentity, inclusion, err := s.loadAuditRecordInclusion(recordDigest)
	if err != nil {
		return ZKProofGenerateResponse{}, err
	}
	compiled, path, runtimeEvidence, err := s.compileZKProofInput(inclusion)
	if err != nil {
		return ZKProofGenerateResponse{}, err
	}
	bindingDigest, err := s.persistCompiledAuditProofBinding(compiled, path, inclusion, runtimeEvidence)
	if err != nil {
		return ZKProofGenerateResponse{}, err
	}
	artifactPayload, err := s.persistCompiledProofArtifact(compiled, bindingDigest)
	if err != nil {
		return ZKProofGenerateResponse{}, err
	}
	proofDigest, err := s.auditLedger.PersistZKProofArtifact(artifactPayload)
	if err != nil {
		return ZKProofGenerateResponse{}, err
	}
	verifyResp, err := s.verifyProofArtifactAndPersistRecord(requestID, proofDigest, artifactPayload, artifactPayload.PublicInputsDigest)
	if err != nil {
		return ZKProofGenerateResponse{}, err
	}
	resp := buildZKProofGenerateResponse(requestID, inclusion.RecordDigest, bindingDigest, proofDigest, verifyResp.ZKProofVerificationRecordDigest)
	s.appendZKProofGenerateAuditEvent(recordIdentity, resp)
	return resp, nil
}

func (s *Service) zkProofVerifyValidated(requestID string, artifactDigest trustpolicy.Digest) (ZKProofVerifyResponse, error) {
	if err := s.requireZKProofBackend("verification"); err != nil {
		return ZKProofVerifyResponse{}, err
	}
	artifact, found, err := s.auditLedger.ZKProofArtifactByDigest(artifactDigest)
	if err != nil {
		return ZKProofVerifyResponse{}, err
	}
	if !found {
		identity, _ := artifactDigest.Identity()
		return ZKProofVerifyResponse{}, fmt.Errorf("zk proof artifact %q not found", identity)
	}
	resp, err := s.verifyProofArtifactAndPersistRecord(requestID, artifactDigest, artifact, artifact.PublicInputsDigest)
	if err != nil {
		return ZKProofVerifyResponse{}, err
	}
	_ = s.AppendTrustedAuditEvent("zk_proof_verify", "brokerapi", map[string]any{
		"zk_proof_artifact_digest":            mustDigestIdentityString(resp.ZKProofArtifactDigest),
		"zk_proof_verification_record_digest": mustDigestIdentityString(resp.ZKProofVerificationRecordDigest),
		"verification_outcome":                resp.VerificationOutcome,
		"reason_codes":                        resp.ReasonCodes,
		"cache_provenance":                    resp.CacheProvenance,
		"evaluation_gate":                     resp.EvaluationGate,
		"user_check_in_required":              resp.UserCheckInRequired,
	})
	return resp, nil
}

func (s *Service) verifyProofArtifactAndPersistRecord(requestID string, artifactDigest trustpolicy.Digest, artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) (ZKProofVerifyResponse, error) {
	publicInputsDigest, err := verifyArtifactPublicInputsDigest(artifact, publicInputsDigest)
	if err != nil {
		return ZKProofVerifyResponse{}, err
	}
	record, err := s.buildVerificationRecord(artifactDigest, artifact, publicInputsDigest)
	if err != nil {
		return ZKProofVerifyResponse{}, err
	}
	if cachedResp, found, err := s.findCachedVerificationResponse(requestID, artifactDigest, record); err == nil && found {
		return cachedResp, nil
	}
	if err != nil {
		return ZKProofVerifyResponse{}, err
	}
	verificationDigest, err := s.auditLedger.PersistZKProofVerificationRecord(record)
	if err != nil {
		return ZKProofVerifyResponse{}, err
	}
	return buildZKProofVerifyResponse(requestID, artifactDigest, verificationDigest, record.VerificationOutcome, record.ReasonCodes, "fresh"), nil
}
