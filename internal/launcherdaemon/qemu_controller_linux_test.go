//go:build linux

package launcherdaemon

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestQEMUPrepareLaunchAssetsVerifiesToolchainArtifactBeforeLaunch(t *testing.T) {
	workRoot, qemuBinary, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	controller := &qemuController{cfg: QEMUControllerConfig{WorkRoot: workRoot, QEMUBinary: qemuBinary, Now: time.Now}, instances: map[string]*qemuInstance{}}

	if _, _, _, _, err := controller.prepareLaunchAssets(context.Background(), qemuBinary, spec); err != nil {
		t.Fatalf("prepareLaunchAssets returned error: %v", err)
	}
}

func TestQEMUPrepareLaunchAssetsSurfacesToolchainVerificationFailure(t *testing.T) {
	workRoot, qemuBinary, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	tamperedBinary := filepath.Join(workRoot, "fixtures", "qemu-system-x86_64-tampered")
	if err := os.WriteFile(tamperedBinary, []byte("#!/bin/sh\necho tampered\n"), 0o700); err != nil {
		t.Fatalf("WriteFile(tampered qemu fixture) returned error: %v", err)
	}
	controller := &qemuController{cfg: QEMUControllerConfig{WorkRoot: workRoot, QEMUBinary: qemuBinary, Now: time.Now}, instances: map[string]*qemuInstance{}}

	_, _, _, _, err := controller.prepareLaunchAssets(context.Background(), tamperedBinary, spec)
	if err == nil {
		t.Fatal("prepareLaunchAssets expected toolchain verification failure")
	}
	if !strings.Contains(err.Error(), launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch) {
		t.Fatalf("prepareLaunchAssets error = %v, want image descriptor signature mismatch", err)
	}
}

func TestQEMULaunchReceiptFailsClosedWithoutRuntimeCollectedAttestationEvidence(t *testing.T) {
	receipt, attestationInput, evidence := qemuRuntimeAttestationEvidenceWithoutCollection(t)
	if receipt.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("provisioning posture = %q, want %q", receipt.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}
	if evidence.Attestation != nil {
		t.Fatalf("attestation evidence = %#v, want nil until runtime-side evidence is collected", evidence.Attestation)
	}
	assertQEMUAttestationVerificationInvalid(t, evidence)
	if !strings.Contains(strings.Join(evidence.AttestationVerification.ReasonCodes, ","), "attestation_runtime_evidence_required") {
		t.Fatalf("reason codes = %v, want attestation_runtime_evidence_required", evidence.AttestationVerification.ReasonCodes)
	}
	if evidence.Launch.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("evidence launch provisioning posture = %q, want %q", evidence.Launch.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}
	if got, want := receipt.AttestationVerificationTimestamp, ""; got != want {
		t.Fatalf("attestation verification timestamp = %q, want %q", got, want)
	}
	if attestationInput == nil {
		t.Fatal("post-handshake attestation input missing")
	}
	if got, want := attestationInput.VerificationTimestamp, ""; got != want {
		t.Fatalf("post-handshake verification timestamp = %q, want %q", got, want)
	}
}

func qemuRuntimeAttestationEvidenceWithoutCollection(t *testing.T) (launcherbackend.BackendLaunchReceipt, *launcherbackend.PostHandshakeRuntimeAttestationInput, launcherbackend.RuntimeEvidenceSnapshot) {
	t.Helper()
	_, _, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}
	receipt, err := buildLaunchReceipt(spec, admission, "isolate-1", "session-1", strings.Repeat("a", 32), "9.0.0", "qemu-system-x86_64 9.0.0", nil)
	if err != nil {
		t.Fatalf("buildLaunchReceipt returned error: %v", err)
	}
	secureSession := mustRuntimeSecureSessionMaterialForTests(t, spec, receipt)
	summary, launchContextDigest, err := validateSecureSessionAndBuildSummary(receipt, secureSession)
	if err != nil {
		t.Fatalf("validateSecureSessionAndBuildSummary returned error: %v", err)
	}
	if err := recordValidatedSecureSession(&receipt, summary, launchContextDigest); err != nil {
		t.Fatalf("recordValidatedSecureSession returned error: %v", err)
	}
	attestationInput, err := buildPostHandshakeAttestationProgress(receipt, admission)
	if err != nil {
		t.Fatalf("buildPostHandshakeAttestationProgress returned error: %v", err)
	}
	facts := launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: receipt, PostHandshakeAttestationInput: attestationInput, HardeningPosture: buildHardeningPosture()}
	evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	return receipt, attestationInput, evidence
}

func assertQEMUAttestationVerificationInvalid(t *testing.T, evidence launcherbackend.RuntimeEvidenceSnapshot) {
	t.Helper()
	if evidence.AttestationVerification == nil {
		t.Fatal("attestation verification missing")
	}
	if evidence.AttestationVerification.VerificationResult != launcherbackend.AttestationVerificationResultInvalid {
		t.Fatalf("verification result = %q, want %q", evidence.AttestationVerification.VerificationResult, launcherbackend.AttestationVerificationResultInvalid)
	}
}

func TestApplyTrustedRuntimeAttestationFailsClosedWithoutLaunchContextDigest(t *testing.T) {
	_, _, spec := qemuToolchainVerificationLaunchSpecForTests(t)
	admission, err := launcherbackend.NewRuntimeAdmissionRecord(spec.Image)
	if err != nil {
		t.Fatalf("NewRuntimeAdmissionRecord returned error: %v", err)
	}

	receipt, err := buildLaunchReceipt(spec, admission, "isolate-1", "session-1", strings.Repeat("a", 32), "9.0.0", "qemu-system-x86_64 9.0.0", nil)
	if err != nil {
		t.Fatalf("buildLaunchReceipt returned error: %v", err)
	}
	secureSession := mustRuntimeSecureSessionMaterialForTests(t, spec, receipt)
	summary, launchContextDigest, err := validateSecureSessionAndBuildSummary(receipt, secureSession)
	if err != nil {
		t.Fatalf("validateSecureSessionAndBuildSummary returned error: %v", err)
	}
	if err := recordValidatedSecureSession(&receipt, summary, launchContextDigest); err != nil {
		t.Fatalf("recordValidatedSecureSession returned error: %v", err)
	}
	receipt.LaunchContextDigest = ""
	_, err = buildPostHandshakeAttestationProgress(receipt, admission)
	if err == nil {
		t.Fatal("buildPostHandshakeAttestationProgress expected missing launch context digest error")
	}
	if !strings.Contains(err.Error(), "session binding is required before attestation") {
		t.Fatalf("buildPostHandshakeAttestationProgress error = %q, want session binding failure", err.Error())
	}
}

func TestQEMURuntimePostHandshakeUpdateRequiresRuntimeProducedSecureSessionMaterial(t *testing.T) {
	spec, admission, receipt := runtimeAttestationReceiptFixtureForValidation(t)
	controller := &qemuController{cfg: QEMUControllerConfig{}, instances: map[string]*qemuInstance{}}
	_, err := controller.runtimePostHandshakeUpdate(preparedLaunchState{
		spec:      spec,
		receipt:   receipt,
		admission: admission,
		hardening: buildHardeningPosture(),
		material:  nil,
	})
	if err == nil {
		t.Fatal("runtimePostHandshakeUpdate expected missing runtime secure-session material error")
	}
	if !strings.Contains(err.Error(), launcherbackend.BackendErrorCodeHandshakeFailed) {
		t.Fatalf("runtimePostHandshakeUpdate error = %q, want handshake failure", err.Error())
	}
}

func TestQEMURuntimePostHandshakeUpdateRuntimeEvidenceCollectedTrueOnlyWithConcreteEvidence(t *testing.T) {
	spec, admission, receipt := runtimeAttestationReceiptFixtureForValidation(t)
	secureSession := mustRuntimeSecureSessionMaterialForTests(t, spec, receipt)
	controller := &qemuController{cfg: QEMUControllerConfig{}, instances: map[string]*qemuInstance{}}

	updateWithoutEvidence, err := controller.runtimePostHandshakeUpdate(qemuPreparedStateForEvidenceTest(spec, admission, receipt, secureSession, nil))
	assertQEMURuntimeEvidenceCollection(t, updateWithoutEvidence, err, false)

	updateWithEvidence, err := controller.runtimePostHandshakeUpdate(qemuPreparedStateForEvidenceTest(spec, admission, receipt, secureSession, &launcherbackend.PostHandshakeRuntimeAttestationInput{
		RunID:                        receipt.RunID,
		IsolateID:                    receipt.IsolateID,
		SessionID:                    receipt.SessionID,
		SessionNonce:                 receipt.SessionNonce,
		LaunchContextDigest:          secureSession.LaunchContext.LaunchContextDigest,
		HandshakeTranscriptHash:      secureSession.SessionReady.HandshakeTranscriptHash,
		IsolateSessionKeyIDValue:     secureSession.SessionReady.IsolateKeyIDValue,
		RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
		RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
		RuntimeEvidenceCollected:     true,
		AttestationSourceKind:        launcherbackend.AttestationSourceKindTrustedRuntime,
		MeasurementProfile:           admission.AttestationMeasurementProfile,
		FreshnessMaterial:            []string{"session_nonce"},
		FreshnessBindingClaims:       []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"},
		EvidenceClaimsDigest:         admission.AttestationExpectedMeasurementDigests[0],
	}))
	assertQEMURuntimeEvidenceCollection(t, updateWithEvidence, err, true)

	evidence, _, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(*updateWithEvidence.Facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if evidence.Launch.ProvisioningPosture == launcherbackend.ProvisioningPostureAttested {
		t.Fatal("attested posture must not be synthesized without valid verification")
	}
}

func qemuPreparedStateForEvidenceTest(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord, receipt launcherbackend.BackendLaunchReceipt, secureSession *launcherbackend.RuntimeSecureSessionMaterial, attestation *launcherbackend.PostHandshakeRuntimeAttestationInput) preparedLaunchState {
	return preparedLaunchState{
		spec:      spec,
		receipt:   receipt,
		admission: admission,
		hardening: buildHardeningPosture(),
		material:  &launcherbackend.RuntimePostHandshakeMaterial{SecureSession: secureSession, Attestation: attestation},
	}
}

func assertQEMURuntimeEvidenceCollection(t *testing.T, update RuntimeUpdate, err error, wantCollected bool) {
	t.Helper()
	if err != nil {
		t.Fatalf("runtimePostHandshakeUpdate returned error: %v", err)
	}
	if update.Facts == nil || update.Facts.PostHandshakeAttestationInput == nil {
		t.Fatal("runtimePostHandshakeUpdate missing post-handshake input")
	}
	if update.Facts.PostHandshakeAttestationInput.RuntimeEvidenceCollected != wantCollected {
		t.Fatalf("runtime evidence collected = %v, want %v", update.Facts.PostHandshakeAttestationInput.RuntimeEvidenceCollected, wantCollected)
	}
}

func TestQEMURuntimePostHandshakeUpdateFailsClosedOnInvalidRuntimeMaterial(t *testing.T) {
	spec, admission, receipt := runtimeAttestationReceiptFixtureForValidation(t)
	secureSession := mustRuntimeSecureSessionMaterialForTests(t, spec, receipt)
	controller := &qemuController{cfg: QEMUControllerConfig{}, instances: map[string]*qemuInstance{}}
	_, err := controller.runtimePostHandshakeUpdate(preparedLaunchState{
		spec:      spec,
		receipt:   receipt,
		admission: admission,
		hardening: buildHardeningPosture(),
		material: &launcherbackend.RuntimePostHandshakeMaterial{
			SecureSession: secureSession,
			Attestation: &launcherbackend.PostHandshakeRuntimeAttestationInput{
				RunID:                        receipt.RunID,
				IsolateID:                    receipt.IsolateID,
				SessionID:                    receipt.SessionID,
				SessionNonce:                 receipt.SessionNonce,
				LaunchContextDigest:          secureSession.LaunchContext.LaunchContextDigest,
				HandshakeTranscriptHash:      secureSession.SessionReady.HandshakeTranscriptHash,
				IsolateSessionKeyIDValue:     secureSession.SessionReady.IsolateKeyIDValue,
				RuntimeImageDescriptorDigest: receipt.RuntimeImageDescriptorDigest,
				RuntimeImageBootProfile:      receipt.RuntimeImageBootProfile,
				RuntimeEvidenceCollected:     true,
				AttestationSourceKind:        launcherbackend.AttestationSourceKindTrustedRuntime,
				MeasurementProfile:           admission.AttestationMeasurementProfile,
				EvidenceClaimsDigest:         "sha256:" + strings.Repeat("f", 64),
			},
		},
	})
	if err == nil {
		t.Fatal("runtimePostHandshakeUpdate expected invalid runtime material error")
	}
	if !strings.Contains(err.Error(), "runtime-reported evidence_claims_digest must bind to admitted runtime identity") {
		t.Fatalf("runtimePostHandshakeUpdate error = %q, want admitted runtime identity binding failure", err.Error())
	}
}

func TestBuildTerminalReportFailsClosedWhenErrorPresentAfterHello(t *testing.T) {
	report := buildTerminalReport(validSpecForTests(), launcherbackend.BackendLaunchReceipt{IsolateID: "iso-1", SessionID: "session-1"}, true, launcherbackend.BackendErrorCodeHandshakeFailed)
	if report.TerminationKind != launcherbackend.BackendTerminationKindFailed {
		t.Fatalf("termination kind = %q, want failed", report.TerminationKind)
	}
	if report.FailureReasonCode != launcherbackend.BackendErrorCodeHandshakeFailed {
		t.Fatalf("failure_reason_code = %q, want %q", report.FailureReasonCode, launcherbackend.BackendErrorCodeHandshakeFailed)
	}
}

func TestQEMUPrepareLaunchDirUsesUniquePathWithFixedClock(t *testing.T) {
	workRoot := t.TempDir()
	controller := &qemuController{cfg: QEMUControllerConfig{WorkRoot: workRoot, Now: func() time.Time { return time.Unix(123, 0).UTC() }}, instances: map[string]*qemuInstance{}}
	spec := validSpecForTests()

	first, err := controller.prepareLaunchDir(spec)
	if err != nil {
		t.Fatalf("prepareLaunchDir(first) returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(first) })
	second, err := controller.prepareLaunchDir(spec)
	if err != nil {
		t.Fatalf("prepareLaunchDir(second) returned error: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(second) })

	if first == second {
		t.Fatalf("prepareLaunchDir returned identical paths %q with fixed clock", first)
	}
	for _, dir := range []string{first, second} {
		manifest := filepath.Join(dir, "attachments")
		if _, err := os.Stat(manifest); err != nil {
			t.Fatalf("os.Stat(%q) returned error: %v", manifest, err)
		}
	}
	firstParts := strings.Split(filepath.Clean(first), string(os.PathSeparator))
	secondParts := strings.Split(filepath.Clean(second), string(os.PathSeparator))
	if len(firstParts) != len(secondParts) {
		t.Fatalf("launch dir depth mismatch: %q vs %q", first, second)
	}
	if !reflect.DeepEqual(firstParts[:len(firstParts)-1], secondParts[:len(secondParts)-1]) {
		t.Fatalf("launch dir parents differ: %q vs %q", first, second)
	}
}

func qemuToolchainVerificationLaunchSpecForTests(t *testing.T) (string, string, launcherbackend.BackendLaunchSpec) {
	t.Helper()
	workRoot := t.TempDir()
	qemuBinary := seedVerticalSliceQEMUFixture(t, workRoot)
	spec := validSpecForTests()
	materializeComponentDigests(t, workRoot, &spec.Image)
	seedRuntimeImageVerificationAssets(t, workRoot, qemuBinary, &spec)
	return workRoot, qemuBinary, spec
}
