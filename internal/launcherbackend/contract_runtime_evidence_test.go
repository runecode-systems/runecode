package launcherbackend

import "testing"

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
