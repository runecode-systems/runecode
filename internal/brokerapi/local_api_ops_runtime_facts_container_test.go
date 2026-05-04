package brokerapi

import (
	"path/filepath"
	"slices"
	"strings"
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

func TestRunDetailRuntimeFactsContainerProjectionSurvivesServiceRestartWithPersistedEvidence(t *testing.T) {
	root := t.TempDir()
	storeRoot := filepath.Join(root, "store")
	ledgerRoot := filepath.Join(root, "ledger")
	svc, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	const runID = "run-container-runtime-restart"
	putRunScopedArtifactForLocalOpsTest(t, svc, runID, "step-1")
	facts := containerRuntimeFactsFixtureWithPostHandshakeEvidence(runID)
	if err := svc.RecordRuntimeFacts(runID, facts); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
	preRestartEvidence := svc.RuntimeEvidence(runID)
	if preRestartEvidence.Attestation == nil || preRestartEvidence.AttestationVerification == nil {
		t.Fatalf("pre-restart runtime evidence attestation/verification missing: %#v", preRestartEvidence)
	}

	reloaded, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig(reload) returned error: %v", err)
	}
	runGet := mustRunGetForRuntimeFactsRestartTest(t, reloaded, runID)
	state := runGet.Run.AuthoritativeState
	if state["backend_kind"] != launcherbackend.BackendKindContainer {
		t.Fatalf("authoritative_state.backend_kind = %v, want %q", state["backend_kind"], launcherbackend.BackendKindContainer)
	}
	if state["attestation_evidence_digest"] != preRestartEvidence.Attestation.EvidenceDigest {
		t.Fatalf("authoritative_state.attestation_evidence_digest = %v, want %q after restart", state["attestation_evidence_digest"], preRestartEvidence.Attestation.EvidenceDigest)
	}
	if state["attestation_verification_digest"] != preRestartEvidence.AttestationVerification.VerificationDigest {
		t.Fatalf("authoritative_state.attestation_verification_digest = %v, want %q after restart", state["attestation_verification_digest"], preRestartEvidence.AttestationVerification.VerificationDigest)
	}
	wantVerificationSucceeded := preRestartEvidence.Attestation != nil && preRestartEvidence.AttestationVerification != nil && preRestartEvidence.AttestationVerification.VerificationDigest != "" && preRestartEvidence.AttestationVerification.VerificationResult == launcherbackend.AttestationVerificationResultValid && preRestartEvidence.AttestationVerification.ReplayVerdict == launcherbackend.AttestationReplayVerdictOriginal
	if state["attestation_verification_succeeded"] != wantVerificationSucceeded {
		t.Fatalf("authoritative_state.attestation_verification_succeeded = %v, want %v from persisted verification", state["attestation_verification_succeeded"], wantVerificationSucceeded)
	}
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
		LaunchReceipt: containerAttestedLaunchReceipt(runID, "workspace-1", "workspace"),
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

func containerAttestedLaunchReceipt(runID, roleInstanceID, roleFamily string) launcherbackend.BackendLaunchReceipt {
	componentDigests := map[string]string{"image": "sha256:" + strings.Repeat("e", 64)}
	return launcherbackend.BackendLaunchReceipt{
		RunID:                             runID,
		StageID:                           "artifact_flow",
		RoleInstanceID:                    roleInstanceID,
		RoleFamily:                        roleFamily,
		BackendKind:                       launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel:           launcherbackend.IsolationAssuranceDegraded,
		ProvisioningPosture:               launcherbackend.ProvisioningPostureAttested,
		IsolateID:                         "isolate-container-1",
		SessionID:                         "session-container-1",
		SessionNonce:                      "nonce-container-0123456789abcdef",
		LaunchContextDigest:               "sha256:" + strings.Repeat("a", 64),
		HandshakeTranscriptHash:           "sha256:" + strings.Repeat("b", 64),
		IsolateSessionKeyIDValue:          strings.Repeat("c", 64),
		SessionSecurity:                   &launcherbackend.SessionSecurityPosture{MutuallyAuthenticated: true, Encrypted: true, ProofOfPossessionVerified: true, ReplayProtected: true},
		RuntimeImageDescriptorDigest:      "sha256:" + strings.Repeat("d", 64),
		RuntimeImageBootProfile:           launcherbackend.BootProfileContainerOCIImageV1,
		BootComponentDigestByName:         componentDigests,
		BootComponentDigests:              []string{"sha256:" + strings.Repeat("e", 64)},
		AuthorityStateDigest:              "sha256:" + strings.Repeat("f", 64),
		AuthorityStateRevision:            1,
		AttestationEvidenceSourceKind:     launcherbackend.AttestationSourceKindTrustedRuntime,
		AttestationMeasurementProfile:     launcherbackend.MeasurementProfileContainerImageV1,
		AttestationFreshnessMaterial:      []string{"session_nonce"},
		AttestationFreshnessBindingClaims: []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"},
		AttestationEvidenceClaimsDigest:   containerRuntimeFactsMeasurementDigest(componentDigests),
	}
}

func containerRuntimeFactsMeasurementDigest(componentDigests map[string]string) string {
	digests, err := launcherbackend.DeriveExpectedMeasurementDigests(launcherbackend.MeasurementProfileContainerImageV1, launcherbackend.BootProfileContainerOCIImageV1, componentDigests)
	if err != nil {
		panic(err)
	}
	return digests[0]
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
			ProvisioningPosture:     launcherbackend.ProvisioningPostureAttested,
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

func containerRuntimeFactsFixtureWithPostHandshakeEvidence(runID string) launcherbackend.RuntimeFactsSnapshot {
	receipt := containerAttestedLaunchReceipt(runID, "workspace-1", "workspace")
	return launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt:                 receipt,
		PostHandshakeAttestationInput: containerRuntimeFactsPostHandshakeAttestationInput(receipt),
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

func containerRuntimeFactsPostHandshakeAttestationInput(receipt launcherbackend.BackendLaunchReceipt) *launcherbackend.PostHandshakeRuntimeAttestationInput {
	return &launcherbackend.PostHandshakeRuntimeAttestationInput{
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
		BootComponentDigestByName:    map[string]string{"image": receipt.BootComponentDigestByName["image"]},
		BootComponentDigests:         append([]string{}, receipt.BootComponentDigests...),
		AttestationSourceKind:        receipt.AttestationEvidenceSourceKind,
		MeasurementProfile:           receipt.AttestationMeasurementProfile,
		FreshnessMaterial:            append([]string{}, receipt.AttestationFreshnessMaterial...),
		FreshnessBindingClaims:       append([]string{}, receipt.AttestationFreshnessBindingClaims...),
		EvidenceClaimsDigest:         receipt.AttestationEvidenceClaimsDigest,
	}
}

func assertContainerSummaryPostureVocabulary(t *testing.T, summary RunSummary) {
	t.Helper()
	if summary.BackendKind != launcherbackend.BackendKindContainer {
		t.Fatalf("summary.backend_kind = %q, want %q", summary.BackendKind, launcherbackend.BackendKindContainer)
	}
	if summary.ProvisioningPosture != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("summary.provisioning_posture = %q, want %q", summary.ProvisioningPosture, launcherbackend.ProvisioningPostureTOFU)
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
	if state["provisioning_posture"] != launcherbackend.ProvisioningPostureTOFU {
		t.Fatalf("authoritative_state.provisioning_posture = %v, want %q", state["provisioning_posture"], launcherbackend.ProvisioningPostureTOFU)
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
