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
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)

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
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)

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
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)

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
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)

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
		RunID:                             "run-att-1",
		StageID:                           "stage-1",
		RoleInstanceID:                    "workspace-1",
		BackendKind:                       BackendKindMicroVM,
		IsolationAssuranceLevel:           IsolationAssuranceIsolated,
		ProvisioningPosture:               ProvisioningPostureAttested,
		IsolateID:                         "isolate-1",
		SessionID:                         "session-1",
		SessionNonce:                      "nonce-0123456789abcdef",
		LaunchContextDigest:               testDigest("11"),
		HandshakeTranscriptHash:           testDigest("12"),
		IsolateSessionKeyIDValue:          testDigest("3")[7:],
		RuntimeImageDescriptorDigest:      testDigest("4"),
		RuntimeImageBootProfile:           BootProfileMicroVMLinuxKernelInitrdV1,
		BootComponentDigestByName:         map[string]string{"kernel": testDigest("5"), "initrd": testDigest("6")},
		BootComponentDigests:              []string{testDigest("5"), testDigest("6")},
		AttestationEvidenceSourceKind:     AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile:     "microvm-boot-v1",
		AttestationFreshnessMaterial:      []string{"quote_nonce"},
		AttestationFreshnessBindingClaims: []string{"session_nonce", "transcript_hash"},
		AttestationEvidenceClaimsDigest:   runtimeEvidenceMeasurementDigestForTests(BootProfileMicroVMLinuxKernelInitrdV1, map[string]string{"kernel": testDigest("5"), "initrd": testDigest("6")}),
		SessionSecurity: &SessionSecurityPosture{
			MutuallyAuthenticated:     true,
			Encrypted:                 true,
			ProofOfPossessionVerified: true,
			ReplayProtected:           true,
		},
	}
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)
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
	if evidence.Attestation.LaunchContextDigest != evidence.Session.LaunchContextDigest {
		t.Fatalf("attestation launch_context_digest = %q, want session digest %q", evidence.Attestation.LaunchContextDigest, evidence.Session.LaunchContextDigest)
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
	facts := replayAttestationRuntimeFactsFixture()
	evidence := requireRuntimeEvidenceForFacts(t, facts)
	assertInvalidVerificationResult(t, evidence)
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeReplayDetected, attestationReasonCodeIdentityBindingInvalid, attestationReasonCodeMeasurementDigestInvalid) {
		t.Fatalf("reason_codes = %#v, expected fail-closed replay or identity-binding reason", evidence.AttestationVerification.ReasonCodes)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedForMissingFreshnessWhenAttestedRequired(t *testing.T) {
	facts := freshnessMissingAttestationRuntimeFactsFixture()
	evidence := requireRuntimeEvidenceForFacts(t, facts)
	assertInvalidVerificationResult(t, evidence)
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeFreshnessMaterialMissing, attestationReasonCodeFreshnessBindingMissing) {
		t.Fatalf("reason_codes = %#v, expected freshness reasons", evidence.AttestationVerification.ReasonCodes)
	}
}

func replayAttestationRuntimeFactsFixture() RuntimeFactsSnapshot {
	facts := DefaultRuntimeFacts("run-att-replay")
	facts.LaunchReceipt = attestedRuntimeEvidenceReceiptFixture("run-att-replay", testDigest("21"), testDigest("22"), testDigest("23")[7:], testDigest("24"))
	facts.LaunchReceipt.AttestationFreshnessMaterial = []string{"quote_nonce"}
	facts.LaunchReceipt.AttestationFreshnessBindingClaims = []string{"session_nonce"}
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)
	return facts
}

func freshnessMissingAttestationRuntimeFactsFixture() RuntimeFactsSnapshot {
	facts := DefaultRuntimeFacts("run-att-freshness")
	facts.LaunchReceipt = attestedRuntimeEvidenceReceiptFixture("run-att-freshness", testDigest("31"), testDigest("32"), testDigest("33")[7:], testDigest("34"))
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)
	return facts
}

func attestedRuntimeEvidenceReceiptFixture(runID, launchContextDigest, handshakeTranscriptHash, keyIDValue, imageDigest string) BackendLaunchReceipt {
	return BackendLaunchReceipt{
		RunID:                         runID,
		StageID:                       "stage-1",
		RoleInstanceID:                "workspace-1",
		BackendKind:                   BackendKindMicroVM,
		IsolationAssuranceLevel:       IsolationAssuranceIsolated,
		ProvisioningPosture:           ProvisioningPostureAttested,
		IsolateID:                     "isolate-1",
		SessionID:                     "session-1",
		SessionNonce:                  "nonce-0123456789abcdef",
		LaunchContextDigest:           launchContextDigest,
		HandshakeTranscriptHash:       handshakeTranscriptHash,
		IsolateSessionKeyIDValue:      keyIDValue,
		RuntimeImageDescriptorDigest:  imageDigest,
		RuntimeImageBootProfile:       BootProfileMicroVMLinuxKernelInitrdV1,
		AttestationEvidenceSourceKind: AttestationSourceKindTPMQuote,
		AttestationMeasurementProfile: "microvm-boot-v1",
		SessionSecurity: &SessionSecurityPosture{
			MutuallyAuthenticated:     true,
			Encrypted:                 true,
			ProofOfPossessionVerified: true,
			ReplayProtected:           true,
		},
	}
}

func requireRuntimeEvidenceForFacts(t *testing.T, facts RuntimeFactsSnapshot) RuntimeEvidenceSnapshot {
	t.Helper()
	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	return evidence
}

func assertInvalidVerificationResult(t *testing.T, evidence RuntimeEvidenceSnapshot) {
	t.Helper()
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultInvalid {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, AttestationVerificationResultInvalid)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleUsesPostHandshakeAttestationInputSeam(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.PostHandshakeAttestationInput = &PostHandshakeRuntimeAttestationInput{
		RunID:                        facts.LaunchReceipt.RunID,
		IsolateID:                    facts.LaunchReceipt.IsolateID,
		SessionID:                    facts.LaunchReceipt.SessionID,
		SessionNonce:                 facts.LaunchReceipt.SessionNonce,
		RuntimeEvidenceCollected:     true,
		LaunchContextDigest:          testDigest("77"),
		HandshakeTranscriptHash:      facts.LaunchReceipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     facts.LaunchReceipt.IsolateSessionKeyIDValue,
		RuntimeImageDescriptorDigest: facts.LaunchReceipt.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      facts.LaunchReceipt.RuntimeImageBootProfile,
		BootComponentDigestByName:    cloneStringMap(facts.LaunchReceipt.BootComponentDigestByName),
		AttestationSourceKind:        facts.LaunchReceipt.AttestationEvidenceSourceKind,
		MeasurementProfile:           facts.LaunchReceipt.AttestationMeasurementProfile,
		FreshnessMaterial:            []string{"session_nonce"},
		FreshnessBindingClaims:       []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"},
		EvidenceClaimsDigest:         facts.LaunchReceipt.AttestationEvidenceClaimsDigest,
		VerifierPolicyID:             facts.LaunchReceipt.AttestationVerifierPolicyID,
		VerifierPolicyDigest:         facts.LaunchReceipt.AttestationVerifierPolicyDigest,
		VerificationResult:           facts.LaunchReceipt.AttestationVerificationResult,
		ReplayVerdict:                facts.LaunchReceipt.AttestationReplayVerdict,
	}

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.Attestation == nil {
		t.Fatal("attestation evidence missing")
	}
	if got, want := evidence.Attestation.LaunchContextDigest, facts.PostHandshakeAttestationInput.LaunchContextDigest; got != want {
		t.Fatalf("attestation launch_context_digest = %q, want seam digest %q", got, want)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedWhenReceiptClaimsAttestedWithoutPostHandshakeInput(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.PostHandshakeAttestationInput = nil

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.Attestation != nil {
		t.Fatalf("attestation evidence = %#v, want nil without post-handshake input", evidence.Attestation)
	}
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != AttestationVerificationResultInvalid {
		t.Fatalf("verification_result = %q, want %q", evidence.AttestationVerification.VerificationResult, AttestationVerificationResultInvalid)
	}
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodePostHandshakeInputRequired) {
		t.Fatalf("reason_codes = %#v, expected %q", evidence.AttestationVerification.ReasonCodes, attestationReasonCodePostHandshakeInputRequired)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedWhenSessionValidationMissing(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.LaunchReceipt.SessionSecurity = nil
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)

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
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeSessionValidationRequired) {
		t.Fatalf("reason_codes = %#v, expected %q", evidence.AttestationVerification.ReasonCodes, attestationReasonCodeSessionValidationRequired)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecyclePromotesAttestedOnlyAfterTrustedVerification(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.LaunchReceipt.ProvisioningPosture = ProvisioningPostureTOFU
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)

	evidence, _, err := SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.AttestationVerification == nil || evidence.AttestationVerification.VerificationResult != AttestationVerificationResultValid {
		t.Fatalf("attestation verification = %#v, want valid", evidence.AttestationVerification)
	}
	if got, want := evidence.Launch.ProvisioningPosture, ProvisioningPostureAttested; got != want {
		t.Fatalf("launch provisioning posture = %q, want %q", got, want)
	}
	if evidence.Session == nil || evidence.Session.ProvisioningPosture != ProvisioningPostureAttested {
		t.Fatalf("session evidence posture = %#v, want %q", evidence.Session, ProvisioningPostureAttested)
	}
}

func TestSplitRuntimeFactsEvidenceAndLifecycleFailsClosedWhenPostHandshakeIdentityMismatchesLaunch(t *testing.T) {
	facts := attestationRuntimeFactsFixture()
	facts.PostHandshakeAttestationInput = postHandshakeInputFromReceipt(facts.LaunchReceipt)
	facts.PostHandshakeAttestationInput.RuntimeImageDescriptorDigest = testDigest("ff")

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
	if !containsAnyReasonCode(evidence.AttestationVerification.ReasonCodes, attestationReasonCodeIdentityBindingInvalid) {
		t.Fatalf("reason_codes = %#v, expected %q", evidence.AttestationVerification.ReasonCodes, attestationReasonCodeIdentityBindingInvalid)
	}
}

func postHandshakeInputFromReceipt(receipt BackendLaunchReceipt) *PostHandshakeRuntimeAttestationInput {
	return &PostHandshakeRuntimeAttestationInput{
		RunID:                        receipt.RunID,
		IsolateID:                    receipt.IsolateID,
		SessionID:                    receipt.SessionID,
		SessionNonce:                 receipt.SessionNonce,
		RuntimeEvidenceCollected:     true,
		LaunchContextDigest:          receipt.LaunchContextDigest,
		HandshakeTranscriptHash:      receipt.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     receipt.IsolateSessionKeyIDValue,
		RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
		RuntimeImageVerifierRef:      receipt.RuntimeImageVerifierRef,
		AuthorityStateDigest:         receipt.AuthorityStateDigest,
		BootComponentDigestByName:    cloneStringMap(receipt.BootComponentDigestByName),
		BootComponentDigests:         append([]string{}, receipt.BootComponentDigests...),
		AttestationSourceKind:        receipt.AttestationEvidenceSourceKind,
		MeasurementProfile:           receipt.AttestationMeasurementProfile,
		FreshnessMaterial:            append([]string{}, receipt.AttestationFreshnessMaterial...),
		FreshnessBindingClaims:       append([]string{}, receipt.AttestationFreshnessBindingClaims...),
		EvidenceClaimsDigest:         receipt.AttestationEvidenceClaimsDigest,
	}
}
