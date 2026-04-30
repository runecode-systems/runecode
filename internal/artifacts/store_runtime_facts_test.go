package artifacts

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestRecordRuntimeEvidenceStatePersistsAcrossReload(t *testing.T) {
	store := newTestStore(t)
	facts, evidence, lifecycle := recordRuntimeEvidenceFixture(t, store, "run-runtime-persist")
	assertSessionStateBoundFromRuntimeFacts(t, store, "session-1", "run-runtime-persist")
	if err := store.MarkRuntimeAuditEventEmitted("run-runtime-persist", "isolate_session_started", evidence.Session.EvidenceDigest); err != nil {
		t.Fatalf("MarkRuntimeAuditEventEmitted(started) returned error: %v", err)
	}
	if err := store.MarkRuntimeAuditEventEmitted("run-runtime-persist", "isolate_session_bound", evidence.Session.EvidenceDigest); err != nil {
		t.Fatalf("MarkRuntimeAuditEventEmitted(bound) returned error: %v", err)
	}
	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore(reload) returned error: %v", err)
	}
	persistedFacts, persistedEvidence, persistedLifecycle, persistedAudit, ok := reloaded.RuntimeEvidenceState("run-runtime-persist")
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	assertPersistedRuntimeEvidence(t, persistedFacts, facts, persistedEvidence, evidence, persistedLifecycle, lifecycle)
	assertPersistedRuntimeAuditMarkers(t, persistedAudit, evidence.Session.EvidenceDigest)
	assertSessionStateBoundFromRuntimeFacts(t, reloaded, "session-1", "run-runtime-persist")
}

func TestUpdateRuntimeLifecycleStateProjectsIntoPersistedFacts(t *testing.T) {
	store := newTestStore(t)
	_, _, _ = recordRuntimeEvidenceFixture(t, store, "run-runtime-lifecycle")
	next := launcherbackend.RuntimeLifecycleState{
		ProvisioningPosture:         launcherbackend.ProvisioningPostureTOFU,
		ProvisioningPostureDegraded: true,
		ProvisioningDegradedReasons: []string{"backend_temporarily_untrusted"},
		LaunchFailureReasonCode:     launcherbackend.BackendErrorCodeAccelerationUnavailable,
	}
	if err := store.UpdateRuntimeLifecycleState("run-runtime-lifecycle", next); err != nil {
		t.Fatalf("UpdateRuntimeLifecycleState returned error: %v", err)
	}
	persistedFacts, persistedEvidence, persistedLifecycle, _, ok := store.RuntimeEvidenceState("run-runtime-lifecycle")
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	if persistedLifecycle.LaunchFailureReasonCode != launcherbackend.BackendErrorCodeAccelerationUnavailable {
		t.Fatalf("persisted launch_failure_reason_code = %q, want %q", persistedLifecycle.LaunchFailureReasonCode, launcherbackend.BackendErrorCodeAccelerationUnavailable)
	}
	if persistedFacts.LaunchReceipt.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("persisted facts provisioning_posture = %q, want %q", persistedFacts.LaunchReceipt.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
	}
	if persistedFacts.LaunchReceipt.LaunchFailureReasonCode != launcherbackend.BackendErrorCodeAccelerationUnavailable {
		t.Fatalf("persisted facts launch_failure_reason_code = %q, want %q", persistedFacts.LaunchReceipt.LaunchFailureReasonCode, launcherbackend.BackendErrorCodeAccelerationUnavailable)
	}
	recomputedEvidence, recomputedLifecycle, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(persistedFacts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if persistedEvidence.Launch.EvidenceDigest != recomputedEvidence.Launch.EvidenceDigest {
		t.Fatalf("persisted launch evidence digest = %q, want %q", persistedEvidence.Launch.EvidenceDigest, recomputedEvidence.Launch.EvidenceDigest)
	}
	if persistedLifecycle.LaunchFailureReasonCode != recomputedLifecycle.LaunchFailureReasonCode {
		t.Fatalf("persisted lifecycle launch_failure_reason_code = %q, want %q", persistedLifecycle.LaunchFailureReasonCode, recomputedLifecycle.LaunchFailureReasonCode)
	}
}

func TestUpdateRuntimeLifecycleStateClearsPersistedLaunchFailureReasonCode(t *testing.T) {
	store := newTestStore(t)
	_, _, _ = recordRuntimeEvidenceFixture(t, store, "run-runtime-lifecycle-clear-reason")

	if err := store.UpdateRuntimeLifecycleState("run-runtime-lifecycle-clear-reason", launcherbackend.RuntimeLifecycleState{
		LaunchFailureReasonCode: launcherbackend.BackendErrorCodeAccelerationUnavailable,
	}); err != nil {
		t.Fatalf("UpdateRuntimeLifecycleState(set reason) returned error: %v", err)
	}

	if err := store.UpdateRuntimeLifecycleState("run-runtime-lifecycle-clear-reason", launcherbackend.RuntimeLifecycleState{}); err != nil {
		t.Fatalf("UpdateRuntimeLifecycleState(clear reason) returned error: %v", err)
	}

	persistedFacts, _, persistedLifecycle, _, ok := store.RuntimeEvidenceState("run-runtime-lifecycle-clear-reason")
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	if persistedFacts.LaunchReceipt.LaunchFailureReasonCode != "" {
		t.Fatalf("persisted facts launch_failure_reason_code = %q, want empty", persistedFacts.LaunchReceipt.LaunchFailureReasonCode)
	}
	if persistedLifecycle.LaunchFailureReasonCode != "" {
		t.Fatalf("persisted lifecycle launch_failure_reason_code = %q, want empty", persistedLifecycle.LaunchFailureReasonCode)
	}
}

func TestRecordRuntimeEvidenceStateUsesCachedAttestationVerificationOnReplay(t *testing.T) {
	store := newTestStore(t)
	runID := "run-runtime-attestation-cache-hit"

	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-cache-hit", "policy")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if evidence.AttestationVerification == nil {
		t.Fatal("expected initial attestation verification record")
	}
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(initial) returned error: %v", err)
	}

	replayedFacts := replayedRuntimeFactsWithoutVerifierIdentity(facts)
	replayedEvidence, replayedLifecycle := splitRuntimeEvidenceStateForStoreTest(t, replayedFacts)
	if replayedEvidence.AttestationVerification == nil {
		t.Fatal("expected fail-closed placeholder verification before cache application")
	}
	if replayedEvidence.AttestationVerification.VerifierPolicyDigest != "" {
		t.Fatal("expected replayed placeholder verification to have empty policy digest")
	}
	if err := store.RecordRuntimeEvidenceState(runID, replayedFacts, replayedEvidence, replayedLifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(replayed) returned error: %v", err)
	}
	assertPersistedInvalidAttestationVerification(t, store, runID)
}

func TestRecordRuntimeEvidenceStateInvalidatesAttestationVerificationCacheOnAuthorityChange(t *testing.T) {
	store := newTestStore(t)
	runID := "run-runtime-attestation-cache-invalidate"

	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-1", "policy")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(initial) returned error: %v", err)
	}

	changedFacts := facts
	changedFacts.LaunchReceipt.AuthorityStateDigest = testDigest("authority-2")
	changedFacts = replayedRuntimeFactsWithoutVerifierIdentity(changedFacts)
	changedEvidence, changedLifecycle := splitRuntimeEvidenceStateForStoreTest(t, changedFacts)
	if err := store.RecordRuntimeEvidenceState(runID, changedFacts, changedEvidence, changedLifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(changed) returned error: %v", err)
	}

	_, persistedEvidence, _, _, ok := store.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	if persistedEvidence.AttestationVerification == nil {
		t.Fatal("expected fail-closed attestation verification when cache key changes")
	}
	if persistedEvidence.AttestationVerification.VerifierPolicyDigest != "" {
		t.Fatal("expected cache miss when authority digest changes")
	}
}

func TestRecordRuntimeEvidenceStateAttestationVerificationCacheSurvivesReload(t *testing.T) {
	store := newTestStore(t)
	runID := "run-runtime-attestation-cache-reload"

	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-reload", "policy-reload")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(initial) returned error: %v", err)
	}

	reloaded, err := NewStore(store.rootDir)
	if err != nil {
		t.Fatalf("NewStore(reload) returned error: %v", err)
	}
	replayedFacts := replayedRuntimeFactsWithoutVerifierIdentity(facts)
	replayedEvidence, replayedLifecycle := splitRuntimeEvidenceStateForStoreTest(t, replayedFacts)
	if err := reloaded.RecordRuntimeEvidenceState(runID, replayedFacts, replayedEvidence, replayedLifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(replayed) returned error: %v", err)
	}
	_, persistedEvidence, _, _, ok := reloaded.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	if persistedEvidence.AttestationVerification == nil {
		t.Fatal("expected attestation verification from persisted cache after reload")
	}
}

func TestRecordRuntimeEvidenceStateAttestationVerificationCacheDoesNotApplyWithoutMeasurementProfile(t *testing.T) {
	store := newTestStore(t)
	runID := "run-runtime-attestation-cache-missing-measurement-profile"
	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-profile", "policy-profile")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(initial) returned error: %v", err)
	}
	replayedFacts := replayedRuntimeFactsWithoutVerifierIdentity(facts)
	replayedFacts.LaunchReceipt.AttestationMeasurementProfile = ""
	replayedEvidence, replayedLifecycle := splitRuntimeEvidenceStateForStoreTest(t, replayedFacts)
	if err := store.RecordRuntimeEvidenceState(runID, replayedFacts, replayedEvidence, replayedLifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(replayed) returned error: %v", err)
	}
	_, persistedEvidence, _, _, ok := store.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	if persistedEvidence.AttestationVerification == nil {
		t.Fatal("expected fail-closed attestation verification to remain persisted")
	}
	if persistedEvidence.AttestationVerification.VerifierPolicyDigest != "" {
		t.Fatalf("expected cache miss without measurement profile, got verifier policy digest %q", persistedEvidence.AttestationVerification.VerifierPolicyDigest)
	}
}

func TestRecordRuntimeEvidenceStateDoesNotApplyCacheToPartialVerificationRecord(t *testing.T) {
	store := newTestStore(t)
	runID := "run-runtime-attestation-cache-partial-verification"
	facts := runtimeFactsWithValidAttestationVerification(runID, "authority-partial", "policy-partial")
	evidence, lifecycle := splitRuntimeEvidenceStateForStoreTest(t, facts)
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(initial) returned error: %v", err)
	}

	replayedFacts := replayedRuntimeFactsWithoutVerifierIdentity(facts)
	replayedEvidence, replayedLifecycle := splitRuntimeEvidenceStateForStoreTest(t, replayedFacts)
	if replayedEvidence.AttestationVerification == nil {
		t.Fatal("expected attestation verification placeholder")
	}
	replayedEvidence.AttestationVerification.AttestationEvidenceDigest = evidence.Attestation.EvidenceDigest
	replayedEvidence.AttestationVerification.ReplayIdentityDigest = evidence.AttestationVerification.ReplayIdentityDigest
	if err := store.RecordRuntimeEvidenceState(runID, replayedFacts, replayedEvidence, replayedLifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState(replayed) returned error: %v", err)
	}

	_, persistedEvidence, _, _, ok := store.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	if persistedEvidence.AttestationVerification == nil {
		t.Fatal("expected persisted attestation verification")
	}
	if persistedEvidence.AttestationVerification.VerifierPolicyDigest != "" {
		t.Fatalf("expected cache not to overwrite partial verification record, got verifier policy digest %q", persistedEvidence.AttestationVerification.VerifierPolicyDigest)
	}
	if persistedEvidence.AttestationVerification.ReplayIdentityDigest == "" {
		t.Fatal("expected partial verification replay identity to be preserved")
	}
}

func TestAttestationVerificationCacheKeyFromFieldsAvoidsDelimiterCollisions(t *testing.T) {
	keyA := attestationVerificationCacheKeyFromFields(testDigest("a"), testDigest("b"), "foo|bar")
	keyB := attestationVerificationCacheKeyFromFields(testDigest("a"), testDigest("b"), "foo")
	if keyA == "" || keyB == "" {
		t.Fatalf("cache key generation failed: keyA=%q keyB=%q", keyA, keyB)
	}
	if keyA == keyB {
		t.Fatalf("cache keys collided: %q", keyA)
	}
}

func recordRuntimeEvidenceFixture(t *testing.T, store *Store, runID string) (launcherbackend.RuntimeFactsSnapshot, launcherbackend.RuntimeEvidenceSnapshot, launcherbackend.RuntimeLifecycleState) {
	t.Helper()
	facts := runtimeFactsFixtureForStoreRuntimeTests(runID)
	evidence, lifecycle, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	if err := store.RecordRuntimeEvidenceState(runID, facts, evidence, lifecycle); err != nil {
		t.Fatalf("RecordRuntimeEvidenceState returned error: %v", err)
	}
	return facts, evidence, lifecycle
}

func splitRuntimeEvidenceStateForStoreTest(t *testing.T, facts launcherbackend.RuntimeFactsSnapshot) (launcherbackend.RuntimeEvidenceSnapshot, launcherbackend.RuntimeLifecycleState) {
	t.Helper()
	evidence, lifecycle, err := launcherbackend.SplitRuntimeFactsEvidenceAndLifecycle(facts)
	if err != nil {
		t.Fatalf("SplitRuntimeFactsEvidenceAndLifecycle returned error: %v", err)
	}
	return evidence, lifecycle
}

func runtimeFactsWithValidAttestationVerification(runID string, authoritySeed string, policySeed string) launcherbackend.RuntimeFactsSnapshot {
	facts := runtimeFactsFixtureForStoreRuntimeTests(runID)
	facts.LaunchReceipt.AuthorityStateDigest = DigestBytes([]byte(authoritySeed))
	facts.LaunchReceipt.AttestationVerifierPolicyID = "runtime_asset_admission_identity"
	facts.LaunchReceipt.AttestationVerifierPolicyDigest = DigestBytes([]byte(policySeed))
	facts.LaunchReceipt.AttestationVerificationResult = launcherbackend.AttestationVerificationResultValid
	facts.LaunchReceipt.AttestationReplayVerdict = launcherbackend.AttestationReplayVerdictOriginal
	return facts
}

func replayedRuntimeFactsWithoutVerifierIdentity(facts launcherbackend.RuntimeFactsSnapshot) launcherbackend.RuntimeFactsSnapshot {
	replayedFacts := facts
	replayedFacts.LaunchReceipt.AttestationVerifierPolicyID = ""
	replayedFacts.LaunchReceipt.AttestationVerifierPolicyDigest = ""
	replayedFacts.LaunchReceipt.AttestationVerificationRulesVersion = ""
	replayedFacts.LaunchReceipt.AttestationVerificationTimestamp = ""
	replayedFacts.LaunchReceipt.AttestationVerificationResult = ""
	replayedFacts.LaunchReceipt.AttestationVerificationReasonCodes = nil
	replayedFacts.LaunchReceipt.AttestationReplayVerdict = ""
	return replayedFacts
}

func assertPersistedInvalidAttestationVerification(t *testing.T, store *Store, runID string) {
	t.Helper()
	_, persistedEvidence, _, _, ok := store.RuntimeEvidenceState(runID)
	if !ok {
		t.Fatal("RuntimeEvidenceState = not found, want persisted runtime state")
	}
	if persistedEvidence.AttestationVerification == nil {
		t.Fatal("expected fail-closed attestation verification to remain persisted")
	}
	if persistedEvidence.AttestationVerification.VerificationResult != launcherbackend.AttestationVerificationResultInvalid {
		t.Fatalf("verification result = %q, want %q", persistedEvidence.AttestationVerification.VerificationResult, launcherbackend.AttestationVerificationResultInvalid)
	}
	if persistedEvidence.AttestationVerification.VerifierPolicyDigest != "" {
		t.Fatalf("expected cache not to overwrite explicit fail-closed verification, got verifier policy digest %q", persistedEvidence.AttestationVerification.VerifierPolicyDigest)
	}
}

func assertPersistedRuntimeEvidence(t *testing.T, persistedFacts, facts launcherbackend.RuntimeFactsSnapshot, persistedEvidence, evidence launcherbackend.RuntimeEvidenceSnapshot, persistedLifecycle, lifecycle launcherbackend.RuntimeLifecycleState) {
	t.Helper()
	if persistedEvidence.Launch.EvidenceDigest != evidence.Launch.EvidenceDigest {
		t.Fatalf("launch evidence digest = %q, want %q", persistedEvidence.Launch.EvidenceDigest, evidence.Launch.EvidenceDigest)
	}
	if persistedEvidence.Hardening.EvidenceDigest != evidence.Hardening.EvidenceDigest {
		t.Fatalf("hardening evidence digest = %q, want %q", persistedEvidence.Hardening.EvidenceDigest, evidence.Hardening.EvidenceDigest)
	}
	if persistedEvidence.Session == nil || persistedEvidence.Session.EvidenceDigest != evidence.Session.EvidenceDigest {
		t.Fatalf("session evidence digest = %#v, want %q", persistedEvidence.Session, evidence.Session.EvidenceDigest)
	}
	if persistedEvidence.Attestation == nil || persistedEvidence.Attestation.EvidenceDigest != evidence.Attestation.EvidenceDigest {
		t.Fatalf("attestation evidence digest = %#v, want %q", persistedEvidence.Attestation, evidence.Attestation.EvidenceDigest)
	}
	if persistedFacts.LaunchReceipt.BackendKind != facts.LaunchReceipt.BackendKind {
		t.Fatalf("persisted backend_kind = %q, want %q", persistedFacts.LaunchReceipt.BackendKind, facts.LaunchReceipt.BackendKind)
	}
	if persistedLifecycle.ProvisioningPosture != lifecycle.ProvisioningPosture {
		t.Fatalf("persisted lifecycle provisioning_posture = %q, want %q", persistedLifecycle.ProvisioningPosture, lifecycle.ProvisioningPosture)
	}
}

func assertPersistedRuntimeAuditMarkers(t *testing.T, persistedAudit RuntimeAuditEmissionState, digest string) {
	t.Helper()
	if persistedAudit.LastIsolateSessionStartedDigest != digest {
		t.Fatalf("started digest marker = %q, want %q", persistedAudit.LastIsolateSessionStartedDigest, digest)
	}
	if persistedAudit.LastIsolateSessionBoundDigest != digest {
		t.Fatalf("bound digest marker = %q, want %q", persistedAudit.LastIsolateSessionBoundDigest, digest)
	}
}

func runtimeFactsFixtureForStoreRuntimeTests(runID string) launcherbackend.RuntimeFactsSnapshot {
	facts := launcherbackend.DefaultRuntimeFacts(runID)
	facts.LaunchReceipt.BackendKind = launcherbackend.BackendKindMicroVM
	facts.LaunchReceipt.IsolationAssuranceLevel = launcherbackend.IsolationAssuranceIsolated
	facts.LaunchReceipt.ProvisioningPosture = launcherbackend.ProvisioningPostureAttested
	facts.LaunchReceipt.IsolateID = "isolate-1"
	facts.LaunchReceipt.SessionID = "session-1"
	facts.LaunchReceipt.SessionNonce = "nonce-runtime-test"
	facts.LaunchReceipt.LaunchContextDigest = testDigest("4")
	facts.LaunchReceipt.HandshakeTranscriptHash = testDigest("5")
	facts.LaunchReceipt.IsolateSessionKeyIDValue = strings.Repeat("f", 64)
	facts.LaunchReceipt.RuntimeImageDescriptorDigest = testDigest("6")
	facts.LaunchReceipt.RuntimeImageBootProfile = launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1
	facts.LaunchReceipt.BootComponentDigests = []string{testDigest("7")}
	facts.LaunchReceipt.AttestationEvidenceSourceKind = launcherbackend.AttestationSourceKindTPMQuote
	facts.LaunchReceipt.AttestationMeasurementProfile = "microvm-boot-v1"
	facts.LaunchReceipt.AttestationFreshnessMaterial = []string{"quote_nonce"}
	facts.LaunchReceipt.AttestationFreshnessBindingClaims = []string{"session_nonce", "handshake_transcript_hash"}
	facts.LaunchReceipt.AttestationEvidenceClaimsDigest = testDigest("8")
	return facts
}

func assertSessionStateBoundFromRuntimeFacts(t *testing.T, store *Store, sessionID, runID string) {
	t.Helper()
	session, ok := store.SessionState(sessionID)
	if !ok {
		t.Fatalf("SessionState(%q) = not found, want durable session", sessionID)
	}
	if session.SessionID != sessionID {
		t.Fatalf("session_id = %q, want %q", session.SessionID, sessionID)
	}
	if session.CreatedByRunID != runID {
		t.Fatalf("created_by_run_id = %q, want %q", session.CreatedByRunID, runID)
	}
	if session.LastActivityKind != "session_created" && session.LastActivityKind != "run_progress" {
		t.Fatalf("last_activity_kind = %q, want runtime-derived session activity", session.LastActivityKind)
	}
	if len(session.LinkedRunIDs) != 1 || session.LinkedRunIDs[0] != runID {
		t.Fatalf("linked_run_ids = %+v, want [%s]", session.LinkedRunIDs, runID)
	}
	if session.TurnCount != 0 {
		t.Fatalf("turn_count = %d, want 0 before transcript messages", session.TurnCount)
	}
}
