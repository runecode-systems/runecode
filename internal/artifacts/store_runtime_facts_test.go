package artifacts

import (
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestRecordRuntimeEvidenceStatePersistsAcrossReload(t *testing.T) {
	store := newTestStore(t)
	facts, evidence, lifecycle := recordRuntimeEvidenceFixture(t, store, "run-runtime-persist")
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
	return facts
}
