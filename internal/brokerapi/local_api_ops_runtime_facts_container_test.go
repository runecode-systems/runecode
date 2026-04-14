package brokerapi

import (
	"slices"
	"testing"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func TestRunDetailRuntimeFactsContainerProjectionUsesNotApplicablePostureVocabulary(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-container-facts"
	putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	if err := s.RecordRuntimeFacts(runID, containerRuntimeFactsFixtureForPostureVocab(runID)); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	runGet := mustRunGetForRuntimeFactsRestartTest(t, s, runID)
	assertContainerSummaryPostureVocabulary(t, runGet.Run.Summary)
	assertContainerAuthoritativePostureVocabulary(t, runGet.Run.AuthoritativeState)
}

func TestRunDetailRuntimeFactsContainerProjectionFailsClosedForNonWorkspaceRoleAndWeakNetworking(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	const runID = "run-container-role-scope"
	putRunScopedArtifactForLocalOpsTest(t, s, runID, "step-1")
	if err := s.RecordRuntimeFacts(runID, containerRuntimeFactsFixtureForRoleScope(runID)); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	runGet := mustRunGetForRuntimeFactsRestartTest(t, s, runID)
	hardening := requireHardeningStateMap(t, runGet.Run.AuthoritativeState)
	if hardening["effective"] != launcherbackend.HardeningEffectiveDegraded {
		t.Fatalf("applied_hardening_posture.effective = %v, want %q", hardening["effective"], launcherbackend.HardeningEffectiveDegraded)
	}
	reasons, ok := hardening["degraded_reasons"].([]string)
	if !ok {
		t.Fatalf("applied_hardening_posture.degraded_reasons = %T, want []string", hardening["degraded_reasons"])
	}
	assertReasonsContainAll(t, reasons,
		"container_role_family_not_supported_v0",
		"network_namespace_shared",
		"workspace_network_default_must_be_none_or_loopback",
		"egress_enforcement_in_container",
	)
}

func containerRuntimeFactsFixtureForPostureVocab(runID string) launcherbackend.RuntimeFactsSnapshot {
	return launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt: launcherbackend.BackendLaunchReceipt{
			RunID:                   runID,
			StageID:                 "artifact_flow",
			RoleInstanceID:          "workspace-1",
			RoleFamily:              "workspace",
			BackendKind:             launcherbackend.BackendKindContainer,
			IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded,
		},
		HardeningPosture: launcherbackend.AppliedHardeningPosture{
			Requested:                 launcherbackend.HardeningRequestedHardened,
			Effective:                 launcherbackend.HardeningEffectiveHardened,
			ExecutionIdentityPosture:  launcherbackend.HardeningExecutionIdentityUnprivileged,
			RootlessPosture:           launcherbackend.HardeningRootlessBestEffort,
			FilesystemExposurePosture: launcherbackend.HardeningFilesystemExposureRestricted,
			WritableLayersPosture:     launcherbackend.HardeningWritableLayersEphemeral,
			NetworkExposurePosture:    launcherbackend.HardeningNetworkExposureNone,
			NetworkNamespacePosture:   launcherbackend.HardeningNetworkNamespacePerRole,
			NetworkDefaultPosture:     launcherbackend.HardeningNetworkDefaultLoopbackOnly,
			EgressEnforcementPosture:  launcherbackend.HardeningEgressEnforcementHostLevel,
			SyscallFilteringPosture:   launcherbackend.HardeningSyscallFilteringSeccomp,
			CapabilitiesPosture:       launcherbackend.HardeningCapabilitiesDropped,
			DeviceSurfacePosture:      launcherbackend.HardeningDeviceSurfaceAllowlist,
			ControlChannelKind:        launcherbackend.TransportKindNotApplicable,
			AccelerationKind:          launcherbackend.AccelerationKindNotApplicable,
			BackendEvidenceRefs:       []string{"container-hardening:mvp-v0"},
		},
	}
}

func containerRuntimeFactsFixtureForRoleScope(runID string) launcherbackend.RuntimeFactsSnapshot {
	return launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt: launcherbackend.BackendLaunchReceipt{
			RunID:                   runID,
			StageID:                 "artifact_flow",
			RoleInstanceID:          "gateway-1",
			RoleFamily:              "gateway",
			BackendKind:             launcherbackend.BackendKindContainer,
			IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded,
		},
		HardeningPosture: launcherbackend.AppliedHardeningPosture{
			Requested:                 launcherbackend.HardeningRequestedHardened,
			Effective:                 launcherbackend.HardeningEffectiveHardened,
			ExecutionIdentityPosture:  launcherbackend.HardeningExecutionIdentityUnprivileged,
			RootlessPosture:           launcherbackend.HardeningRootlessEnabled,
			FilesystemExposurePosture: launcherbackend.HardeningFilesystemExposureRestricted,
			WritableLayersPosture:     launcherbackend.HardeningWritableLayersEphemeral,
			NetworkExposurePosture:    launcherbackend.HardeningNetworkExposureRestricted,
			NetworkNamespacePosture:   launcherbackend.HardeningNetworkNamespaceShared,
			NetworkDefaultPosture:     launcherbackend.HardeningNetworkDefaultEgress,
			EgressEnforcementPosture:  launcherbackend.HardeningEgressEnforcementInContainer,
			SyscallFilteringPosture:   launcherbackend.HardeningSyscallFilteringSeccomp,
			CapabilitiesPosture:       launcherbackend.HardeningCapabilitiesDropped,
			DeviceSurfacePosture:      launcherbackend.HardeningDeviceSurfaceAllowlist,
			ControlChannelKind:        launcherbackend.TransportKindNotApplicable,
			AccelerationKind:          launcherbackend.AccelerationKindNotApplicable,
		},
	}
}

func assertContainerSummaryPostureVocabulary(t *testing.T, summary RunSummary) {
	t.Helper()
	if summary.BackendKind != launcherbackend.BackendKindContainer {
		t.Fatalf("summary.backend_kind = %q, want %q", summary.BackendKind, launcherbackend.BackendKindContainer)
	}
	if summary.ProvisioningPosture != launcherbackend.ProvisioningPostureNotApplicable {
		t.Fatalf("summary.provisioning_posture = %q, want %q", summary.ProvisioningPosture, launcherbackend.ProvisioningPostureNotApplicable)
	}
	if !summary.RuntimePostureDegraded {
		t.Fatal("summary.runtime_posture_degraded = false, want true for container reduced assurance")
	}
}

func assertContainerAuthoritativePostureVocabulary(t *testing.T, state map[string]any) {
	t.Helper()
	if state["runtime_posture_degraded"] != true {
		t.Fatalf("authoritative_state.runtime_posture_degraded = %v, want true", state["runtime_posture_degraded"])
	}
	if state["provisioning_posture"] != launcherbackend.ProvisioningPostureNotApplicable {
		t.Fatalf("authoritative_state.provisioning_posture = %v, want %q", state["provisioning_posture"], launcherbackend.ProvisioningPostureNotApplicable)
	}
	if state["transport_kind"] != launcherbackend.TransportKindNotApplicable {
		t.Fatalf("authoritative_state.transport_kind = %v, want %q", state["transport_kind"], launcherbackend.TransportKindNotApplicable)
	}
	if state["acceleration_kind"] != launcherbackend.AccelerationKindNotApplicable {
		t.Fatalf("authoritative_state.acceleration_kind = %v, want %q", state["acceleration_kind"], launcherbackend.AccelerationKindNotApplicable)
	}
	if state["hypervisor_implementation"] != launcherbackend.HypervisorImplementationNotApplicable {
		t.Fatalf("authoritative_state.hypervisor_implementation = %v, want %q", state["hypervisor_implementation"], launcherbackend.HypervisorImplementationNotApplicable)
	}
	if _, ok := state["qemu_provenance"]; ok {
		t.Fatal("authoritative_state.qemu_provenance should be omitted for container runtime facts")
	}
	hardening := requireHardeningStateMap(t, state)
	assertContainerHardeningVocabulary(t, hardening)
}

func requireHardeningStateMap(t *testing.T, state map[string]any) map[string]any {
	t.Helper()
	hardening, ok := state["applied_hardening_posture"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.applied_hardening_posture = %T, want map", state["applied_hardening_posture"])
	}
	return hardening
}

func assertContainerHardeningVocabulary(t *testing.T, hardening map[string]any) {
	t.Helper()
	if hardening["rootless_posture"] != launcherbackend.HardeningRootlessBestEffort {
		t.Fatalf("applied_hardening_posture.rootless_posture = %v, want %q", hardening["rootless_posture"], launcherbackend.HardeningRootlessBestEffort)
	}
	if hardening["writable_layers_posture"] != launcherbackend.HardeningWritableLayersEphemeral {
		t.Fatalf("applied_hardening_posture.writable_layers_posture = %v, want %q", hardening["writable_layers_posture"], launcherbackend.HardeningWritableLayersEphemeral)
	}
	if hardening["network_namespace_posture"] != launcherbackend.HardeningNetworkNamespacePerRole {
		t.Fatalf("applied_hardening_posture.network_namespace_posture = %v, want %q", hardening["network_namespace_posture"], launcherbackend.HardeningNetworkNamespacePerRole)
	}
	if hardening["network_default_posture"] != launcherbackend.HardeningNetworkDefaultLoopbackOnly {
		t.Fatalf("applied_hardening_posture.network_default_posture = %v, want %q", hardening["network_default_posture"], launcherbackend.HardeningNetworkDefaultLoopbackOnly)
	}
	if hardening["egress_enforcement_posture"] != launcherbackend.HardeningEgressEnforcementHostLevel {
		t.Fatalf("applied_hardening_posture.egress_enforcement_posture = %v, want %q", hardening["egress_enforcement_posture"], launcherbackend.HardeningEgressEnforcementHostLevel)
	}
	if hardening["capabilities_posture"] != launcherbackend.HardeningCapabilitiesDropped {
		t.Fatalf("applied_hardening_posture.capabilities_posture = %v, want %q", hardening["capabilities_posture"], launcherbackend.HardeningCapabilitiesDropped)
	}
	if hardening["effective"] != launcherbackend.HardeningEffectiveDegraded {
		t.Fatalf("applied_hardening_posture.effective = %v, want %q", hardening["effective"], launcherbackend.HardeningEffectiveDegraded)
	}
	reasons, ok := hardening["degraded_reasons"].([]string)
	if !ok {
		t.Fatalf("applied_hardening_posture.degraded_reasons = %T, want []string", hardening["degraded_reasons"])
	}
	assertReasonsContainAll(t, reasons, "rootless_best_effort_only")
}

func assertReasonsContainAll(t *testing.T, reasons []string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !slices.Contains(reasons, want) {
			t.Fatalf("degraded_reasons = %v, want include %q", reasons, want)
		}
	}
}
