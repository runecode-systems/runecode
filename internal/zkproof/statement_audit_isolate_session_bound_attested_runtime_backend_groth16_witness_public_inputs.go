package zkproof

import "github.com/consensys/gnark/frontend"

type circuitPublicDigestBytesV0 struct {
	PublicInputsDigest            [32]frontend.Variable
	AuditSegmentSealDigest        [32]frontend.Variable
	BindingCommitment             [32]frontend.Variable
	MerkleRoot                    [32]frontend.Variable
	AuditRecordDigest             [32]frontend.Variable
	ProtocolBundleManifestHash    [32]frontend.Variable
	RuntimeImageDescriptorDigest  [32]frontend.Variable
	AttestationEvidenceDigest     [32]frontend.Variable
	AppliedHardeningPostureDigest [32]frontend.Variable
	SessionBindingDigest          [32]frontend.Variable
}

func buildCircuitPublicWitnessV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) (auditIsolateSessionBoundCircuitV0, error) {
	bytes, err := buildCircuitPublicDigestBytesV0(publicInputs)
	if err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	return auditIsolateSessionBoundCircuitV0{
		PublicInputsDigest:            bytes.PublicInputsDigest,
		AuditSegmentSealDigest:        bytes.AuditSegmentSealDigest,
		BindingCommitment:             bytes.BindingCommitment,
		MerkleRoot:                    bytes.MerkleRoot,
		AuditRecordDigest:             bytes.AuditRecordDigest,
		ProtocolBundleManifestHash:    bytes.ProtocolBundleManifestHash,
		RuntimeImageDescriptorDigest:  bytes.RuntimeImageDescriptorDigest,
		AttestationEvidenceDigest:     bytes.AttestationEvidenceDigest,
		AppliedHardeningPostureDigest: bytes.AppliedHardeningPostureDigest,
		SessionBindingDigest:          bytes.SessionBindingDigest,
	}, nil
}

func buildCircuitPublicDigestBytesV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) (circuitPublicDigestBytesV0, error) {
	publicInputsDigestBytes, err := digestToPublicByteArrayV0(publicInputs.PublicInputsDigest, "public_inputs_digest")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	auditSegmentSealBytes, err := digestToPublicByteArrayV0(publicInputs.AuditSegmentSealDigest, "audit_segment_seal_digest")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	bindingBytes, err := buildBindingCommitmentBytesV0(publicInputs.BindingCommitment)
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	rest, err := buildRemainingCircuitPublicDigestBytesV0(publicInputs)
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	rest.PublicInputsDigest = publicInputsDigestBytes
	rest.AuditSegmentSealDigest = auditSegmentSealBytes
	rest.BindingCommitment = bindingBytes
	return rest, nil
}

func buildBindingCommitmentBytesV0(identity string) ([32]frontend.Variable, error) {
	bindingDigest, err := parseDigestIdentity(identity, "binding_commitment")
	if err != nil {
		return [32]frontend.Variable{}, err
	}
	return digestToPublicByteArrayV0(bindingDigest, "binding_commitment")
}

func buildRemainingCircuitPublicDigestBytesV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) (circuitPublicDigestBytesV0, error) {
	merkleRootBytes, err := digestToPublicByteArrayV0(publicInputs.MerkleRoot, "merkle_root")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	auditRecordBytes, err := digestToPublicByteArrayV0(publicInputs.AuditRecordDigest, "audit_record_digest")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	protocolBundleBytes, err := digestToPublicByteArrayV0(publicInputs.ProtocolBundleManifestHash, "protocol_bundle_manifest_hash")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	runtimeImageBytes, err := digestIdentityStringToPublicByteArrayV0(publicInputs.RuntimeImageDescriptorDigest, "runtime_image_descriptor_digest")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	attestationEvidenceBytes, err := digestIdentityStringToPublicByteArrayV0(publicInputs.AttestationEvidenceDigest, "attestation_evidence_digest")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	appliedHardeningBytes, err := digestIdentityStringToPublicByteArrayV0(publicInputs.AppliedHardeningPostureDigest, "applied_hardening_posture_digest")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	sessionBindingBytes, err := digestIdentityStringToPublicByteArrayV0(publicInputs.SessionBindingDigest, "session_binding_digest")
	if err != nil {
		return circuitPublicDigestBytesV0{}, err
	}
	return circuitPublicDigestBytesV0{
		MerkleRoot:                    merkleRootBytes,
		AuditRecordDigest:             auditRecordBytes,
		ProtocolBundleManifestHash:    protocolBundleBytes,
		RuntimeImageDescriptorDigest:  runtimeImageBytes,
		AttestationEvidenceDigest:     attestationEvidenceBytes,
		AppliedHardeningPostureDigest: appliedHardeningBytes,
		SessionBindingDigest:          sessionBindingBytes,
	}, nil
}
