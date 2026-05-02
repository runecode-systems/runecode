package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

func (s *Service) requireZKProofBackend(operation string) error {
	if s.auditLedger == nil {
		return fmt.Errorf("audit ledger unavailable")
	}
	if s.store == nil {
		return fmt.Errorf("artifact store unavailable")
	}
	if _, _, _, _, err := zkproof.NewTrustedLocalGroth16BackendV0(); err == nil {
		return nil
	}
	return &zkproof.FeasibilityError{Code: "unconfigured_proof_backend", Message: fmt.Sprintf("zk proof %s requires trusted local proof backend availability", operation)}
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

func (s *Service) compileZKProofInput(inclusion auditd.AuditRecordInclusion) (zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, zkproof.MerkleAuthenticationPath, launcherbackend.RuntimeEvidenceSnapshot, error) {
	auditEvent, err := decodeAuditEventPayload(inclusion.RecordEnvelope)
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, zkproof.MerkleAuthenticationPath{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	runtimeEvidence, err := s.runtimeEvidenceForAuditEvent(auditEvent)
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, zkproof.MerkleAuthenticationPath{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	path, err := zkproof.DeriveAuditSegmentMerkleAuthenticationPathV0(inclusion.SegmentRecordDigests, inclusion.RecordIndex)
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, zkproof.MerkleAuthenticationPath{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	compiled, err := zkproof.CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0(zkproof.CompileAuditIsolateSessionBoundAttestedRuntimeInput{
		DeterministicVerification:        true,
		VerifiedAuditEvent:               auditEvent,
		VerifiedAuditRecordDigest:        inclusion.RecordDigest,
		VerifiedAuditSegmentSeal:         inclusion.SealPayload,
		VerifiedAuditSegmentSealDigest:   inclusion.SealEnvelopeDigest,
		MerkleAuthenticationPath:         path,
		BindingCommitmentDeriver:         zkproof.NewPoseidonBindingCommitmentDeriverV0(),
		SessionBindingRelationshipVerify: runtimeSessionBindingRelationshipVerifier{evidence: runtimeEvidence},
		NormalizationProfileID:           zkProofNormalizationProfileV0,
		SchemeAdapterID:                  zkProofSchemeAdapterIDV0,
		ProjectSubstrateSnapshotDigest:   strings.TrimSpace(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest),
	})
	if err != nil {
		return zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract{}, zkproof.MerkleAuthenticationPath{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	return compiled, path, runtimeEvidence, nil
}

func (s *Service) runtimeEvidenceForAuditEvent(auditEvent trustpolicy.AuditEventPayload) (launcherbackend.RuntimeEvidenceSnapshot, error) {
	payload, err := decodeEligibleAuditEventForRuntimeEvidence(auditEvent)
	if err != nil {
		return launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	if s.store == nil {
		return launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("artifact store unavailable")
	}
	_, evidence, _, _, ok := s.store.RuntimeEvidenceState(payload.RunID)
	if !ok {
		return launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("runtime evidence for run %q not found", payload.RunID)
	}
	if evidence.Session == nil || strings.TrimSpace(evidence.Session.EvidenceDigest) == "" {
		return launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("runtime session evidence missing for run %q", payload.RunID)
	}
	if strings.TrimSpace(evidence.Session.EvidenceDigest) != strings.TrimSpace(payload.SessionBindingDigest) {
		return launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("runtime session evidence digest does not match audited session_binding_digest")
	}
	if evidence.AttestationVerification == nil || strings.TrimSpace(evidence.AttestationVerification.VerificationDigest) == "" {
		return launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("attestation verification record missing for run %q", payload.RunID)
	}
	if evidence.AttestationVerification.VerificationResult != launcherbackend.AttestationVerificationResultValid {
		return launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("attestation verification result for run %q is not valid", payload.RunID)
	}
	if evidence.AttestationVerification.ReplayVerdict != launcherbackend.AttestationReplayVerdictOriginal {
		return launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("attestation replay verdict for run %q is not original", payload.RunID)
	}
	return evidence, nil
}

func decodeEligibleAuditEventForRuntimeEvidence(event trustpolicy.AuditEventPayload) (trustpolicy.IsolateSessionBoundPayload, error) {
	payload := trustpolicy.IsolateSessionBoundPayload{}
	if strings.TrimSpace(event.AuditEventType) != "isolate_session_bound" {
		return payload, fmt.Errorf("audit_event_type must be isolate_session_bound")
	}
	if err := json.Unmarshal(event.EventPayload, &payload); err != nil {
		return payload, fmt.Errorf("decode isolate_session_bound payload: %w", err)
	}
	return payload, nil
}

type runtimeSessionBindingRelationshipVerifier struct {
	evidence launcherbackend.RuntimeEvidenceSnapshot
}

func (v runtimeSessionBindingRelationshipVerifier) VerifyNormalizedPrivateRemainderSessionBinding(_ zkproof.IsolateSessionBoundPrivateRemainder, sourceSessionBindingDigest string) error {
	if v.evidence.Session == nil || strings.TrimSpace(v.evidence.Session.EvidenceDigest) == "" {
		return fmt.Errorf("runtime session evidence digest missing")
	}
	if strings.TrimSpace(v.evidence.Session.EvidenceDigest) != strings.TrimSpace(sourceSessionBindingDigest) {
		return fmt.Errorf("session_binding_digest mismatch against authoritative runtime evidence")
	}
	return nil
}

func decodeAuditEventPayload(envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.AuditEventPayload, error) {
	auditEvent := trustpolicy.AuditEventPayload{}
	if err := json.Unmarshal(envelope.Payload, &auditEvent); err != nil {
		return trustpolicy.AuditEventPayload{}, fmt.Errorf("decode audit event payload: %w", err)
	}
	return auditEvent, nil
}

func (s *Service) persistCompiledAuditProofBinding(compiled zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, path zkproof.MerkleAuthenticationPath, inclusion auditd.AuditRecordInclusion, runtimeEvidence launcherbackend.RuntimeEvidenceSnapshot) (trustpolicy.Digest, error) {
	var attestationVerificationDigest *trustpolicy.Digest
	if runtimeEvidence.AttestationVerification != nil && strings.TrimSpace(runtimeEvidence.AttestationVerification.VerificationDigest) != "" {
		digest, err := parseDigestIdentityString(runtimeEvidence.AttestationVerification.VerificationDigest)
		if err != nil {
			return trustpolicy.Digest{}, err
		}
		attestationVerificationDigest = &digest
	}
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
			AttestationVerificationRecord:  attestationVerificationDigest,
		},
		MerklePathVersion:        path.PathVersion,
		MerkleAuthenticationPath: toTrustpolicyMerklePath(path),
		MerklePathDepth:          len(path.Steps),
		LeafIndex:                inclusion.RecordIndex,
		SourceRefs:               bindingSourceRefs(inclusion, attestationVerificationDigest),
	}
	bindingDigest, _, err := s.auditLedger.PersistAuditProofBinding(bindingPayload)
	return bindingDigest, err
}

func (s *Service) persistCompiledProofArtifact(compiled zkproof.AuditIsolateSessionBoundAttestedRuntimeProofInputContract, bindingDigest trustpolicy.Digest) (trustpolicy.ZKProofArtifactPayload, error) {
	backend, frozen, _, trusted, err := zkproof.NewTrustedLocalGroth16BackendV0()
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, err
	}
	artifactPayload, err := buildZKProofArtifact(compiled, bindingDigest, backend, frozen, trusted)
	if err != nil {
		return trustpolicy.ZKProofArtifactPayload{}, err
	}
	return artifactPayload, nil
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
