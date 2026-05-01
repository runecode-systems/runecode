package launcherbackend

import "testing"

func TestDeriveAttestationVerifierClassFromReceipt(t *testing.T) {
	tests := []struct {
		name       string
		sourceKind string
		want       string
	}{
		{name: "trusted runtime", sourceKind: AttestationSourceKindTrustedRuntime, want: AttestationVerifierClassTrustedDomainLocal},
		{name: "tpm quote", sourceKind: AttestationSourceKindTPMQuote, want: AttestationVerifierClassHardwareRooted},
		{name: "tdx quote", sourceKind: AttestationSourceKindTDXQuote, want: AttestationVerifierClassHardwareRooted},
		{name: "sev-snp report", sourceKind: AttestationSourceKindSEVSNPReport, want: AttestationVerifierClassHardwareRooted},
		{name: "container image", sourceKind: AttestationSourceKindContainerImage, want: AttestationVerifierClassExternalVerifier},
		{name: "unknown", sourceKind: "", want: AttestationVerifierClassUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveAttestationVerifierClass(BackendLaunchReceipt{AttestationEvidenceSourceKind: tc.sourceKind})
			if got != tc.want {
				t.Fatalf("DeriveAttestationVerifierClass() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDeriveAttestationVerifierClassFromEvidence(t *testing.T) {
	evidence := RuntimeEvidenceSnapshot{Attestation: &IsolateAttestationEvidence{AttestationSourceKind: AttestationSourceKindTrustedRuntime}}
	if got := DeriveAttestationVerifierClassFromEvidence(evidence); got != AttestationVerifierClassTrustedDomainLocal {
		t.Fatalf("DeriveAttestationVerifierClassFromEvidence() = %q, want %q", got, AttestationVerifierClassTrustedDomainLocal)
	}
	if got := DeriveAttestationVerifierClassFromEvidence(RuntimeEvidenceSnapshot{}); got != AttestationVerifierClassUnknown {
		t.Fatalf("DeriveAttestationVerifierClassFromEvidence(nil attestation) = %q, want %q", got, AttestationVerifierClassUnknown)
	}
}
