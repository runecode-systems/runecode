package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func bindingSourceRefs(inclusion auditd.AuditRecordInclusion, attestationVerificationDigest *trustpolicy.Digest) []trustpolicy.ZKProofSourceRef {
	refs := []trustpolicy.ZKProofSourceRef{{SourceFamily: "audit_segment_seal", SourceDigest: inclusion.SealEnvelopeDigest, SourceRole: "seal"}, {SourceFamily: "audit_event", SourceDigest: inclusion.RecordDigest, SourceRole: "event"}}
	if attestationVerificationDigest != nil {
		refs = append(refs, trustpolicy.ZKProofSourceRef{SourceFamily: "attestation_verification_record", SourceDigest: *attestationVerificationDigest, SourceRole: "attestation_verification"})
	}
	return refs
}

func parseDigestIdentityString(value string) (trustpolicy.Digest, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return trustpolicy.Digest{}, fmt.Errorf("digest identity must be sha256:<64 lowercase hex>")
	}
	d := trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}
	if _, err := d.Identity(); err != nil {
		return trustpolicy.Digest{}, err
	}
	return d, nil
}

func buildZKProofGenerateResponse(requestID string, recordDigest, bindingDigest, proofDigest, verificationDigest trustpolicy.Digest) ZKProofGenerateResponse {
	resp := ZKProofGenerateResponse{SchemaID: "runecode.protocol.v0.ZKProofGenerateResponse", SchemaVersion: "0.1.0", RequestID: requestID, StatementFamily: zkProofStatementFamilyV0, StatementVersion: zkProofStatementVersionV0, NormalizationProfileID: zkProofNormalizationProfileV0, SchemeAdapterID: zkProofSchemeAdapterIDV0, RecordDigest: recordDigest, AuditProofBindingDigest: bindingDigest, ZKProofArtifactDigest: proofDigest, EvaluationGate: zkProofEvaluationGatePass, UserCheckInRequired: true, CheckInNote: zkProofUserCheckInNote}
	resp.ZKProofVerificationDigest = &verificationDigest
	return resp
}

func buildZKProofVerifyResponse(requestID string, artifactDigest, verificationDigest trustpolicy.Digest, outcome string, reasons []string, cacheProvenance string) ZKProofVerifyResponse {
	return ZKProofVerifyResponse{SchemaID: "runecode.protocol.v0.ZKProofVerifyResponse", SchemaVersion: "0.1.0", RequestID: requestID, ZKProofArtifactDigest: artifactDigest, ZKProofVerificationRecordDigest: verificationDigest, VerificationOutcome: outcome, ReasonCodes: reasons, CacheProvenance: cacheProvenance, EvaluationGate: zkProofEvaluationGatePass, UserCheckInRequired: true, CheckInNote: zkProofUserCheckInNote}
}

func mustDigestIdentityString(d trustpolicy.Digest) string {
	identity, err := d.Identity()
	if err != nil {
		panic(err)
	}
	return identity
}

func canonicalMapDigest(value map[string]any) (trustpolicy.Digest, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	sum := sha256.Sum256(canonical)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func classifyProofVerificationReason(err error) string {
	var feasibilityErr *zkproof.FeasibilityError
	if !errors.As(err, &feasibilityErr) {
		return trustpolicy.ProofVerificationReasonProofInvalid
	}
	switch feasibilityErr.Code {
	case "setup_identity_mismatch":
		return trustpolicy.ProofVerificationReasonSetupIdentityMismatch
	case "unsupported_profile":
		return trustpolicy.ProofVerificationReasonUnsupportedProfile
	case "invalid_public_inputs_digest":
		return trustpolicy.ProofVerificationReasonInvalidPublicInputsHash
	case "unconfigured_proof_backend":
		return trustpolicy.ProofVerificationReasonUnconfiguredBackend
	default:
		return trustpolicy.ProofVerificationReasonProofInvalid
	}
}

func (s *Service) zkProofGenerateErrorResponse(requestID string, err error) ErrorResponse {
	return s.errorFromValidation(requestID, err)
}

func (s *Service) zkProofVerifyErrorResponse(requestID string, err error) ErrorResponse {
	return s.errorFromValidation(requestID, err)
}
