package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

func (s *Service) requireZKProofFixtureBackend(operation string) error {
	if s.auditLedger == nil {
		return fmt.Errorf("audit ledger unavailable")
	}
	if s.apiConfig.ZKProof.EnableFixtureBackend {
		return nil
	}
	return &zkproof.FeasibilityError{Code: "unconfigured_proof_backend", Message: fmt.Sprintf("zk proof %s requires explicit trusted fixture backend enablement in this evaluation-only implementation", operation)}
}

func (s *Service) loadAuditRecordInclusion(recordDigest trustpolicy.Digest) (string, auditd.AuditRecordInclusion, error) {
	recordIdentity, err := recordDigest.Identity()
	if err != nil {
		return "", auditd.AuditRecordInclusion{}, fmt.Errorf("record_digest: %w", err)
	}
	inclusion, found, err := s.auditLedger.AuditRecordInclusion(recordIdentity)
	if err != nil {
		return "", auditd.AuditRecordInclusion{}, err
	}
	if !found {
		return "", auditd.AuditRecordInclusion{}, fmt.Errorf("audit record %q not found", recordIdentity)
	}
	if err := inclusion.Validate(); err != nil {
		return "", auditd.AuditRecordInclusion{}, err
	}
	return recordIdentity, inclusion, nil
}

func (s *Service) compileZKProofInput(inclusion auditd.AuditRecordInclusion) (zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, zkproof.MerkleAuthenticationPath, error) {
	auditEvent, err := decodeAuditEventPayload(inclusion.RecordEnvelope)
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, zkproof.MerkleAuthenticationPath{}, err
	}
	path, err := zkproof.DeriveAuditSegmentMerkleAuthenticationPathV0(inclusion.SegmentRecordDigests, inclusion.RecordIndex)
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, zkproof.MerkleAuthenticationPath{}, err
	}
	compiled, err := zkproof.CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0(zkproof.CompileAuditIsolateSessionBoundAttestedRuntimeInput{
		DeterministicVerification:        true,
		VerifiedAuditEvent:               auditEvent,
		VerifiedAuditRecordDigest:        inclusion.RecordDigest,
		VerifiedAuditSegmentSeal:         inclusion.SealPayload,
		VerifiedAuditSegmentSealDigest:   inclusion.SealEnvelopeDigest,
		MerkleAuthenticationPath:         path,
		BindingCommitmentDeriver:         deterministicBindingCommitmentDeriver{},
		SessionBindingRelationshipVerify: deterministicSessionBindingRelationshipVerifier{},
		NormalizationProfileID:           zkProofNormalizationProfileV0,
		SchemeAdapterID:                  zkProofSchemeAdapterIDV0,
		ProjectSubstrateSnapshotDigest:   strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest),
	})
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, zkproof.MerkleAuthenticationPath{}, err
	}
	return compiled, path, nil
}

func decodeAuditEventPayload(envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.AuditEventPayload, error) {
	auditEvent := trustpolicy.AuditEventPayload{}
	if err := json.Unmarshal(envelope.Payload, &auditEvent); err != nil {
		return trustpolicy.AuditEventPayload{}, fmt.Errorf("decode audit event payload: %w", err)
	}
	return auditEvent, nil
}

func (s *Service) persistCompiledAuditProofBinding(compiled zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, path zkproof.MerkleAuthenticationPath, inclusion auditd.AuditRecordInclusion) (trustpolicy.Digest, error) {
	bindingPayload := trustpolicy.AuditProofBindingPayload{
		SchemaID:               trustpolicy.AuditProofBindingSchemaID,
		SchemaVersion:          trustpolicy.AuditProofBindingSchemaVersion,
		StatementFamily:        zkProofStatementFamilyV0,
		StatementVersion:       zkProofStatementVersionV0,
		NormalizationProfileID: compiled.PublicInputs.NormalizationProfileID,
		SchemeAdapterID:        compiled.PublicInputs.SchemeAdapterID,
		AuditRecordDigest:      compiled.PublicInputs.AuditRecordDigest,
		AuditSegmentSealDigest: compiled.PublicInputs.AuditSegmentSealDigest,
		MerkleRoot:             compiled.PublicInputs.MerkleRoot,
		ProtocolBundleManifest: compiled.PublicInputs.ProtocolBundleManifestHash,
		BindingCommitment:      compiled.PublicInputs.BindingCommitment,
		ProjectedPublicBindings: trustpolicy.AuditProofBindingProjectedPublicBindings{
			RuntimeImageDescriptorDigest:   compiled.PublicInputs.RuntimeImageDescriptorDigest,
			AttestationEvidenceDigest:      compiled.PublicInputs.AttestationEvidenceDigest,
			AppliedHardeningPostureDigest:  compiled.PublicInputs.AppliedHardeningPostureDigest,
			SessionBindingDigest:           compiled.PublicInputs.SessionBindingDigest,
			ProjectSubstrateSnapshotDigest: strings.TrimSpace(compiled.PublicInputs.ProjectSubstrateSnapshotDigest),
		},
		MerklePathVersion:        path.PathVersion,
		MerkleAuthenticationPath: toTrustpolicyMerklePath(path),
		MerklePathDepth:          len(path.Steps),
		LeafIndex:                inclusion.RecordIndex,
		SourceRefs:               []trustpolicy.ZKProofSourceRef{{SourceFamily: "audit_segment_seal", SourceDigest: inclusion.SealEnvelopeDigest, SourceRole: "seal"}, {SourceFamily: "audit_event", SourceDigest: inclusion.RecordDigest, SourceRole: "event"}},
	}
	bindingDigest, _, err := s.auditLedger.PersistAuditProofBinding(bindingPayload)
	return bindingDigest, err
}

func (s *Service) persistCompiledProofArtifact(compiled zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, bindingDigest trustpolicy.Digest) (trustpolicy.ZKProofArtifactPayload, trustpolicy.Digest, trustpolicy.Digest, error) {
	artifactPayload, publicInputsDigest, err := buildDeterministicZKProofArtifact(compiled, bindingDigest)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	proofDigest, err := s.auditLedger.PersistZKProofArtifact(artifactPayload)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	return artifactPayload, publicInputsDigest, proofDigest, nil
}

func buildZKProofGenerateResponse(requestID string, recordDigest, bindingDigest, proofDigest, verificationDigest trustpolicy.Digest) ZKProofGenerateResponse {
	resp := ZKProofGenerateResponse{SchemaID: "runecode.protocol.v0.ZKProofGenerateResponse", SchemaVersion: "0.1.0", RequestID: requestID, StatementFamily: zkProofStatementFamilyV0, StatementVersion: zkProofStatementVersionV0, NormalizationProfileID: zkProofNormalizationProfileV0, SchemeAdapterID: zkProofSchemeAdapterIDV0, RecordDigest: recordDigest, AuditProofBindingDigest: bindingDigest, ZKProofArtifactDigest: proofDigest, EvaluationGate: zkProofEvaluationGatePass, UserCheckInRequired: true, CheckInNote: zkProofUserCheckInNote}
	resp.ZKProofVerificationDigest = &verificationDigest
	return resp
}

func (s *Service) appendZKProofGenerateAuditEvent(recordIdentity string, resp ZKProofGenerateResponse) {
	_ = s.AppendTrustedAuditEvent("zk_proof_generate", "brokerapi", map[string]any{"statement_family": resp.StatementFamily, "record_digest": recordIdentity, "audit_proof_binding_digest": mustDigestIdentityString(resp.AuditProofBindingDigest), "zk_proof_artifact_digest": mustDigestIdentityString(resp.ZKProofArtifactDigest), "zk_proof_verification_record_digest": mustDigestIdentityString(*resp.ZKProofVerificationDigest), "evaluation_gate": resp.EvaluationGate, "user_check_in_required": resp.UserCheckInRequired})
}

func verifyArtifactPublicInputsDigest(artifact trustpolicy.ZKProofArtifactPayload, _ trustpolicy.Digest) (trustpolicy.Digest, error) {
	recomputedDigest, err := canonicalMapDigest(artifact.PublicInputs)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	if mustDigestIdentityString(artifact.PublicInputsDigest) != mustDigestIdentityString(recomputedDigest) {
		return trustpolicy.Digest{}, &zkproof.FeasibilityError{Code: "invalid_public_inputs_digest", Message: "proof public_inputs_digest does not match canonical public_inputs content"}
	}
	return recomputedDigest, nil
}

func (s *Service) buildVerificationRecord(artifactDigest trustpolicy.Digest, artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) (trustpolicy.ZKProofVerificationRecordPayload, error) {
	outcome, reasons, err := verifyProofArtifactOutcome(artifact, publicInputsDigest)
	if err != nil {
		return trustpolicy.ZKProofVerificationRecordPayload{}, err
	}
	return trustpolicy.ZKProofVerificationRecordPayload{SchemaID: trustpolicy.ZKProofVerificationRecordSchemaID, SchemaVersion: trustpolicy.ZKProofVerificationRecordSchemaVersion, ProofDigest: artifactDigest, StatementFamily: artifact.StatementFamily, StatementVersion: artifact.StatementVersion, SchemeID: artifact.SchemeID, CurveID: artifact.CurveID, CircuitID: artifact.CircuitID, ConstraintSystemDigest: artifact.ConstraintSystemDigest, VerifierKeyDigest: artifact.VerifierKeyDigest, SetupProvenanceDigest: artifact.SetupProvenanceDigest, NormalizationProfileID: artifact.NormalizationProfileID, SchemeAdapterID: artifact.SchemeAdapterID, PublicInputsDigest: artifact.PublicInputsDigest, VerifierImplementationID: zkProofVerifierImplementationID, VerifiedAt: s.now().UTC().Format(time.RFC3339), VerificationOutcome: outcome, ReasonCodes: reasons, CacheProvenance: "fresh"}, nil
}

func verifyProofArtifactOutcome(artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) (string, []string, error) {
	proofBytes, identity, trusted, err := decodeProofVerificationInputs(artifact)
	if err != nil {
		return "", nil, err
	}
	err = zkproof.VerifyProofWithTrustedPostureV0(deterministicProofBackend{}, proofBytes, publicInputsDigest, identity, trusted)
	if err != nil {
		return trustpolicy.ProofVerificationOutcomeRejected, []string{classifyProofVerificationReason(err)}, nil
	}
	return trustpolicy.ProofVerificationOutcomeVerified, []string{trustpolicy.ProofVerificationReasonVerified}, nil
}

func decodeProofVerificationInputs(artifact trustpolicy.ZKProofArtifactPayload) ([]byte, zkproof.ProofVerificationIdentity, zkproof.TrustedVerifierPosture, error) {
	proofBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(artifact.ProofBytes))
	if err != nil {
		return nil, zkproof.ProofVerificationIdentity{}, zkproof.TrustedVerifierPosture{}, fmt.Errorf("decode proof bytes: %w", err)
	}
	identity := zkproof.ProofVerificationIdentity{VerifierKeyDigest: artifact.VerifierKeyDigest, ConstraintSystemDigest: artifact.ConstraintSystemDigest, SetupProvenanceDigest: artifact.SetupProvenanceDigest}
	return proofBytes, identity, trustedVerifierPostureFixtureV0(), nil
}

func (s *Service) findCachedVerificationResponse(requestID string, artifactDigest trustpolicy.Digest, record trustpolicy.ZKProofVerificationRecordPayload) (ZKProofVerifyResponse, bool, error) {
	cachedDigest, cachedRecord, found, err := s.auditLedger.FindMatchingZKProofVerificationRecord(record)
	if err != nil || !found {
		return ZKProofVerifyResponse{}, false, err
	}
	return buildZKProofVerifyResponse(requestID, artifactDigest, cachedDigest, cachedRecord.VerificationOutcome, append([]string{}, cachedRecord.ReasonCodes...), "cache_hit"), true, nil
}

func buildZKProofVerifyResponse(requestID string, artifactDigest, verificationDigest trustpolicy.Digest, outcome string, reasons []string, cacheProvenance string) ZKProofVerifyResponse {
	return ZKProofVerifyResponse{SchemaID: "runecode.protocol.v0.ZKProofVerifyResponse", SchemaVersion: "0.1.0", RequestID: requestID, ZKProofArtifactDigest: artifactDigest, ZKProofVerificationRecordDigest: verificationDigest, VerificationOutcome: outcome, ReasonCodes: reasons, CacheProvenance: cacheProvenance, EvaluationGate: zkProofEvaluationGatePass, UserCheckInRequired: true, CheckInNote: zkProofUserCheckInNote}
}
