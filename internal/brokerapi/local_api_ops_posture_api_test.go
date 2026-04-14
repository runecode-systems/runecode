package brokerapi

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestBackendPostureGetReturnsTypedState(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	resp, errResp := s.HandleBackendPostureGet(context.Background(), BackendPostureGetRequest{SchemaID: "runecode.protocol.v0.BackendPostureGetRequest", SchemaVersion: "0.1.0", RequestID: "req-posture-get"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleBackendPostureGet error response: %+v", errResp)
	}
	if resp.Posture.InstanceID == "" {
		t.Fatal("posture.instance_id empty")
	}
	if resp.Posture.BackendKind == "" || resp.Posture.PreferredBackendKind == "" {
		t.Fatalf("unexpected posture: %+v", resp.Posture)
	}
	if len(resp.Posture.Availability) < 2 {
		t.Fatalf("availability len=%d, want >=2", len(resp.Posture.Availability))
	}
	assertBackendAvailabilityByPlatform(t, resp.Posture.Availability)
}

func assertBackendAvailabilityByPlatform(t *testing.T, availability []BackendPostureAvailability) {
	t.Helper()
	microvmAvailable := false
	containerAvailable := false
	for _, entry := range availability {
		switch entry.BackendKind {
		case launcherbackend.BackendKindMicroVM:
			microvmAvailable = entry.Available
		case launcherbackend.BackendKindContainer:
			containerAvailable = entry.Available
		}
	}
	if !microvmAvailable {
		t.Fatal("microvm availability should be true")
	}
	wantContainer := runtime.GOOS == "linux"
	if containerAvailable != wantContainer {
		t.Fatalf("container availability=%t, want %t on %s", containerAvailable, wantContainer, runtime.GOOS)
	}
}

func TestBackendPostureChangeRequiresApprovalAndThenAppliesViaSharedApprovalResolve(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedBackendSelectionE2EContext(t, s, "run-posture-e2e")
	changeResp := requireBackendPostureApprovalRequired(t, s, s.currentInstanceBackendPosture().InstanceID)
	pending := requirePendingBackendPostureApproval(t, s, changeResp.Outcome.ApprovalID)
	requireBackendPosturePolicyDecisionRecordedForInstanceSelector(t, s, pending)
	resolveResp := resolveBackendPostureApproval(t, s, pending, "container")
	if resolveResp.ResolutionReasonCode != "approval_consumed" {
		t.Fatalf("resolution_reason_code=%q, want approval_consumed", resolveResp.ResolutionReasonCode)
	}

	postureResp, postureErr := s.HandleBackendPostureGet(context.Background(), BackendPostureGetRequest{SchemaID: "runecode.protocol.v0.BackendPostureGetRequest", SchemaVersion: "0.1.0", RequestID: "req-posture-post-resolve"}, RequestContext{})
	if postureErr != nil {
		t.Fatalf("HandleBackendPostureGet error response: %+v", postureErr)
	}
	if postureResp.Posture.BackendKind != "container" {
		t.Fatalf("post-resolve backend_kind=%q, want container", postureResp.Posture.BackendKind)
	}
}

func requireBackendPostureApprovalRequired(t *testing.T, s *Service, instanceID string) BackendPostureChangeResponse {
	t.Helper()
	changeResp, errResp := s.HandleBackendPostureChange(context.Background(), BackendPostureChangeRequest{
		SchemaID:                     "runecode.protocol.v0.BackendPostureChangeRequest",
		SchemaVersion:                "0.1.0",
		RequestID:                    "req-posture-change-approval",
		TargetInstanceID:             instanceID,
		TargetBackendKind:            "container",
		SelectionMode:                "explicit_selection",
		ChangeKind:                   "select_backend",
		AssuranceChangeKind:          "reduce_assurance",
		OptInKind:                    "exact_action_approval",
		ReducedAssuranceAcknowledged: true,
		Reason:                       "operator_requested_reduced_assurance_backend_opt_in",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleBackendPostureChange error response: %+v", errResp)
	}
	if changeResp.Outcome.Outcome != "approval_required" {
		t.Fatalf("outcome=%q, want approval_required", changeResp.Outcome.Outcome)
	}
	if changeResp.Outcome.ApprovalID == "" {
		t.Fatal("approval_id empty")
	}
	return changeResp
}

func requirePendingBackendPostureApproval(t *testing.T, s *Service, approvalID string) artifacts.ApprovalRecord {
	t.Helper()
	pending, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	if pending.Status != "pending" || pending.ActionKind != policyengine.ActionKindBackendPosture {
		t.Fatalf("unexpected pending approval state: %+v", pending)
	}
	return pending
}

func requireBackendPosturePolicyDecisionRecordedForInstanceSelector(t *testing.T, s *Service, pending artifacts.ApprovalRecord) {
	t.Helper()
	selectorRunID := instanceControlRunIDForTests(pending.InstanceID)
	refs := s.PolicyDecisionRefsForRun(selectorRunID)
	if len(refs) == 0 {
		t.Fatalf("PolicyDecisionRefsForRun(%q) = empty, want backend-posture decision", selectorRunID)
	}
	if pending.PolicyDecisionHash == "" {
		t.Fatalf("pending policy_decision_hash empty for approval %q", pending.ApprovalID)
	}
	for _, ref := range refs {
		if ref == pending.PolicyDecisionHash {
			return
		}
	}
	t.Fatalf("pending policy_decision_hash %q not found in selector refs %v", pending.PolicyDecisionHash, refs)
}

func resolveBackendPostureApproval(t *testing.T, s *Service, pending artifacts.ApprovalRecord, targetBackend string) ApprovalResolveResponse {
	t.Helper()
	resolveResp, resolveErr := s.HandleApprovalResolve(context.Background(), backendPostureResolveRequest(t, s, pending, targetBackend), RequestContext{})
	if resolveErr != nil {
		t.Fatalf("HandleApprovalResolve error response: %+v", resolveErr)
	}
	return resolveResp
}

func backendPostureResolveRequest(t *testing.T, s *Service, pending artifacts.ApprovalRecord, targetBackend string) ApprovalResolveRequest {
	t.Helper()
	stored, ok := s.ApprovalGet(pending.ApprovalID)
	if !ok || stored.RequestEnvelope == nil {
		t.Fatalf("stored pending approval %q missing request envelope", pending.ApprovalID)
	}
	requestEnv, decisionEnv, verifier := signedResolveEnvelopesForStoredPendingRequest(t, *stored.RequestEnvelope, "human", "approve")
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	return ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-posture-approval-resolve",
		ApprovalID:    pending.ApprovalID,
		BoundScope: ApprovalBoundScope{
			SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion:      "0.1.0",
			WorkspaceID:        pending.WorkspaceID,
			InstanceID:         pending.InstanceID,
			RunID:              pending.RunID,
			ActionKind:         pending.ActionKind,
			PolicyDecisionHash: pending.PolicyDecisionHash,
		},
		ResolutionDetails: ApprovalResolveDetails{
			SchemaID:      approvalResolveDetailsSchemaID,
			SchemaVersion: approvalResolveDetailsSchemaVersion,
			BackendPostureSelection: &ApprovalResolveBackendPostureSelectionDetail{
				SchemaID:          approvalResolveBackendSelectionDetailsSchemaID,
				SchemaVersion:     approvalResolveBackendSelectionDetailsSchemaVersion,
				TargetInstanceID:  pending.InstanceID,
				TargetBackendKind: targetBackend,
			},
		},
		SignedApprovalRequest:  requestEnv,
		SignedApprovalDecision: decisionEnv,
	}
}

func TestBackendPostureChangeRejectsAutomaticFallbackAndDoesNotAutoApply(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedBackendSelectionE2EContext(t, s, "run-posture-auto-fallback")
	instanceID := s.currentInstanceBackendPosture().InstanceID

	resp, errResp := s.HandleBackendPostureChange(context.Background(), BackendPostureChangeRequest{
		SchemaID:                     "runecode.protocol.v0.BackendPostureChangeRequest",
		SchemaVersion:                "0.1.0",
		RequestID:                    "req-posture-auto-fallback",
		TargetInstanceID:             instanceID,
		TargetBackendKind:            "container",
		SelectionMode:                "automatic_fallback_attempt",
		ChangeKind:                   "select_backend",
		AssuranceChangeKind:          "reduce_assurance",
		OptInKind:                    "exact_action_approval",
		ReducedAssuranceAcknowledged: true,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleBackendPostureChange error response: %+v", errResp)
	}
	if resp.Outcome.Outcome != "rejected" {
		t.Fatalf("outcome=%q, want rejected", resp.Outcome.Outcome)
	}
	if resp.Outcome.OutcomeReasonCode != "deny_container_automatic_fallback" {
		t.Fatalf("outcome_reason_code=%q, want deny_container_automatic_fallback", resp.Outcome.OutcomeReasonCode)
	}
	if resp.Posture.BackendKind != "microvm" {
		t.Fatalf("backend_kind=%q, want microvm", resp.Posture.BackendKind)
	}
}

func TestBackendPostureApprovalResolvePreventsReplayAcrossInstanceRestart(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	seedBackendSelectionE2EContext(t, s, "run-posture-replay")
	oldInstance := s.currentInstanceBackendPosture().InstanceID
	changeResp := requireBackendPostureApprovalRequired(t, s, oldInstance)
	restartBackendPostureControllerInstanceForTest(s, "launcher-instance-restarted")
	pending := requirePendingBackendPostureApproval(t, s, changeResp.Outcome.ApprovalID)
	resolveReq := staleBackendPostureResolveRequest(t, s, pending, oldInstance, "container")
	_, resolveErr := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if resolveErr == nil {
		t.Fatal("HandleApprovalResolve expected stale instance binding failure")
	}
	if resolveErr.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("error.code=%q, want broker_approval_state_invalid", resolveErr.Error.Code)
	}
	if !strings.Contains(resolveErr.Error.Message, "stale") {
		t.Fatalf("error.message=%q, want stale signal", resolveErr.Error.Message)
	}
}

func restartBackendPostureControllerInstanceForTest(s *Service, instanceID string) {
	if controller, ok := s.instancePostureController.(*localInstanceBackendPostureController); ok {
		controller.mu.Lock()
		controller.current.InstanceID = instanceID
		controller.mu.Unlock()
	}
}

func staleBackendPostureResolveRequest(t *testing.T, s *Service, pending artifacts.ApprovalRecord, instanceID, targetBackend string) ApprovalResolveRequest {
	t.Helper()
	resolveReq := backendPostureResolveRequest(t, s, pending, targetBackend)
	resolveReq.RequestID = "req-posture-replay-resolve"
	resolveReq.BoundScope.InstanceID = instanceID
	resolveReq.ResolutionDetails.BackendPostureSelection.TargetInstanceID = instanceID
	return resolveReq
}
