package launcherbackend

import (
	"maps"
	"testing"
)

func TestSplitRuntimeFactsEvidenceAndLifecycleSeparatesImmutableEvidence(t *testing.T) {
	facts := DefaultRuntimeFacts("run-1")
	facts.LaunchReceipt = BackendLaunchReceipt{
		RunID:                    "run-1",
		StageID:                  "stage-1",
		RoleInstanceID:           "workspace-1",
		BackendKind:              BackendKindMicroVM,
		IsolationAssuranceLevel:  IsolationAssuranceIsolated,
		ProvisioningPosture:      ProvisioningPostureTOFU,
		IsolateID:                "isolate-1",
		SessionID:                "session-1",
		SessionNonce:             "nonce-0123456789abcdef",
		LaunchContextDigest:      testDigest("1"),
		HandshakeTranscriptHash:  testDigest("2"),
		IsolateSessionKeyIDValue: testDigest("3")[7:],
		SessionSecurity: &SessionSecurityPosture{
			MutuallyAuthenticated:     true,
			Encrypted:                 true,
			ProofOfPossessionVerified: true,
			ReplayProtected:           true,
		},
		Lifecycle: &BackendLifecycleSnapshot{CurrentState: BackendLifecycleStateActive, PreviousState: BackendLifecycleStateBinding, TerminateBetweenSteps: true},
	}
	evidence, state, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.Launch.EvidenceDigest == "" || evidence.Hardening.EvidenceDigest == "" {
		t.Fatalf("evidence digests should be populated, got launch=%q hardening=%q", evidence.Launch.EvidenceDigest, evidence.Hardening.EvidenceDigest)
	}
	if evidence.Session == nil || evidence.Session.EvidenceDigest == "" {
		t.Fatalf("session evidence should be present with digest, got %#v", evidence.Session)
	}
	if state.BackendLifecycle == nil || state.BackendLifecycle.CurrentState != BackendLifecycleStateActive {
		t.Fatalf("runtime lifecycle state not preserved: %#v", state.BackendLifecycle)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleBuildsIsolateAttestationEvidence(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	assertAttestationEvidenceLinkedToRuntime(t, evidence)
}

func TestSplitRuntimeFactsEvidenceAndLifecyclePreservesBootComponentIdentityAgainstPositionalSwap(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.LaunchReceipt.BootComponentDigests = []string{testDigest("a"), testDigest("b")}
	facts.LaunchReceipt.BootComponentDigestByName = map[string]string{
		"kernel": testDigest("b"),
		"initrd": testDigest("a"),
	}
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = runtimeEvidenceMeasurementDigestForTests(
		BootProfileMicroVMLinuxKernelInitrdV1,
		facts.LaunchReceipt.BootComponentDigestByName,
	)

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.Attestation == nil {
		t.Fatal("attestation evidence missing")
	}
	actualByName := evidence.Attestation.BootComponentDigestByName
	wantByName := map[string]string{"kernel": testDigest("b"), "initrd": testDigest("a")}
	if !maps.Equal(actualByName, wantByName) {
		t.Fatalf("boot_component_digest_by_name = %#v, want %#v", actualByName, wantByName)
	}
	positionalByName := map[string]string{"kernel": testDigest("a"), "initrd": testDigest("b")}
	if maps.Equal(actualByName, positionalByName) {
		t.Fatalf("boot_component_digest_by_name should not collapse to positional mapping: %#v", actualByName)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedForIncompleteNamedBootComponents(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.LaunchReceipt.BootComponentDigests = []string{testDigest("a"), testDigest("b")}
	facts.LaunchReceipt.BootComponentDigestByName = map[string]string{
		"kernel": testDigest("a"),
	}
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = runtimeEvidenceMeasurementDigestForTests(
		BootProfileMicroVMLinuxKernelInitrdV1,
		map[string]string{"kernel": testDigest("a"), "initrd": testDigest("b")},
	)

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultInvalid {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, AttestationVerificationResultInvalid)
	}
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeMeasurementDigestInvalid) {
		t.Fatalf("reason_codes = %#v, expected measurement digest invalid", evidence.AttestationVerification.ReasonCodes)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedWithoutNamedBootComponents(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.LaunchReceipt.BootComponentDigests = []string{testDigest("a"), testDigest("b")}
	facts.LaunchReceipt.BootComponentDigestByName = nil
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = runtimeEvidenceMeasurementDigestForTests(
		BootProfileMicroVMLinuxKernelInitrdV1,
		map[string]string{"kernel": testDigest("a"), "initrd": testDigest("b")},
	)

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultInvalid {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, AttestationVerificationResultInvalid)
	}
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeMeasurementDigestInvalid) {
		t.Fatalf("reason_codes = %#v, expected measurement digest invalid", evidence.AttestationVerification.ReasonCodes)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedForMalformedNamedBootComponentDigest(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.LaunchReceipt.BootComponentDigests = []string{testDigest("a"), testDigest("b")}
	facts.LaunchReceipt.BootComponentDigestByName = map[string]string{
		"kernel": "not-a-digest",
		"initrd": testDigest("b"),
	}
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = runtimeEvidenceMeasurementDigestForTests(
		BootProfileMicroVMLinuxKernelInitrdV1,
		map[string]string{"kernel": testDigest("a"), "initrd": testDigest("b")},
	)

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultInvalid {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, AttestationVerificationResultInvalid)
	}
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeMeasurementDigestInvalid) {
		t.Fatalf("reason_codes = %#v, expected measurement digest invalid", evidence.AttestationVerification.ReasonCodes)
	}
}

func attestationRuntimeFactsFixture() RuntimeFactsSnapshot {
	facts := DefaultRuntimeFacts("run-att-1")
	facts.LaunchReceipt = BackendLaunchReceipt{
		RunID:                               "run-att-1",
		StageID:                             "stage-1",
		RoleInstanceID:                      "workspace-1",
		BackendKind:                         BackendKindMicroVM,
		IsolationAssuranceLevel:             IsolationAssuranceIsolated,
		ProvisioningPosture:                 ProvisioningPostureAttested,
		IsolateID:                           "isolate-1",
		SessionID:                           "session-1",
		SessionNonce:                        "nonce-0123456789abcdef",
		LaunchContextDigest:                 testDigest("11"),
		HandshakeTranscriptHash:             testDigest("12"),
		IsolateSessionKeyIDValue:            testDigest("13")[7:],
		RuntimeImageDescriptorDigest:        testDigest("14"),
		RuntimeImageBootProfile:             BootProfileMicroVMLinuxKernelInitrdV1,
		BootComponentDigestByName:           map[string]string{"kernel": testDigest("15"), "initrd": testDigest("16")},
		BootComponentDigests:                []string{testDigest("15"), testDigest("16")},
		AttestationEvidenceSourceKind:       AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile:       "microvm-boot-v1",
		AttestationFreshnessMaterial:        []string{"quote_nonce"},
		AttestationFreshnessBindingClaims:   []string{"session_nonce", "transcript_hash"},
		AttestationEvidenceClaimsDigest:     runtimeEvidenceMeasurementDigestForTests(BootProfileMicroVMLinuxKernelInitrdV1, map[string]string{"kernel": testDigest("15"), "initrd": testDigest("16")}),
		AttestationVerifierPolicyID:         "policy-default",
		AttestationVerifierPolicyDigest:     testDigest("18"),
		AttestationVerificationRulesVersion: "v1",
		AttestationVerificationResult:       AttestationVerificationResultValid,
		AttestationVerificationReasonCodes:  []string{"ok"},
		AttestationReplayVerdict:            AttestationReplayVerdictOriginal,
		AttestationVerificationTimestamp:    "2026-04-29T12:00:00Z",
	}
	return facts
}

func runtimeEvidenceMeasurementDigestForTests(bootProfile string, componentDigests map[string]string) string {
	digests, err := DeriveExpectedMeasurementDigests(MeasurementProfileMicroVMBootV1, bootProfile, componentDigests)
	if err != nil {
		panic(err)
	}
	return digests[0]
}

func assertAttestationEvidenceLinkedToRuntime(t *testing.T, evidence RuntimeEvidenceSnapshot) {
	t.Helper()
	if evidence.Session == nil || evidence.Session.EvidenceDigest == "" {
		t.Fatalf("session binding evidence missing: %#v", evidence.Session)
	}
	if evidence.Attestation == nil || evidence.Attestation.EvidenceDigest == "" {
		t.Fatalf("attestation evidence missing: %#v", evidence.Attestation)
	}
	if evidence.Attestation.LaunchRuntimeEvidenceDigest != evidence.Launch.EvidenceDigest {
		t.Fatalf("attestation launch linkage digest = %q, want %q", evidence.Attestation.LaunchRuntimeEvidenceDigest, evidence.Launch.EvidenceDigest)
	}
	if evidence.AttestationVerification == nil || evidence.AttestationVerification.VerificationDigest == "" {
		t.Fatalf("attestation verification missing: %#v", evidence.AttestationVerification)
	}
	if evidence.AttestationVerification.AttestationEvidenceDigest != evidence.Attestation.EvidenceDigest {
		t.Fatalf("verification evidence digest = %q, want %q", evidence.AttestationVerification.AttestationEvidenceDigest, evidence.Attestation.EvidenceDigest)
	}
	if evidence.AttestationVerification.ReplayIdentityDigest == "" {
		t.Fatal("replay identity digest should be populated")
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedForReplayWhenAttestedRequired(t *testing.T) {
	facts := DefaultRuntimeFacts("run-att-replay")
	facts.LaunchReceipt = BackendLaunchReceipt{
		RunID:                             "run-att-replay",
		StageID:                           "stage-1",
		RoleInstanceID:                    "workspace-1",
		BackendKind:                       BackendKindMicroVM,
		IsolationAssuranceLevel:           IsolationAssuranceIsolated,
		ProvisioningPosture:               ProvisioningPostureAttested,
		IsolateID:                         "isolate-1",
		SessionID:                         "session-1",
		SessionNonce:                      "nonce-0123456789abcdef",
		HandshakeTranscriptHash:           testDigest("22"),
		IsolateSessionKeyIDValue:          testDigest("23")[7:],
		RuntimeImageDescriptorDigest:      testDigest("24"),
		RuntimeImageBootProfile:           BootProfileMicroVMLinuxKernelInitrdV1,
		AttestationEvidenceSourceKind:     AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile:     "microvm-boot-v1",
		AttestationFreshnessMaterial:      []string{"quote_nonce"},
		AttestationFreshnessBindingClaims: []string{"session_nonce"},
		AttestationVerificationResult:     AttestationVerificationResultValid,
		AttestationReplayVerdict:          AttestationReplayVerdictReplay,
	}

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultInvalid {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, AttestationVerificationResultInvalid)
	}
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeReplayDetected) {
		t.Fatalf("reason_codes = %#v, expected replay reason", evidence.AttestationVerification.ReasonCodes)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedForMissingFreshnessWhenAttestedRequired(t *testing.T) {
	facts := DefaultRuntimeFacts("run-att-freshness")
	facts.LaunchReceipt = BackendLaunchReceipt{
		RunID:                         "run-att-freshness",
		StageID:                       "stage-1",
		RoleInstanceID:                "workspace-1",
		BackendKind:                   BackendKindMicroVM,
		IsolationAssuranceLevel:       IsolationAssuranceIsolated,
		ProvisioningPosture:           ProvisioningPostureAttested,
		IsolateID:                     "isolate-1",
		SessionID:                     "session-1",
		SessionNonce:                  "nonce-0123456789abcdef",
		HandshakeTranscriptHash:       testDigest("32"),
		IsolateSessionKeyIDValue:      testDigest("33")[7:],
		RuntimeImageDescriptorDigest:  testDigest("34"),
		RuntimeImageBootProfile:       BootProfileMicroVMLinuxKernelInitrdV1,
		AttestationEvidenceSourceKind: AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile: "microvm-boot-v1",
		AttestationVerificationResult: AttestationVerificationResultValid,
	}

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultInvalid {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, AttestationVerificationResultInvalid)
	}
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeFreshnessMaterialMissing, attestationReasonCodeFreshnessBindingMissing) {
		t.Fatalf("reason_codes = %#v, expected freshness reasons", evidence.AttestationVerification.ReasonCodes)
	}
}
