package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *Service) zkProofGenerateErrorResponse(requestID string, err error) ErrorResponse {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "zk proof generate failed"
	}
	code := "gateway_failure"
	category := "internal"
	if strings.Contains(message, "not found") {
		code = "broker_not_found_audit_record"
		category = "storage"
	} else if _, ok := err.(*zkproof.FeasibilityError); ok {
		code = "broker_validation_schema_invalid"
		category = "validation"
	}
	return s.makeError(requestID, code, category, false, message)
}

func (s *Service) zkProofVerifyErrorResponse(requestID string, err error) ErrorResponse {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "zk proof verify failed"
	}
	code := "gateway_failure"
	category := "internal"
	if strings.Contains(message, "not found") {
		code = "broker_not_found_artifact"
		category = "storage"
	} else if _, ok := err.(*zkproof.FeasibilityError); ok {
		code = "broker_validation_schema_invalid"
		category = "validation"
	}
	return s.makeError(requestID, code, category, false, message)
}

func canonicalMapDigest(value any) (trustpolicy.Digest, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	sum := sha256.Sum256(canonical)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func mustDigestIdentityString(d trustpolicy.Digest) string {
	identity, err := d.Identity()
	if err != nil {
		return ""
	}
	return identity
}

func classifyProofVerificationReason(err error) string {
	if err == nil {
		return trustpolicy.ProofVerificationReasonVerified
	}
	var feasibility *zkproof.FeasibilityError
	if errors.As(err, &feasibility) {
		switch strings.TrimSpace(feasibility.Code) {
		case "setup_identity_mismatch":
			return trustpolicy.ProofVerificationReasonSetupIdentityMismatch
		case "unsupported_profile":
			return trustpolicy.ProofVerificationReasonUnsupportedProfile
		case "unconfigured_proof_backend":
			return trustpolicy.ProofVerificationReasonUnconfiguredBackend
		case "invalid_public_inputs_digest":
			return trustpolicy.ProofVerificationReasonInvalidPublicInputsHash
		default:
			return trustpolicy.ProofVerificationReasonProofInvalid
		}
	}
	return trustpolicy.ProofVerificationReasonProofInvalid
}
