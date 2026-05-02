package zkproof

import (
	"fmt"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	evaluationOnlyGroth16BackendIdentityV0 = "runecode.evaluation.zk.backend.gnark.groth16.bn254.v0"
	evaluationSetupPhase1LineageIDV0       = "runecode.zk.setup.phase1.local.evaluation.synthetic.groth16.bn254.v0"

	// This digest pins the frozen circuit source descriptor for benchmark-only setup.
	evaluationFrozenCircuitSourceDigestIdentityV0 = "sha256:abf97a7c13c19d5f6236546172691d91d52b8e1c6542e81f8e199bd777baa993"

	// This metadata pin detects circuit-shape drift for evaluation-only setup.
	evaluationCircuitMetadataPinV0 = "sha256:b5f3b5ec4e479e522c42f5ec3212a57d9b4411687c5785e655a659ea8537a23f"
)

var (
	evaluationSetupOnceV0     sync.Once
	evaluationSetupMaterialV0 trustedLocalSetupMaterialV0
	evaluationSetupErrV0      error
)

type evaluationOnlyGroth16BackendV0 struct {
	inner trustedLocalGroth16BackendV0
}

// NewEvaluationOnlyGroth16BackendForBenchmarkV0 constructs a strictly
// non-authoritative backend for local tests and benchmarks. This constructor
// must not be wired into authoritative broker command surfaces.
func NewEvaluationOnlyGroth16BackendForBenchmarkV0() (ProofProver, FrozenCircuitIdentity, SetupLineageIdentity, TrustedVerifierPosture, error) {
	setup, err := loadEvaluationBenchmarkSetupMaterialV0()
	if err != nil {
		return nil, FrozenCircuitIdentity{}, SetupLineageIdentity{}, TrustedVerifierPosture{}, err
	}
	return evaluationOnlyGroth16BackendV0{inner: trustedLocalGroth16BackendV0{setup: setup}}, setup.Frozen, setup.Lineage, setup.Trusted, nil
}

func (b evaluationOnlyGroth16BackendV0) BackendIdentity() string {
	return evaluationOnlyGroth16BackendIdentityV0
}

func (b evaluationOnlyGroth16BackendV0) ProveDeterministic(contract AuditIsolateSessionBoundAttestedRuntimeProofInputContract) ([]byte, ProofVerificationIdentity, error) {
	return b.inner.ProveDeterministic(contract)
}

func (b evaluationOnlyGroth16BackendV0) VerifyDeterministic(proof []byte, publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) error {
	return b.inner.VerifyDeterministic(proof, publicInputs)
}

func loadEvaluationBenchmarkSetupMaterialV0() (trustedLocalSetupMaterialV0, error) {
	evaluationSetupOnceV0.Do(func() {
		evaluationSetupMaterialV0, evaluationSetupErrV0 = buildEvaluationBenchmarkSetupMaterialV0()
	})
	return evaluationSetupMaterialV0, evaluationSetupErrV0
}

func newEvaluationOnlyGroth16BackendFreshV0() (trustedLocalSetupMaterialV0, error) {
	return buildEvaluationBenchmarkSetupMaterialV0()
}

func buildEvaluationBenchmarkSetupMaterialV0() (trustedLocalSetupMaterialV0, error) {
	frozenSourceDigest, err := validateEvaluationFrozenCircuitSourceV0()
	if err != nil {
		return trustedLocalSetupMaterialV0{}, err
	}
	cs, csDigest, err := compileAndValidateEvaluationCircuitV0()
	if err != nil {
		return trustedLocalSetupMaterialV0{}, err
	}
	pk, vk, verifierKeyDigest, err := runEvaluationGroth16SetupV0(cs)
	if err != nil {
		return trustedLocalSetupMaterialV0{}, err
	}
	return buildEvaluationSetupMaterialV0(frozenSourceDigest, cs, csDigest, pk, vk, verifierKeyDigest)
}

func validateEvaluationFrozenCircuitSourceV0() (trustpolicy.Digest, error) {
	frozenSourceDigest, err := parseDigestIdentity(evaluationFrozenCircuitSourceDigestIdentityV0, "evaluation frozen_circuit_source_digest")
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	computedFrozenSourceDigest := sha256DigestFromBytesV0([]byte(frozenCircuitSourceDescriptorV0))
	computedFrozenSourceIdentity, err := computedFrozenSourceDigest.Identity()
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	if computedFrozenSourceIdentity != evaluationFrozenCircuitSourceDigestIdentityV0 {
		return trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeSetupIdentityMismatch, Message: fmt.Sprintf("evaluation frozen_circuit_source_digest drift: got %q want %q", computedFrozenSourceIdentity, evaluationFrozenCircuitSourceDigestIdentityV0)}
	}
	return frozenSourceDigest, nil
}

func compileAndValidateEvaluationCircuitV0() (constraint.ConstraintSystem, trustpolicy.Digest, error) {
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &auditIsolateSessionBoundCircuitV0{})
	if err != nil {
		return nil, trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeUnconfiguredProofBackend, Message: fmt.Sprintf("evaluation-only compile circuit: %v", err)}
	}
	if err := validateEvaluationCircuitMetadataPinV0(cs); err != nil {
		return nil, trustpolicy.Digest{}, err
	}
	csDigest, err := hashConstraintSystemV0(cs)
	if err != nil {
		return nil, trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeUnconfiguredProofBackend, Message: fmt.Sprintf("evaluation-only hash constraint system: %v", err)}
	}
	return cs, csDigest, nil
}

func validateEvaluationCircuitMetadataPinV0(cs constraint.ConstraintSystem) error {
	metadataDigest := evaluationCircuitMetadataDigestV0(cs)
	metadataIdentity, err := metadataDigest.Identity()
	if err != nil {
		return err
	}
	if metadataIdentity != evaluationCircuitMetadataPinV0 {
		return &FeasibilityError{Code: feasibilityCodeSetupIdentityMismatch, Message: fmt.Sprintf("evaluation circuit metadata drift: got %q want %q", metadataIdentity, evaluationCircuitMetadataPinV0)}
	}
	return nil
}

func runEvaluationGroth16SetupV0(cs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, trustpolicy.Digest, error) {
	pk, vk, err := groth16.Setup(cs)
	if err != nil {
		return nil, nil, trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeUnconfiguredProofBackend, Message: fmt.Sprintf("evaluation-only groth16 setup: %v", err)}
	}
	verifierKeyDigest, err := hashVerifyingKeyV0(vk)
	if err != nil {
		return nil, nil, trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeUnconfiguredProofBackend, Message: fmt.Sprintf("evaluation-only hash verifying key: %v", err)}
	}
	return pk, vk, verifierKeyDigest, nil
}

func buildEvaluationSetupMaterialV0(frozenSourceDigest trustpolicy.Digest, cs constraint.ConstraintSystem, csDigest trustpolicy.Digest, pk groth16.ProvingKey, vk groth16.VerifyingKey, verifierKeyDigest trustpolicy.Digest) (trustedLocalSetupMaterialV0, error) {

	lineage := SetupLineageIdentity{
		Phase1LineageID:        evaluationSetupPhase1LineageIDV0,
		Phase1LineageDigest:    syntheticEvaluationDigestV0("phase1"),
		Phase2TranscriptDigest: syntheticEvaluationDigestV0("phase2"),
		FrozenCircuitSourceDig: frozenSourceDigest,
		ConstraintSystemDigest: csDigest,
		GnarkModuleVersion:     gnarkModuleVersionV0,
	}
	setupProvenance, err := CanonicalSetupProvenanceDigestV0(lineage)
	if err != nil {
		return trustedLocalSetupMaterialV0{}, err
	}
	frozen := FrozenCircuitIdentity{
		SchemeID:               ProofSchemeIDGroth16V0,
		CurveID:                ProofCurveIDBN254V0,
		CircuitID:              CircuitIDAuditIsolateSessionBoundAttestedRuntimeMembershipV0,
		ConstraintSystemDigest: csDigest,
	}
	identity := ProofVerificationIdentity{
		VerifierKeyDigest:      verifierKeyDigest,
		ConstraintSystemDigest: csDigest,
		SetupProvenanceDigest:  setupProvenance,
	}
	trusted := TrustedVerifierPosture(identity)
	return trustedLocalSetupMaterialV0{CS: cs, PK: pk, VK: vk, Frozen: frozen, Lineage: lineage, Identity: identity, Trusted: trusted}, nil
}

func syntheticEvaluationDigestV0(label string) trustpolicy.Digest {
	return sha256DigestFromBytesV0([]byte("runecode.zkproof.evaluation.synthetic.v0:" + label))
}

func evaluationCircuitMetadataDigestV0(cs constraint.ConstraintSystem) trustpolicy.Digest {
	shape := fmt.Sprintf("public=%d|secret=%d|internal=%d|constraints=%d|coefficients=%d|instructions=%d", cs.GetNbPublicVariables(), cs.GetNbSecretVariables(), cs.GetNbInternalVariables(), cs.GetNbConstraints(), cs.GetNbCoefficients(), cs.GetNbInstructions())
	return sha256DigestFromBytesV0([]byte("runecode.zkproof.evaluation.circuit_shape.v0:" + shape))
}
