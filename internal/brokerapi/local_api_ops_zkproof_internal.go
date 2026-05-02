package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

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

func (v runtimeSessionBindingRelationshipVerifier) VerifyNormalizedPrivateRemainderSessionBinding(normalized zkproof.IsolateSessionBoundPrivateRemainder, sourceSessionBindingDigest string) error {
	if v.evidence.Session == nil || strings.TrimSpace(v.evidence.Session.EvidenceDigest) == "" {
		return fmt.Errorf("runtime session evidence digest missing")
	}
	if strings.TrimSpace(v.evidence.Session.EvidenceDigest) != strings.TrimSpace(sourceSessionBindingDigest) {
		return fmt.Errorf("session_binding_digest mismatch against authoritative runtime evidence")
	}
	if stableIdentifierDigestIdentityV0(v.evidence.Session.RunID) != digestIdentityFromDigestV0(normalized.RunIDDigest) {
		return fmt.Errorf("run_id mismatch against authoritative runtime session evidence")
	}
	if stableIdentifierDigestIdentityV0(v.evidence.Session.IsolateID) != digestIdentityFromDigestV0(normalized.IsolateIDDigest) {
		return fmt.Errorf("isolate_id mismatch against authoritative runtime session evidence")
	}
	if stableIdentifierDigestIdentityV0(v.evidence.Session.SessionID) != digestIdentityFromDigestV0(normalized.SessionIDDigest) {
		return fmt.Errorf("session_id mismatch against authoritative runtime session evidence")
	}
	if strings.TrimSpace(v.evidence.Session.LaunchContextDigest) != digestIdentityFromDigestV0(normalized.LaunchContextDigest) {
		return fmt.Errorf("launch_context_digest mismatch against authoritative runtime session evidence")
	}
	if strings.TrimSpace(v.evidence.Session.HandshakeTranscriptHash) != digestIdentityFromDigestV0(normalized.HandshakeTranscriptHashDigest) {
		return fmt.Errorf("handshake_transcript_hash mismatch against authoritative runtime session evidence")
	}
	if strings.TrimSpace(v.evidence.Launch.BackendKind) != backendKindFromCodeV0(normalized.BackendKindCode) {
		return fmt.Errorf("backend_kind mismatch against authoritative runtime launch evidence")
	}
	if strings.TrimSpace(v.evidence.Launch.IsolationAssuranceLevel) != isolationAssuranceLevelFromCodeV0(normalized.IsolationAssuranceLevelCode) {
		return fmt.Errorf("isolation_assurance_level mismatch against authoritative runtime launch evidence")
	}
	if strings.TrimSpace(v.evidence.Session.ProvisioningPosture) != provisioningPostureFromCodeV0(normalized.ProvisioningPostureCode) {
		return fmt.Errorf("provisioning_posture mismatch against authoritative runtime session evidence")
	}
	return nil
}

func digestIdentityFromDigestV0(d trustpolicy.Digest) string {
	identity, _ := d.Identity()
	return identity
}

func stableIdentifierDigestIdentityV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func backendKindFromCodeV0(code uint16) string {
	switch code {
	case 1:
		return launcherbackend.BackendKindMicroVM
	case 2:
		return launcherbackend.BackendKindContainer
	default:
		return launcherbackend.BackendKindUnknown
	}
}

func isolationAssuranceLevelFromCodeV0(code uint16) string {
	switch code {
	case 1:
		return launcherbackend.IsolationAssuranceIsolated
	case 2:
		return launcherbackend.IsolationAssuranceDegraded
	case 254:
		return launcherbackend.IsolationAssuranceNotApplicable
	default:
		return launcherbackend.IsolationAssuranceUnknown
	}
}

func provisioningPostureFromCodeV0(code uint16) string {
	switch code {
	case 1:
		return launcherbackend.ProvisioningPostureTOFU
	case 2:
		return launcherbackend.ProvisioningPostureAttested
	case 254:
		return launcherbackend.ProvisioningPostureNotApplicable
	default:
		return launcherbackend.ProvisioningPostureUnknown
	}
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

func (s *Service) appendZKProofGenerateAuditEvent(recordIdentity string, resp ZKProofGenerateResponse) error {
	return s.AppendTrustedAuditEvent("zk_proof_generate", "brokerapi", map[string]any{"statement_family": resp.StatementFamily, "record_digest": recordIdentity, "audit_proof_binding_digest": mustDigestIdentityString(resp.AuditProofBindingDigest), "zk_proof_artifact_digest": mustDigestIdentityString(resp.ZKProofArtifactDigest), "zk_proof_verification_record_digest": mustDigestIdentityString(*resp.ZKProofVerificationDigest), "evaluation_gate": resp.EvaluationGate, "user_check_in_required": resp.UserCheckInRequired})
}

func fromTrustpolicyMerklePath(version string, leafIndex int, path []trustpolicy.AuditProofBindingMerkleAuthenticationStep) zkproof.MerkleAuthenticationPath {
	steps := make([]zkproof.MerkleAuthenticationStep, 0, len(path))
	for _, step := range path {
		steps = append(steps, zkproof.MerkleAuthenticationStep{SiblingDigest: step.SiblingDigest, SiblingPosition: step.SiblingPosition})
	}
	return zkproof.MerkleAuthenticationPath{PathVersion: strings.TrimSpace(version), LeafIndex: uint64(leafIndex), Steps: steps}
}
