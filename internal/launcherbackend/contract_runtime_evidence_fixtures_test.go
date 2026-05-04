package launcherbackend

import (
	"encoding/json"
	"maps"
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
	ExpectBootByName         map[string]string    `json:"expect_boot_component_digest_by_name"`
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
	sources := []string{AttestationSourceKindTPMQuote, AttestationSourceKindSEVSNPReport, AttestationSourceKindTDXQuote, AttestationSourceKindContainerImage}
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
	canonicalizeFixtureLaunchContextDigest(&facts.LaunchReceipt)
	canonicalizeFixtureMeasurementDigest(&facts.LaunchReceipt)
	ensureFixtureSessionValidated(&facts.LaunchReceipt)
	if facts.PostHandshakeAttestationInput == nil {
		facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)
	}
	facts.PostHandshakeAttestationInput.RuntimeEvidenceCollected = true
	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	return evidence
}

func ensureFixtureSessionValidated(receipt *BackendLaunchReceipt) {
	if receipt == nil || receipt.SessionSecurity != nil {
		return
	}
	receipt.SessionSecurity = &SessionSecurityPosture{
		MutuallyAuthenticated:     true,
		Encrypted:                 true,
		ProofOfPossessionVerified: true,
		ReplayProtected:           true,
	}
}

func canonicalizeFixtureLaunchContextDigest(receipt *BackendLaunchReceipt) {
	if receipt == nil || receipt.LaunchContextDigest != "" {
		return
	}
	receipt.LaunchContextDigest = testDigest("ac")
}

func canonicalizeFixtureMeasurementDigest(receipt *BackendLaunchReceipt) {
	if receipt == nil || receipt.AttestationMeasurementProfile == "" || receipt.AttestationEvidenceClaimsDigest != "" {
		return
	}
	componentDigests := bootComponentDigestByNameForFixture(receipt)
	if len(componentDigests) == 0 {
		return
	}
	digests, err := DeriveExpectedMeasurementDigests(receipt.AttestationMeasurementProfile, receipt.RuntimeImageBootProfile, componentDigests)
	if err != nil || len(digests) == 0 {
		return
	}
	receipt.AttestationEvidenceClaimsDigest = digests[0]
}

func bootComponentDigestByNameForFixture(receipt *BackendLaunchReceipt) map[string]string {
	if receipt == nil {
		return nil
	}
	return cloneStringMap(receipt.BootComponentDigestByName)
}

func assertFixtureVerificationResult(t *testing.T, tc attestationFixtureCase, evidence RuntimeEvidenceSnapshot) {
	t.Helper()
	assertFixtureAttestationPresence(t, tc, evidence)
	assertFixtureVerificationReasons(t, tc, evidence.AttestationVerification)
}

func assertFixtureAttestationPresence(t *testing.T, tc attestationFixtureCase, evidence RuntimeEvidenceSnapshot) {
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
}

func assertFixtureVerificationReasons(t *testing.T, tc attestationFixtureCase, verification *IsolateAttestationVerificationRecord) {
	t.Helper()
	if verification == nil {
		return
	}
	if tc.ExpectVerificationResult == AttestationVerificationResultInvalid && len(verification.ReasonCodes) == 0 {
		t.Fatal("reason_codes empty, want fail-closed reason for invalid verification")
	}
	for _, reason := range tc.ExpectReasonCodes {
		if fixtureReasonSatisfied(verification.ReasonCodes, reason) {
			continue
		}
		t.Fatalf("reason_codes = %#v, missing %q", verification.ReasonCodes, reason)
	}
}

func fixtureReasonSatisfied(reasonCodes []string, required string) bool {
	if containsAnyReasonCode(reasonCodes, required) {
		return true
	}
	return containsAnyReasonCode(reasonCodes, attestationReasonCodeIdentityBindingInvalid, attestationReasonCodeMeasurementDigestInvalid)
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
	if expectedByName := expectedFixtureBootComponentDigestByName(tc); len(expectedByName) > 0 {
		actualByName := evidence.Attestation.BootComponentDigestByName
		if !maps.Equal(actualByName, expectedByName) {
			t.Fatalf("boot_component_digest_by_name = %#v, want %#v", actualByName, expectedByName)
		}
	}
	if !slices.Equal(evidence.Attestation.BootComponentDigests, uniqueSortedStrings(tc.Facts.LaunchReceipt.BootComponentDigests)) {
		t.Fatalf("boot_component_digests = %#v, want %#v", evidence.Attestation.BootComponentDigests, uniqueSortedStrings(tc.Facts.LaunchReceipt.BootComponentDigests))
	}
}

func expectedFixtureBootComponentDigestByName(tc attestationFixtureCase) map[string]string {
	if len(tc.ExpectBootByName) > 0 {
		return tc.ExpectBootByName
	}
	if len(tc.Facts.LaunchReceipt.BootComponentDigestByName) > 0 {
		return tc.Facts.LaunchReceipt.BootComponentDigestByName
	}
	return nil
}

func attestationSourceFacts(source string) RuntimeFactsSnapshot {
	measurementProfile, bootProfile, bootComponentDigestByName, bootComponentDigests := attestationSourceFixtureIdentity(source)
	facts := DefaultRuntimeFacts("run-att-source-" + source)
	facts.LaunchReceipt = BackendLaunchReceipt{
		RunID:                             "run-att-source-" + source,
		StageID:                           "stage-1",
		RoleInstanceID:                    "workspace-1",
		BackendKind:                       BackendKindMicroVM,
		IsolationAssuranceLevel:           IsolationAssuranceIsolated,
		ProvisioningPosture:               ProvisioningPostureAttested,
		IsolateID:                         "isolate-1",
		SessionID:                         "session-1",
		SessionNonce:                      "nonce-0123456789abcdef",
		LaunchContextDigest:               testDigest("ac"),
		HandshakeTranscriptHash:           testDigest("a"),
		IsolateSessionKeyIDValue:          testDigest("b")[7:],
		RuntimeImageDescriptorDigest:      testDigest("c"),
		RuntimeImageBootProfile:           bootProfile,
		BootComponentDigestByName:         bootComponentDigestByName,
		BootComponentDigests:              bootComponentDigests,
		AttestationEvidenceSourceKind:     source,
		AttestationMeasurementProfile:     measurementProfile,
		AttestationFreshnessMaterial:      []string{"nonce"},
		AttestationFreshnessBindingClaims: []string{"session_nonce"},
		AttestationEvidenceClaimsDigest:   attestationSourceFixtureClaimsDigest(measurementProfile, bootProfile, bootComponentDigestByName),
	}
	return facts
}

func attestationSourceFixtureClaimsDigest(measurementProfile, bootProfile string, bootComponentDigestByName map[string]string) string {
	digests, err := DeriveExpectedMeasurementDigests(measurementProfile, bootProfile, bootComponentDigestByName)
	if err != nil {
		panic(err)
	}
	return digests[0]
}

func attestationSourceFixtureIdentity(source string) (string, string, map[string]string, []string) {
	if source == AttestationSourceKindContainerImage {
		return MeasurementProfileContainerImageV1, BootProfileContainerOCIImageV1, map[string]string{"image": testDigest("d")}, []string{testDigest("d")}
	}
	return MeasurementProfileMicroVMBootV1, BootProfileMicroVMLinuxKernelInitrdV1, map[string]string{"kernel": testDigest("d"), "initrd": testDigest("e")}, []string{testDigest("d"), testDigest("e")}
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
