package launcherbackend

import "testing"

func TestDefaultRuntimeFactsUsesUnknownClosedVocabulary(t *testing.T) {
	facts := DefaultRuntimeFacts("run-1")
	if facts.LaunchReceipt.RunID != "run-1" {
		t.Fatalf("run_id = %q, want run-1", facts.LaunchReceipt.RunID)
	}
	if facts.LaunchReceipt.BackendKind != BackendKindUnknown {
		t.Fatalf("backend_kind = %q, want %q", facts.LaunchReceipt.BackendKind, BackendKindUnknown)
	}
	if facts.LaunchReceipt.IsolationAssuranceLevel != IsolationAssuranceUnknown {
		t.Fatalf("isolation_assurance_level = %q, want %q", facts.LaunchReceipt.IsolationAssuranceLevel, IsolationAssuranceUnknown)
	}
	if facts.LaunchReceipt.ProvisioningPosture != ProvisioningPostureUnknown {
		t.Fatalf("provisioning_posture = %q, want %q", facts.LaunchReceipt.ProvisioningPosture, ProvisioningPostureUnknown)
	}
	if facts.LaunchReceipt.SessionSecurity == nil || !facts.LaunchReceipt.SessionSecurity.Degraded {
		t.Fatalf("session_security = %#v, want degraded default posture", facts.LaunchReceipt.SessionSecurity)
	}
	if facts.LaunchReceipt.Lifecycle == nil || facts.LaunchReceipt.Lifecycle.CurrentState != BackendLifecycleStatePlanned || !facts.LaunchReceipt.Lifecycle.TerminateBetweenSteps {
		t.Fatalf("backend_lifecycle = %#v, want planned + terminate_between_steps=true", facts.LaunchReceipt.Lifecycle)
	}
	if facts.HardeningPosture.Effective != HardeningEffectiveDegraded {
		t.Fatalf("hardening_posture.effective = %q, want %q", facts.HardeningPosture.Effective, HardeningEffectiveDegraded)
	}
	if !facts.HardeningPosture.IsDegraded() {
		t.Fatal("hardening_posture should be degraded by default")
	}
}

func TestAppliedHardeningPostureNormalizedFailClosedUnknownAndNone(t *testing.T) {
	posture := AppliedHardeningPosture{
		Requested:                 "unknown",
		Effective:                 "hardened",
		ExecutionIdentityPosture:  "none",
		FilesystemExposurePosture: "broad",
		NetworkExposurePosture:    "open",
		SyscallFilteringPosture:   "none",
		DeviceSurfacePosture:      "broad",
		ControlChannelKind:        "unknown",
		AccelerationKind:          "none",
	}
	normalized := posture.Normalized()
	if normalized.Effective != HardeningEffectiveDegraded {
		t.Fatalf("effective = %q, want %q", normalized.Effective, HardeningEffectiveDegraded)
	}
	if !normalized.IsDegraded() {
		t.Fatal("normalized posture should be degraded")
	}
	if len(normalized.DegradedReasons) == 0 {
		t.Fatal("degraded_reasons should not be empty for insecure posture")
	}
}

func TestAppliedHardeningPostureValidateRejectsContradictionsAndPathLeakage(t *testing.T) {
	if err := (AppliedHardeningPosture{Requested: "strict", Effective: "degraded", DegradedReasons: []string{"policy_mismatch"}}).Validate(); err == nil {
		t.Fatal("Validate expected requested vocabulary rejection")
	}
	if err := (AppliedHardeningPosture{Requested: "hardened", Effective: "partial", DegradedReasons: []string{"policy_mismatch"}}).Validate(); err == nil {
		t.Fatal("Validate expected effective vocabulary rejection")
	}
	if err := (AppliedHardeningPosture{Requested: "hardened", Effective: "degraded"}).Validate(); err == nil {
		t.Fatal("Validate expected degraded_reasons requirement")
	}
	if err := (AppliedHardeningPosture{Requested: "hardened", Effective: "hardened", DegradedReasons: []string{"seccomp_unavailable"}}).Validate(); err == nil {
		t.Fatal("Validate expected contradiction for effective=hardened with degraded reasons")
	}
	if err := (AppliedHardeningPosture{Requested: "hardened", Effective: "degraded", DegradedReasons: []string{"/etc/seccomp/missing"}}).Validate(); err == nil {
		t.Fatal("Validate expected host-path rejection in degraded_reasons")
	}
	if err := (AppliedHardeningPosture{Requested: "hardened", Effective: "degraded", DegradedReasons: []string{"seccomp_unavailable"}, BackendEvidenceRefs: []string{"/usr/bin/qemu"}}).Validate(); err == nil {
		t.Fatal("Validate expected host-path rejection in backend_evidence_refs")
	}
}
