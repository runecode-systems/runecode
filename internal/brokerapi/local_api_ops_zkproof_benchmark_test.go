package brokerapi

import (
	"testing"
)

func BenchmarkEvaluationOnlyZKProofHarnessEndToEnd(b *testing.B) {
	service, recordDigest := mustSetupZKProofE2EService(b)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		outcome := runEvaluationOnlyZKProofHarnessV0(b, service, recordDigest)
		assertBenchmarkVerificationOutcome(b, outcome)
		if i == 0 {
			reportBenchmarkArtifactMetrics(b, service, outcome)
		}
	}
}

func assertBenchmarkVerificationOutcome(b *testing.B, outcome evaluationHarnessOutcome) {
	b.Helper()
	if outcome.VerificationRecord.VerificationOutcome != "verified" {
		b.Fatalf("verification_outcome = %q, want verified", outcome.VerificationRecord.VerificationOutcome)
	}
}

func reportBenchmarkArtifactMetrics(b *testing.B, service *Service, outcome evaluationHarnessOutcome) {
	b.Helper()
	proofArtifact, found, err := service.auditLedger.ZKProofArtifactByDigest(outcome.ProofArtifactDigest)
	if err != nil {
		b.Fatalf("ZKProofArtifactByDigest returned error: %v", err)
	}
	if !found {
		b.Fatal("ZKProofArtifactByDigest found=false, want true")
	}
	b.ReportMetric(float64(len(proofArtifact.ProofBytes)), "proof_b64_bytes")
	b.ReportMetric(float64(len(proofArtifact.PublicInputs)), "public_input_fields")
}
