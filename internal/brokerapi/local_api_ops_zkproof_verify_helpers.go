package brokerapi

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

func verifyArtifactPublicInputsDigest(artifact trustpolicy.ZKProofArtifactPayload, _ trustpolicy.Digest) (trustpolicy.Digest, error) {
	publicInputs, err := decodeArtifactPublicInputs(artifact.PublicInputs, artifact.PublicInputsDigest)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	recomputedDigest, err := zkproof.CanonicalPublicInputsDigestV0(publicInputs)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	if artifact.PublicInputsDigest != recomputedDigest {
		return trustpolicy.Digest{}, &zkproof.FeasibilityError{Code: "invalid_public_inputs_digest", Message: "proof public_inputs_digest does not match canonical typed public_inputs content"}
	}
	return recomputedDigest, nil
}

func (s *Service) buildVerificationRecordBase(artifactDigest trustpolicy.Digest, artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) (trustpolicy.ZKProofVerificationRecordPayload, error) {
	return trustpolicy.ZKProofVerificationRecordPayload{SchemaID: trustpolicy.ZKProofVerificationRecordSchemaID, SchemaVersion: trustpolicy.ZKProofVerificationRecordSchemaVersion, ProofDigest: artifactDigest, StatementFamily: artifact.StatementFamily, StatementVersion: artifact.StatementVersion, SchemeID: artifact.SchemeID, CurveID: artifact.CurveID, CircuitID: artifact.CircuitID, ConstraintSystemDigest: artifact.ConstraintSystemDigest, VerifierKeyDigest: artifact.VerifierKeyDigest, SetupProvenanceDigest: artifact.SetupProvenanceDigest, NormalizationProfileID: artifact.NormalizationProfileID, SchemeAdapterID: artifact.SchemeAdapterID, PublicInputsDigest: publicInputsDigest, VerifierImplementationID: zkProofVerifierImplementationID}, nil
}

func (s *Service) finalizeVerificationRecord(base trustpolicy.ZKProofVerificationRecordPayload, artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) (trustpolicy.ZKProofVerificationRecordPayload, error) {
	outcome, reasons, err := verifyProofArtifactOutcome(artifact, publicInputsDigest)
	if err != nil {
		return trustpolicy.ZKProofVerificationRecordPayload{}, err
	}
	base.VerifiedAt = s.now().UTC().Format(time.RFC3339)
	base.VerificationOutcome = outcome
	base.ReasonCodes = reasons
	base.CacheProvenance = "fresh"
	return base, nil
}

func verifyProofArtifactOutcome(artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) (string, []string, error) {
	proofBytes, publicInputs, identity, err := decodeProofVerificationInputs(artifact, publicInputsDigest)
	if err != nil {
		return "", nil, err
	}
	backend, _, _, authoritativeTrusted, err := zkproof.NewTrustedLocalGroth16BackendV0()
	if err != nil {
		return "", nil, err
	}
	err = zkproof.VerifyProofWithTrustedPostureV0(backend, proofBytes, publicInputs, identity, authoritativeTrusted)
	if err != nil {
		return trustpolicy.ProofVerificationOutcomeRejected, []string{classifyProofVerificationReason(err)}, nil
	}
	return trustpolicy.ProofVerificationOutcomeVerified, []string{trustpolicy.ProofVerificationReasonVerified}, nil
}

func decodeProofVerificationInputs(artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) ([]byte, zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs, zkproof.ProofVerificationIdentity, error) {
	proofBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(artifact.ProofBytes))
	if err != nil {
		return nil, zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs{}, zkproof.ProofVerificationIdentity{}, fmt.Errorf("decode proof bytes: %w", err)
	}
	publicInputs, err := decodeArtifactPublicInputs(artifact.PublicInputs, publicInputsDigest)
	if err != nil {
		return nil, zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs{}, zkproof.ProofVerificationIdentity{}, err
	}
	identity := zkproof.ProofVerificationIdentity{VerifierKeyDigest: artifact.VerifierKeyDigest, ConstraintSystemDigest: artifact.ConstraintSystemDigest, SetupProvenanceDigest: artifact.SetupProvenanceDigest}
	return proofBytes, publicInputs, identity, nil
}

func (s *Service) findCachedVerificationResponse(requestID string, artifactDigest trustpolicy.Digest, record trustpolicy.ZKProofVerificationRecordPayload) (ZKProofVerifyResponse, bool, error) {
	cachedDigest, cachedRecord, found, err := s.auditLedger.FindMatchingZKProofVerificationRecord(record)
	if err != nil || !found {
		return ZKProofVerifyResponse{}, false, err
	}
	return buildZKProofVerifyResponse(requestID, artifactDigest, cachedDigest, cachedRecord.VerificationOutcome, append([]string{}, cachedRecord.ReasonCodes...), "cache_hit"), true, nil
}
