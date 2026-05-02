package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

func (s *Service) validateArtifactBindingAndAuthoritativeEvidence(artifact trustpolicy.ZKProofArtifactPayload, publicInputsDigest trustpolicy.Digest) error {
	publicInputs, err := decodeArtifactPublicInputs(artifact.PublicInputs, publicInputsDigest)
	if err != nil {
		return err
	}
	bindingDigest, err := bindingDigestFromArtifactSourceRefs(artifact.SourceRefs)
	if err != nil {
		return err
	}
	bindingPayload, found, err := s.auditLedger.AuditProofBindingByDigest(bindingDigest)
	if err != nil {
		return err
	}
	if !found {
		identity, _ := bindingDigest.Identity()
		return fmt.Errorf("referenced audit proof binding %q not found", identity)
	}
	if err := validateBindingAgainstPublicInputs(bindingPayload, publicInputs); err != nil {
		return err
	}
	return s.validateBindingAuthoritativeSourceEvidence(bindingPayload)
}

func bindingDigestFromArtifactSourceRefs(refs []trustpolicy.ZKProofSourceRef) (trustpolicy.Digest, error) {
	for i := range refs {
		ref := refs[i]
		if strings.TrimSpace(ref.SourceFamily) == "audit_proof_binding" && strings.TrimSpace(ref.SourceRole) == "binding" {
			if _, err := ref.SourceDigest.Identity(); err != nil {
				return trustpolicy.Digest{}, fmt.Errorf("source_refs[%d].source_digest: %w", i, err)
			}
			return ref.SourceDigest, nil
		}
	}
	return trustpolicy.Digest{}, fmt.Errorf("artifact source_refs missing audit_proof_binding binding reference")
}

func validateBindingAgainstPublicInputs(binding trustpolicy.AuditProofBindingPayload, publicInputs zkproof.AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	if strings.TrimSpace(binding.StatementFamily) != strings.TrimSpace(publicInputs.StatementFamily) {
		return fmt.Errorf("binding statement_family does not match proof public inputs")
	}
	if strings.TrimSpace(binding.StatementVersion) != strings.TrimSpace(publicInputs.StatementVersion) {
		return fmt.Errorf("binding statement_version does not match proof public inputs")
	}
	if strings.TrimSpace(binding.NormalizationProfileID) != strings.TrimSpace(publicInputs.NormalizationProfileID) {
		return fmt.Errorf("binding normalization_profile_id does not match proof public inputs")
	}
	if strings.TrimSpace(binding.SchemeAdapterID) != strings.TrimSpace(publicInputs.SchemeAdapterID) {
		return fmt.Errorf("binding scheme_adapter_id does not match proof public inputs")
	}
	if binding.AuditRecordDigest != publicInputs.AuditRecordDigest || binding.AuditSegmentSealDigest != publicInputs.AuditSegmentSealDigest || binding.MerkleRoot != publicInputs.MerkleRoot || binding.ProtocolBundleManifest != publicInputs.ProtocolBundleManifestHash {
		return fmt.Errorf("binding digest fields do not match proof public inputs")
	}
	if strings.TrimSpace(binding.BindingCommitment) != strings.TrimSpace(publicInputs.BindingCommitment) {
		return fmt.Errorf("binding_commitment does not match proof public inputs")
	}
	projected := binding.ProjectedPublicBindings
	if strings.TrimSpace(projected.RuntimeImageDescriptorDigest) != strings.TrimSpace(publicInputs.RuntimeImageDescriptorDigest) || strings.TrimSpace(projected.AttestationEvidenceDigest) != strings.TrimSpace(publicInputs.AttestationEvidenceDigest) || strings.TrimSpace(projected.AppliedHardeningPostureDigest) != strings.TrimSpace(publicInputs.AppliedHardeningPostureDigest) || strings.TrimSpace(projected.SessionBindingDigest) != strings.TrimSpace(publicInputs.SessionBindingDigest) || strings.TrimSpace(projected.ProjectSubstrateSnapshotDigest) != strings.TrimSpace(publicInputs.ProjectSubstrateSnapshotDigest) {
		return fmt.Errorf("binding projected_public_bindings do not match proof public inputs")
	}
	return nil
}

func (s *Service) validateBindingAuthoritativeSourceEvidence(binding trustpolicy.AuditProofBindingPayload) error {
	inclusion, auditEvent, payload, evidence, err := s.loadBindingAuthoritativeEvidence(binding)
	if err != nil {
		return err
	}
	if err := validateBindingLedgerReferences(binding, inclusion); err != nil {
		return err
	}
	if err := validateBindingMerkleAuthentication(binding, inclusion); err != nil {
		return err
	}
	if err := validateBindingProjectedRuntimeEvidence(binding, auditEvent, payload, evidence); err != nil {
		return err
	}
	return validateBindingAttestationVerification(binding, evidence)
}

func validateBindingLedgerReferences(binding trustpolicy.AuditProofBindingPayload, inclusion auditd.AuditRecordInclusion) error {
	if inclusion.RecordDigest != binding.AuditRecordDigest || inclusion.SealEnvelopeDigest != binding.AuditSegmentSealDigest || inclusion.SealPayload.MerkleRoot != binding.MerkleRoot {
		return fmt.Errorf("binding audit/seal references do not match authoritative ledger evidence")
	}
	return nil
}

func validateBindingMerkleAuthentication(binding trustpolicy.AuditProofBindingPayload, inclusion auditd.AuditRecordInclusion) error {
	return zkproof.VerifyAuditSegmentMerkleAuthenticationPathAgainstSealV0(binding.AuditRecordDigest, fromTrustpolicyMerklePath(binding.MerklePathVersion, binding.LeafIndex, binding.MerkleAuthenticationPath), inclusion.SealPayload)
}

func (s *Service) loadBindingAuthoritativeEvidence(binding trustpolicy.AuditProofBindingPayload) (auditd.AuditRecordInclusion, trustpolicy.AuditEventPayload, trustpolicy.IsolateSessionBoundPayload, launcherbackend.RuntimeEvidenceSnapshot, error) {
	recordIdentity, err := binding.AuditRecordDigest.Identity()
	if err != nil {
		return auditd.AuditRecordInclusion{}, trustpolicy.AuditEventPayload{}, trustpolicy.IsolateSessionBoundPayload{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	inclusion, found, err := s.auditLedger.AuditRecordInclusion(recordIdentity)
	if err != nil {
		return auditd.AuditRecordInclusion{}, trustpolicy.AuditEventPayload{}, trustpolicy.IsolateSessionBoundPayload{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	if !found {
		return auditd.AuditRecordInclusion{}, trustpolicy.AuditEventPayload{}, trustpolicy.IsolateSessionBoundPayload{}, launcherbackend.RuntimeEvidenceSnapshot{}, fmt.Errorf("authoritative audit record inclusion missing for binding audit_record_digest")
	}
	auditEvent, err := decodeAuditEventPayload(inclusion.RecordEnvelope)
	if err != nil {
		return auditd.AuditRecordInclusion{}, trustpolicy.AuditEventPayload{}, trustpolicy.IsolateSessionBoundPayload{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	payload, err := decodeEligibleAuditEventForRuntimeEvidence(auditEvent)
	if err != nil {
		return auditd.AuditRecordInclusion{}, trustpolicy.AuditEventPayload{}, trustpolicy.IsolateSessionBoundPayload{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	evidence, err := s.runtimeEvidenceForAuditEvent(auditEvent)
	if err != nil {
		return auditd.AuditRecordInclusion{}, trustpolicy.AuditEventPayload{}, trustpolicy.IsolateSessionBoundPayload{}, launcherbackend.RuntimeEvidenceSnapshot{}, err
	}
	return inclusion, auditEvent, payload, evidence, nil
}

func validateBindingProjectedRuntimeEvidence(binding trustpolicy.AuditProofBindingPayload, auditEvent trustpolicy.AuditEventPayload, payload trustpolicy.IsolateSessionBoundPayload, evidence launcherbackend.RuntimeEvidenceSnapshot) error {
	if evidence.Attestation == nil {
		return fmt.Errorf("authoritative runtime attestation evidence is required for binding validation")
	}
	if strings.TrimSpace(evidence.Attestation.EvidenceDigest) != strings.TrimSpace(binding.ProjectedPublicBindings.AttestationEvidenceDigest) {
		return fmt.Errorf("binding attestation_evidence_digest does not match authoritative runtime evidence")
	}
	if strings.TrimSpace(evidence.Hardening.EvidenceDigest) != strings.TrimSpace(binding.ProjectedPublicBindings.AppliedHardeningPostureDigest) {
		return fmt.Errorf("binding applied_hardening_posture_digest does not match authoritative runtime evidence")
	}
	if strings.TrimSpace(evidence.Launch.RuntimeImageDescriptorDigest) != strings.TrimSpace(binding.ProjectedPublicBindings.RuntimeImageDescriptorDigest) {
		return fmt.Errorf("binding runtime_image_descriptor_digest does not match authoritative runtime evidence")
	}
	if auditEvent.ProtocolBundleManifestHash != binding.ProtocolBundleManifest {
		return fmt.Errorf("binding protocol_bundle_manifest_hash does not match authoritative audit payload")
	}
	if strings.TrimSpace(payload.SessionBindingDigest) != strings.TrimSpace(binding.ProjectedPublicBindings.SessionBindingDigest) {
		return fmt.Errorf("binding session_binding_digest does not match authoritative audit payload")
	}
	if strings.TrimSpace(binding.ProjectedPublicBindings.ProjectSubstrateSnapshotDigest) != "" {
		return fmt.Errorf("binding project_substrate_snapshot_digest is unsupported for the v0 statement family")
	}
	return nil
}

func validateBindingAttestationVerification(binding trustpolicy.AuditProofBindingPayload, evidence launcherbackend.RuntimeEvidenceSnapshot) error {
	attestationDigest := binding.ProjectedPublicBindings.AttestationVerificationRecord
	if attestationDigest == nil {
		return nil
	}
	if evidence.AttestationVerification == nil {
		return fmt.Errorf("binding requires attestation verification record but runtime evidence missing attestation verification")
	}
	if strings.TrimSpace(evidence.AttestationVerification.VerificationDigest) != mustDigestIdentityString(*attestationDigest) {
		return fmt.Errorf("binding attestation_verification_record_digest does not match authoritative runtime evidence")
	}
	return nil
}
