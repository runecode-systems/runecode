package zkproof

import (
	"encoding/json"
	"testing"
)

func BenchmarkEvaluationOnlyGroth16Prove(b *testing.B) {
	contract := mustEvaluationProofContractFixture(b)
	backend, _, _, _, err := NewEvaluationOnlyGroth16BackendForBenchmarkV0()
	if err != nil {
		b.Fatalf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %v", err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		proof, _, err := backend.ProveDeterministic(contract)
		if err != nil {
			b.Fatalf("ProveDeterministic returned error: %v", err)
		}
		if i == 0 {
			b.ReportMetric(float64(len(proof)), "proof_bytes")
		}
	}
	publicInputsJSON, err := json.Marshal(contract.PublicInputs)
	if err != nil {
		b.Fatalf("marshal public inputs: %v", err)
	}
	b.ReportMetric(float64(len(publicInputsJSON)), "public_inputs_json_bytes")
}

func BenchmarkEvaluationOnlyGroth16VerifyWarm(b *testing.B) {
	contract := mustEvaluationProofContractFixture(b)
	backend, _, _, trusted, err := NewEvaluationOnlyGroth16BackendForBenchmarkV0()
	if err != nil {
		b.Fatalf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %v", err)
	}
	proof, identity, err := backend.ProveDeterministic(contract)
	if err != nil {
		b.Fatalf("ProveDeterministic returned error: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := VerifyProofWithTrustedPostureV0(backend, proof, contract.PublicInputs, identity, trusted); err != nil {
			b.Fatalf("VerifyProofWithTrustedPostureV0 returned error: %v", err)
		}
	}
}

func BenchmarkEvaluationOnlyGroth16VerifyColdishSetupAndVerify(b *testing.B) {
	contract := mustEvaluationProofContractFixture(b)
	backend, _, _, _, err := NewEvaluationOnlyGroth16BackendForBenchmarkV0()
	if err != nil {
		b.Fatalf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %v", err)
	}
	proof, _, err := backend.ProveDeterministic(contract)
	if err != nil {
		b.Fatalf("ProveDeterministic returned error: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setup, err := newEvaluationOnlyGroth16BackendFreshV0()
		if err != nil {
			b.Fatalf("newEvaluationOnlyGroth16BackendFreshV0 returned error: %v", err)
		}
		coldBackend := evaluationOnlyGroth16BackendV0{inner: trustedLocalGroth16BackendV0{setup: setup}}
		if err := coldBackend.VerifyDeterministic(proof, contract.PublicInputs); err != nil {
			b.Fatalf("VerifyDeterministic returned error: %v", err)
		}
	}
}

func BenchmarkEvaluationOnlyGroth16RejectInvalidProof(b *testing.B) {
	contract := mustEvaluationProofContractFixture(b)
	backend, _, _, trusted, err := NewEvaluationOnlyGroth16BackendForBenchmarkV0()
	if err != nil {
		b.Fatalf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %v", err)
	}
	proof, identity, err := backend.ProveDeterministic(contract)
	if err != nil {
		b.Fatalf("ProveDeterministic returned error: %v", err)
	}
	invalid := append([]byte(nil), proof...)
	invalid[len(invalid)-1] ^= 0x01
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := VerifyProofWithTrustedPostureV0(backend, invalid, contract.PublicInputs, identity, trusted); err == nil {
			b.Fatal("expected invalid proof rejection")
		}
	}
}

func BenchmarkEvaluationOnlyGroth16SetupCacheHit(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend, _, _, _, err := NewEvaluationOnlyGroth16BackendForBenchmarkV0()
		if err != nil {
			b.Fatalf("NewEvaluationOnlyGroth16BackendForBenchmarkV0 returned error: %v", err)
		}
		if backend.BackendIdentity() != evaluationOnlyGroth16BackendIdentityV0 {
			b.Fatalf("unexpected backend identity: %q", backend.BackendIdentity())
		}
	}
}
