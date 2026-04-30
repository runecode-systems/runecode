package launcherbackend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

type attestationFixtureSuite struct {
	Cases []attestationFixtureCase `json:"cases"`
}

type attestationFixtureCase struct {
	Name                     string               `json:"name"`
	Facts                    RuntimeFactsSnapshot `json:"facts"`
	ExpectAttestation        bool                 `json:"expect_attestation"`
	ExpectVerificationResult string               `json:"expect_verification_result"`
	ExpectReasonCodes        []string             `json:"expect_reason_codes"`
}

func TestAttestationFixturesCoverFailClosedAndPlatformNeutralBindings(t *testing.T) {
	suite := loadAttestationFixtures(t)
	if len(suite.Cases) < 5 {
		t.Fatalf("fixture cases = %d, want at least 5", len(suite.Cases))
	}
	for _, tc := range suite.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			evidence := splitRuntimeEvidenceForFixture(t, tc.Facts)
			assertFixtureVerificationResult(t, tc, evidence)
			assertFixtureRuntimeIdentityBinding(t, tc, evidence)
		})
	}
}

func TestAttestationFixturesSupportPlatformSpecificSourceKindsWithoutSemanticFork(t *testing.T) {
	sources := []string{AttestationSourceKindTPMQuote, AttestationSourceKindSEVSNPReport, AttestationSourceKindTDXQuote, AttestationSourceKindContainerSig}
	for _, source := range sources {
		source := source
		t.Run(source, func(t *testing.T) {
			facts := attestationSourceFacts(source)
			evidence := splitRuntimeEvidenceForFixture(t, facts)
			assertAttestationSourceSemantics(t, source, facts, evidence)
		})
	}
}

func splitRuntimeEvidenceForFixture(t *testing.T, facts RuntimeFactsSnapshot) RuntimeEvidenceSnapshot {
	t.Helper()
	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	return evidence
}

func assertFixtureVerificationResult(t *testing.T, tc attestationFixtureCase, evidence RuntimeEvidenceSnapshot) {
	t.Helper()
	if tc.ExpectAttestation && evidence.Attestation == nil {
		t.Fatal("expected attestation evidence")
	}
	if !tc.ExpectAttestation && evidence.Attestation != nil {
		t.Fatalf("unexpected attestation evidence: %#v", evidence.Attestation)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("expected attestation verification record")
	}
	if evidence.AttestationVerification.VerificationResult != tc.ExpectVerificationResult {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, tc.ExpectVerificationResult)
	}
	for _, reason := range tc.ExpectReasonCodes {
		if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, reason) {
			t.Fatalf("reason_codes = %#v, missing %q", evidence.AttestationVerification.ReasonCodes, reason)
		}
	}
}

func assertFixtureRuntimeIdentityBinding(t *testing.T, tc attestationFixtureCase, evidence RuntimeEvidenceSnapshot) {
	t.Helper()
	if evidence.Attestation == nil {
		return
	}
	if evidence.Attestation.RuntimeImageDescriptorDigest != tc.Facts.LaunchReceipt.RuntimeImageDescriptorDigest {
		t.Fatalf("runtime_image_descriptor_digest = %q, want %q", evidence.Attestation.RuntimeImageDescriptorDigest, tc.Facts.LaunchReceipt.RuntimeImageDescriptorDigest)
	}
	if evidence.Attestation.RuntimeImageBootProfile != tc.Facts.LaunchReceipt.RuntimeImageBootProfile {
		t.Fatalf("runtime_image_boot_profile = %q, want %q", evidence.Attestation.RuntimeImageBootProfile, tc.Facts.LaunchReceipt.RuntimeImageBootProfile)
	}
	if !slices.Equal(evidence.Attestation.BootComponentDigests, uniqueSortedStrings(tc.Facts.LaunchReceipt.BootComponentDigests)) {
		t.Fatalf("boot_component_digests = %#v, want %#v", evidence.Attestation.BootComponentDigests, uniqueSortedStrings(tc.Facts.LaunchReceipt.BootComponentDigests))
	}
}

func attestationSourceFacts(source string) RuntimeFactsSnapshot {
	facts := DefaultRuntimeFacts("run-att-source-" + source)
	facts.LaunchReceipt = BackendLaunchReceipt{
		RunID:                               "run-att-source-" + source,
		StageID:                             "stage-1",
		RoleInstanceID:                      "workspace-1",
		BackendKind:                         BackendKindMicroVM,
		IsolationAssuranceLevel:             IsolationAssuranceIsolated,
		ProvisioningPosture:                 ProvisioningPostureAttested,
		IsolateID:                           "isolate-1",
		SessionID:                           "session-1",
		SessionNonce:                        "nonce-0123456789abcdef",
		HandshakeTranscriptHash:             testDigest("a"),
		IsolateSessionKeyIDValue:            testDigest("b")[7:],
		RuntimeImageDescriptorDigest:        testDigest("c"),
		RuntimeImageBootProfile:             BootProfileMicroVMLinuxKernelInitrdV1,
		BootComponentDigests:                []string{testDigest("d"), testDigest("e")},
		AttestationEvidenceSourceKind:       source,
		AttestationMeasurementProfile:       "microvm-boot-v1",
		AttestationFreshnessMaterial:        []string{"nonce"},
		AttestationFreshnessBindingClaims:   []string{"session_nonce"},
		AttestationEvidenceClaimsDigest:     testDigest("f"),
		AttestationVerifierPolicyID:         "runtime_asset_admission_identity",
		AttestationVerifierPolicyDigest:     testDigest("1"),
		AttestationVerificationRulesVersion: "v1",
		AttestationVerificationTimestamp:    "2026-04-29T12:00:00Z",
		AttestationVerificationResult:       AttestationVerificationResultValid,
		AttestationReplayVerdict:            AttestationReplayVerdictOriginal,
	}
	return facts
}

func assertAttestationSourceSemantics(t *testing.T, source string, facts RuntimeFactsSnapshot, evidence RuntimeEvidenceSnapshot) {
	t.Helper()
	if evidence.Attestation == nil {
		t.Fatal("expected attestation evidence")
	}
	if evidence.Attestation.AttestationSourceKind != source {
		t.Fatalf("attestation_source_kind = %q, want %q", evidence.Attestation.AttestationSourceKind, source)
	}
	if evidence.AttestationVerification == nil || evidence.AttestationVerification.VerificationResult != AttestationVerificationResultValid {
		t.Fatalf("verification = %#v, want valid", evidence.AttestationVerification)
	}
	if evidence.Attestation.RuntimeImageDescriptorDigest != facts.LaunchReceipt.RuntimeImageDescriptorDigest {
		t.Fatal("runtime identity binding changed by source kind")
	}
}

func loadAttestationFixtures(t *testing.T) attestationFixtureSuite {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "fixtures", "attestation-evidence-cases.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", path, err)
	}
	var suite attestationFixtureSuite
	if err := json.Unmarshal(raw, &suite); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	return suite
}
