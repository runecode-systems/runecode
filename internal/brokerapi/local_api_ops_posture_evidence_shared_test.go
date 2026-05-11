package brokerapi

import (
	"strings"
	"testing"
	"time"

	"github.com/runecode-systems/runecode/internal/artifacts"
	"github.com/runecode-systems/runecode/internal/launcherbackend"
	"github.com/runecode-systems/runecode/internal/policyengine"
)

func syntheticReceiptOnlyAttestationFacts(runID string, backend string, isolation string) launcherbackend.RuntimeFactsSnapshot {
	bootProfile, measurementProfile, bootByName, measurementDigests := syntheticReceiptOnlyAttestationIdentity(backend)
	receipt := syntheticReceiptOnlyAttestationLaunchReceipt(runID, backend, isolation, bootProfile, measurementProfile, bootByName, measurementDigests)
	return launcherbackend.RuntimeFactsSnapshot{
		LaunchReceipt:    receipt,
		HardeningPosture: syntheticReceiptOnlyAttestationHardeningPosture(),
	}
}

func syntheticReceiptOnlyAttestationLaunchReceipt(runID, backend, isolation, bootProfile, measurementProfile string, bootByName map[string]string, measurementDigests []string) launcherbackend.BackendLaunchReceipt {
	return launcherbackend.BackendLaunchReceipt{
		RunID:                             runID,
		StageID:                           "artifact_flow",
		RoleInstanceID:                    "workspace-1",
		RoleFamily:                        "workspace",
		RoleKind:                          "workspace-edit",
		BackendKind:                       backend,
		IsolationAssuranceLevel:           isolation,
		ProvisioningPosture:               launcherbackend.ProvisioningPostureAttested,
		IsolateID:                         "isolate-synthetic",
		SessionID:                         "session-synthetic",
		SessionNonce:                      "nonce-synthetic-0123456789abcdef",
		LaunchContextDigest:               "sha256:" + strings.Repeat("c", 64),
		HandshakeTranscriptHash:           "sha256:" + strings.Repeat("d", 64),
		IsolateSessionKeyIDValue:          strings.Repeat("e", 64),
		SessionSecurity:                   &launcherbackend.SessionSecurityPosture{MutuallyAuthenticated: true, Encrypted: true, ProofOfPossessionVerified: true, ReplayProtected: true},
		RuntimeImageDescriptorDigest:      "sha256:" + strings.Repeat("f", 64),
		RuntimeImageBootProfile:           bootProfile,
		BootComponentDigestByName:         bootByName,
		BootComponentDigests:              append([]string{}, measurementDigests...),
		AttestationEvidenceSourceKind:     launcherbackend.AttestationSourceKindTrustedRuntime,
		AttestationMeasurementProfile:     measurementProfile,
		AttestationFreshnessMaterial:      []string{"session_nonce"},
		AttestationFreshnessBindingClaims: []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"},
		AttestationEvidenceClaimsDigest:   measurementDigests[0],
		CachePosture:                      syntheticReceiptOnlyAttestationCachePosture(),
	}
}

func syntheticReceiptOnlyAttestationCachePosture() *launcherbackend.BackendCachePosture {
	return &launcherbackend.BackendCachePosture{WarmPoolEnabled: true, BootCacheEnabled: true, ResetOrDestroyBeforeReuse: false, ReusePriorSessionIdentityKeys: true, DigestPinned: true, SignaturePinned: true}
}

func syntheticReceiptOnlyAttestationHardeningPosture() launcherbackend.AppliedHardeningPosture {
	return launcherbackend.AppliedHardeningPosture{
		Requested:                 launcherbackend.HardeningRequestedHardened,
		Effective:                 launcherbackend.HardeningEffectiveHardened,
		ExecutionIdentityPosture:  launcherbackend.HardeningExecutionIdentityUnprivileged,
		FilesystemExposurePosture: launcherbackend.HardeningFilesystemExposureRestricted,
		NetworkExposurePosture:    launcherbackend.HardeningNetworkExposureNone,
		SyscallFilteringPosture:   launcherbackend.HardeningSyscallFilteringSeccomp,
		DeviceSurfacePosture:      launcherbackend.HardeningDeviceSurfaceAllowlist,
	}
}

func syntheticReceiptOnlyAttestationIdentity(backend string) (string, string, map[string]string, []string) {
	bootByName := map[string]string{"kernel": "sha256:" + strings.Repeat("a", 64), "initrd": "sha256:" + strings.Repeat("b", 64)}
	bootProfile := launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1
	measurementProfile := launcherbackend.MeasurementProfileMicroVMBootV1
	if backend == launcherbackend.BackendKindContainer {
		bootByName = map[string]string{"image": "sha256:" + strings.Repeat("a", 64)}
		bootProfile = launcherbackend.BootProfileContainerOCIImageV1
		measurementProfile = launcherbackend.MeasurementProfileContainerImageV1
	}
	measurementDigests, err := launcherbackend.DeriveExpectedMeasurementDigests(measurementProfile, bootProfile, bootByName)
	if err != nil {
		panic(err)
	}
	return bootProfile, measurementProfile, bootByName, measurementDigests
}

func recordBackendPosturePolicyDecisionForRun(t *testing.T, s *Service, runID, manifestHash, actionHash, instanceID string) string {
	t.Helper()
	decision := policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           manifestHash,
		PolicyInputHashes:      []string{"sha256:" + strings.Repeat("2", 64)},
		ActionRequestHash:      actionHash,
		RelevantArtifactHashes: []string{"sha256:" + strings.Repeat("4", 64)},
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details:                map[string]any{"precedence": "approval_profile_moderate", "instance_id": instanceID},
	}
	if err := s.RecordPolicyDecision(runID, "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	refs := s.PolicyDecisionRefsForRun(runID)
	if len(refs) == 0 {
		t.Fatal("PolicyDecisionRefsForRun returned empty refs")
	}
	return refs[0]
}

func recordBackendPostureApprovalForRun(t *testing.T, s *Service, runID, selectorRunID, policyRef, manifestHash, actionHash, requestDigest, decisionDigest, instanceID string) string {
	t.Helper()
	approvalID := "sha256:" + strings.Repeat("a", 64)
	now := time.Now().UTC().Round(0)
	if err := s.RecordApproval(artifacts.ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "consumed",
		WorkspaceID:            workspaceIDForRun(runID),
		InstanceID:             instanceID,
		RunID:                  selectorRunID,
		ActionKind:             policyengine.ActionKindBackendPosture,
		RequestedAt:            now.Add(-2 * time.Minute),
		DecidedAt:              func() *time.Time { t := now.Add(-1 * time.Minute); return &t }(),
		ConsumedAt:             func() *time.Time { t := now; return &t }(),
		ApprovalTriggerCode:    "reduced_assurance_backend",
		ChangesIfApproved:      "Reduced-assurance backend posture change may be applied.",
		ApprovalAssuranceLevel: "reauthenticated",
		PresenceMode:           "hardware_touch",
		PolicyDecisionHash:     policyRef,
		ManifestHash:           manifestHash,
		ActionRequestHash:      actionHash,
		RequestDigest:          requestDigest,
		DecisionDigest:         decisionDigest,
	}); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return approvalID
}

func recordContainerRuntimeFactsForBackendEvidence(t *testing.T, s *Service, runID string) {
	t.Helper()
	if err := s.RecordRuntimeFacts(runID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
		RunID:                   runID,
		StageID:                 "artifact_flow",
		RoleInstanceID:          "workspace-1",
		RoleFamily:              "workspace",
		BackendKind:             launcherbackend.BackendKindContainer,
		IsolationAssuranceLevel: launcherbackend.IsolationAssuranceDegraded,
		ProvisioningPosture:     launcherbackend.ProvisioningPostureAttested,
	}}); err != nil {
		t.Fatalf("RecordRuntimeFacts returned error: %v", err)
	}
}

func backendPostureSelectionEvidenceForState(t *testing.T, state map[string]any) map[string]any {
	t.Helper()
	evidence, ok := state["backend_posture_selection_evidence"].(map[string]any)
	if !ok {
		t.Fatalf("authoritative_state.backend_posture_selection_evidence = %T, want map", state["backend_posture_selection_evidence"])
	}
	return evidence
}

func backendPosturePolicyRefsFromEvidence(t *testing.T, evidence map[string]any) []string {
	t.Helper()
	if refs, ok := evidence["policy_decision_refs"].([]string); ok {
		return refs
	}
	refsAny, ok := evidence["policy_decision_refs"].([]any)
	if !ok {
		t.Fatalf("backend_posture_selection_evidence.policy_decision_refs = %T, want []string", evidence["policy_decision_refs"])
	}
	refs := make([]string, 0, len(refsAny))
	for _, item := range refsAny {
		value, ok := item.(string)
		if !ok {
			t.Fatalf("policy_decision_refs entry = %T, want string", item)
		}
		refs = append(refs, value)
	}
	return refs
}

func backendPostureApprovalEvidenceFromEvidence(t *testing.T, evidence map[string]any) map[string]any {
	t.Helper()
	approvalEvidence, ok := evidence["approval"].(map[string]any)
	if !ok {
		t.Fatalf("backend_posture_selection_evidence.approval = %T, want map", evidence["approval"])
	}
	return approvalEvidence
}

func containsStringInSlice(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func assertBackendPostureApprovalEvidence(t *testing.T, approvalEvidence map[string]any, approvalID, requestDigest, decisionDigest, policyRef string) {
	t.Helper()
	if approvalEvidence["approval_id"] != approvalID {
		t.Fatalf("backend_posture_selection_evidence.approval.approval_id = %v, want %q", approvalEvidence["approval_id"], approvalID)
	}
	if approvalEvidence["approval_request_digest"] != requestDigest {
		t.Fatalf("backend_posture_selection_evidence.approval.approval_request_digest = %v, want %q", approvalEvidence["approval_request_digest"], requestDigest)
	}
	if approvalEvidence["approval_decision_digest"] != decisionDigest {
		t.Fatalf("backend_posture_selection_evidence.approval.approval_decision_digest = %v, want %q", approvalEvidence["approval_decision_digest"], decisionDigest)
	}
	if approvalEvidence["policy_decision_hash"] != policyRef {
		t.Fatalf("backend_posture_selection_evidence.approval.policy_decision_hash = %v, want %q", approvalEvidence["policy_decision_hash"], policyRef)
	}
	if approvalEvidence["status"] != "consumed" {
		t.Fatalf("backend_posture_selection_evidence.approval.status = %v, want consumed", approvalEvidence["status"])
	}
}
