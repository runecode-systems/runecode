package zkproof

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	trustedLocalGroth16BackendIdentityV0 = "runecode.trusted.zk.backend.gnark.groth16.bn254.v0"
	setupPhase1LineageIDV0               = "runecode.zk.setup.phase1.local.trusted.groth16.bn254.v0"
	frozenCircuitSourceDescriptorV0      = "runecode.zk.circuit.audit.isolate_session_bound.attested_runtime_membership.v0:poseidon2_fold(private_remainder_fields)+sha256_merkle_membership"
	gnarkModuleVersionV0                 = "github.com/consensys/gnark@v0.14.0"

	bindingCommitmentPrefixV0 = "runecode.zkproof.binding_commitment.poseidon2.v0:"
	merkleLeafPrefixV0        = "runecode.audit.merkle.leaf.v1:"
	merkleNodePrefixV0        = "runecode.audit.merkle.node.v1:"
)

var (
	setupOnceV0      sync.Once
	setupMaterialV0  trustedLocalSetupMaterialV0
	setupMaterialErr error
)

type trustedLocalSetupMaterialV0 struct {
	CS       constraint.ConstraintSystem
	PK       groth16.ProvingKey
	VK       groth16.VerifyingKey
	Frozen   FrozenCircuitIdentity
	Lineage  SetupLineageIdentity
	Identity ProofVerificationIdentity
	Trusted  TrustedVerifierPosture
}

type trustedLocalGroth16BackendV0 struct {
	setup trustedLocalSetupMaterialV0
}

func NewTrustedLocalGroth16BackendV0() (ProofProver, FrozenCircuitIdentity, SetupLineageIdentity, TrustedVerifierPosture, error) {
	setup, err := loadTrustedLocalSetupMaterialV0()
	if err != nil {
		return nil, FrozenCircuitIdentity{}, SetupLineageIdentity{}, TrustedVerifierPosture{}, err
	}
	return trustedLocalGroth16BackendV0{setup: setup}, setup.Frozen, setup.Lineage, setup.Trusted, nil
}

func (b trustedLocalGroth16BackendV0) BackendIdentity() string {
	return trustedLocalGroth16BackendIdentityV0
}

func (b trustedLocalGroth16BackendV0) ProveDeterministic(contract AuditIsolateSessionBoundAttestedRuntimeProofInputContract) ([]byte, ProofVerificationIdentity, error) {
	if err := validateProofInputContractV0(contract); err != nil {
		return nil, ProofVerificationIdentity{}, err
	}
	wantCommitment, err := NewPoseidonBindingCommitmentDeriverV0().DeriveBindingCommitment(contract.PublicInputs.SchemeAdapterID, contract.WitnessInputs.PrivateRemainder)
	if err != nil {
		return nil, ProofVerificationIdentity{}, err
	}
	if wantCommitment != contract.PublicInputs.BindingCommitment {
		return nil, ProofVerificationIdentity{}, &FeasibilityError{Code: feasibilityCodeSessionBindingMismatch, Message: "binding_commitment mismatch against private remainder"}
	}
	if err := VerifyAuditSegmentMerkleAuthenticationPathAgainstSealV0(contract.PublicInputs.AuditRecordDigest, contract.WitnessInputs.MerkleAuthenticationPath, trustpolicy.AuditSegmentSealPayload{SchemaID: trustpolicy.AuditSegmentSealSchemaID, SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1, MerkleRoot: contract.PublicInputs.MerkleRoot}); err != nil {
		return nil, ProofVerificationIdentity{}, err
	}
	assignment, err := buildCircuitWitnessV0(contract)
	if err != nil {
		return nil, ProofVerificationIdentity{}, err
	}
	w, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, ProofVerificationIdentity{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("build witness: %v", err)}
	}
	proof, err := groth16.Prove(b.setup.CS, b.setup.PK, w)
	if err != nil {
		return nil, ProofVerificationIdentity{}, &FeasibilityError{Code: feasibilityCodeProofInvalid, Message: fmt.Sprintf("groth16 prove: %v", err)}
	}
	proofBytes, err := serializeProofV0(proof)
	if err != nil {
		return nil, ProofVerificationIdentity{}, err
	}
	return proofBytes, b.setup.Identity, nil
}

func (b trustedLocalGroth16BackendV0) VerifyDeterministic(proof []byte, publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	if len(proof) == 0 {
		return &FeasibilityError{Code: feasibilityCodeProofInvalid, Message: "proof is required"}
	}
	if err := validateProofInputContractV0(AuditIsolateSessionBoundAttestedRuntimeProofInputContract{PublicInputs: publicInputs}); err != nil {
		return err
	}
	assignment, err := buildCircuitPublicWitnessV0(publicInputs)
	if err != nil {
		return err
	}
	p := groth16.NewProof(ecc.BN254)
	if _, err := p.ReadFrom(bytes.NewReader(proof)); err != nil {
		return &FeasibilityError{Code: feasibilityCodeProofInvalid, Message: fmt.Sprintf("decode proof: %v", err)}
	}
	publicWitness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("build public witness: %v", err)}
	}
	if err := groth16.Verify(p, b.setup.VK, publicWitness); err != nil {
		return &FeasibilityError{Code: feasibilityCodeProofInvalid, Message: fmt.Sprintf("groth16 verify: %v", err)}
	}
	return nil
}

func validateProofInputContractV0(contract AuditIsolateSessionBoundAttestedRuntimeProofInputContract) error {
	if contract.PublicInputs.StatementFamily != StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0 {
		return &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: "statement_family mismatch"}
	}
	if contract.PublicInputs.StatementVersion != StatementVersionV0 {
		return &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: "statement_version mismatch"}
	}
	if contract.PublicInputs.SchemeAdapterID != SchemeAdapterGnarkGroth16IsolateSessionBoundV0 {
		return &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: "scheme_adapter_id mismatch"}
	}
	return nil
}
