package zkproof

import (
	"errors"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

func TestTrustedLocalGroth16BackendV0FailsClosedUntilReviewedSetupAssetsExist(t *testing.T) {
	_, _, _, _, err := NewTrustedLocalGroth16BackendV0()
	if err == nil {
		t.Fatal("NewTrustedLocalGroth16BackendV0 expected fail-closed error")
	}
	var feasibility *FeasibilityError
	if !errors.As(err, &feasibility) {
		t.Fatalf("error type = %T, want *FeasibilityError", err)
	}
	if feasibility.Code != feasibilityCodeUnconfiguredProofBackend {
		t.Fatalf("feasibility code = %q, want %q", feasibility.Code, feasibilityCodeUnconfiguredProofBackend)
	}
}

func TestEvaluationOnlyGroth16BackendForBenchmarkV0EndToEnd(t *testing.T) {
	contract := mustEvaluationProofContractFixture(t)
	backend, frozen, lineage, trusted, err := NewEvaluationOnlyGroth16BackendForBenchmarkV0()
	if err != nil {
		t.Fatalf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %v", err)
	}
	if backend.BackendIdentity() != evaluationOnlyGroth16BackendIdentityV0 {
		t.Fatalf("backend identity = %q, want %q", backend.BackendIdentity(), evaluationOnlyGroth16BackendIdentityV0)
	}
	if err := frozen.ValidateV0(); err != nil {
		t.Fatalf("frozen.ValidateV0 returned error: %v", err)
	}
	if got, err := lineage.ConstraintSystemDigest.Identity(); err != nil || got == "" {
		t.Fatalf("lineage.constraint_system_digest identity invalid: %q err=%v", got, err)
	}
	proof, identity, err := backend.ProveDeterministic(contract)
	if err != nil {
		t.Fatalf("ProveDeterministic returned error: %v", err)
	}
	if len(proof) == 0 {
		t.Fatal("proof bytes are empty")
	}
	if err := VerifySetupIdentityMatchesTrustedPostureV0(identity, trusted); err != nil {
		t.Fatalf("VerifySetupIdentityMatchesTrustedPostureV0 returned error: %v", err)
	}
	if err := VerifyProofWithTrustedPostureV0(backend, proof, contract.PublicInputs, identity, trusted); err != nil {
		t.Fatalf("VerifyProofWithTrustedPostureV0 returned error: %v", err)
	}
}

func TestEvaluationOnlyGroth16BackendForBenchmarkV0PinnedDigests(t *testing.T) {
	setup, err := newEvaluationOnlyGroth16BackendFreshV0()
	if err != nil {
		t.Fatalf("newEvaluationOnlyGroth16BackendFreshV0 returned error: %v", err)
	}
	constraintIdentity, err := setup.Frozen.ConstraintSystemDigest.Identity()
	if err != nil {
		t.Fatalf("constraint digest identity: %v", err)
	}
	lineageConstraintIdentity, err := setup.Lineage.ConstraintSystemDigest.Identity()
	if err != nil {
		t.Fatalf("lineage constraint digest identity: %v", err)
	}
	if constraintIdentity != lineageConstraintIdentity {
		t.Fatalf("constraint digest mismatch frozen=%q lineage=%q", constraintIdentity, lineageConstraintIdentity)
	}
	cs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &auditIsolateSessionBoundCircuitV0{})
	if err != nil {
		t.Fatalf("compile circuit for metadata pin: %v", err)
	}
	metadataIdentity, err := evaluationCircuitMetadataDigestV0(cs).Identity()
	if err != nil {
		t.Fatalf("metadata digest identity: %v", err)
	}
	if metadataIdentity != evaluationCircuitMetadataPinV0 {
		t.Fatalf("circuit metadata pin mismatch: got %q want %q", metadataIdentity, evaluationCircuitMetadataPinV0)
	}
	frozenSourceDigest := sha256DigestFromBytesV0([]byte(frozenCircuitSourceDescriptorV0))
	frozenSourceIdentity, err := frozenSourceDigest.Identity()
	if err != nil {
		t.Fatalf("frozen source digest identity: %v", err)
	}
	if frozenSourceIdentity != evaluationFrozenCircuitSourceDigestIdentityV0 {
		t.Fatalf("frozen_circuit_source_digest pin mismatch: got %q want %q", frozenSourceIdentity, evaluationFrozenCircuitSourceDigestIdentityV0)
	}
}

func TestEvaluationOnlyGroth16BackendForBenchmarkV0RejectsTamperedTypedPublicInputs(t *testing.T) {
	contract := mustEvaluationProofContractFixture(t)
	backend, _, _, trusted, err := NewEvaluationOnlyGroth16BackendForBenchmarkV0()
	if err != nil {
		t.Fatalf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %v", err)
	}
	proof, identity, err := backend.ProveDeterministic(contract)
	if err != nil {
		t.Fatalf("ProveDeterministic returned error: %v", err)
	}
	tampered := contract.PublicInputs
	tampered.SessionBindingDigest = digestIdentityFixture("3")
	err = VerifyProofWithTrustedPostureV0(backend, proof, tampered, identity, trusted)
	if err == nil {
		t.Fatal("expected typed public input tamper rejection, got nil")
	}
	var feasibility *FeasibilityError
	if !errors.As(err, &feasibility) {
		t.Fatalf("error type = %T, want *FeasibilityError", err)
	}
	if feasibility.Code != "invalid_public_inputs_digest" {
		t.Fatalf("feasibility code = %q, want invalid_public_inputs_digest", feasibility.Code)
	}
}

func mustEvaluationProofContractFixture(t testing.TB) AuditIsolateSessionBoundAttestedRuntimeProofInputContract {
	t.Helper()
	input := validCompileInputFixture(t)
	input.BindingCommitmentDeriver = NewPoseidonBindingCommitmentDeriverV0()
	compiled, err := CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0(input)
	if err != nil {
		t.Fatalf("CompileAuditIsolateSessionBoundAttestedRuntimeMembershipV0 fixture: %v", err)
	}
	return compiled
}
