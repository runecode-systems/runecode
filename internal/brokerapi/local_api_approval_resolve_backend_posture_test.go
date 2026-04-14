package brokerapi

import (
	"context"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func TestApprovalResolveBackendPostureFailsClosedOnInstanceMismatch(t *testing.T) {
	s, requestEnv, decisionEnv := setupServiceWithBackendPostureApprovalFixture(t)
	approvalID := approvalIDForBrokerTest(t, requestEnv)
	policyDecisionHash := policyDecisionHashForStoredApproval(t, s, approvalID)
	resolveReq := ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-backend-posture-instance-mismatch",
		ApprovalID:    approvalID,
		BoundScope: ApprovalBoundScope{
			SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion:      "0.1.0",
			WorkspaceID:        workspaceIDForRun("run-backend"),
			InstanceID:         "launcher-instance-stale",
			RunID:              "run-backend",
			ActionKind:         policyengine.ActionKindBackendPosture,
			PolicyDecisionHash: policyDecisionHash,
		},
		ResolutionDetails: ApprovalResolveDetails{
			SchemaID:      approvalResolveDetailsSchemaID,
			SchemaVersion: approvalResolveDetailsSchemaVersion,
			BackendPostureSelection: &ApprovalResolveBackendPostureSelectionDetail{
				SchemaID:          approvalResolveBackendSelectionDetailsSchemaID,
				SchemaVersion:     approvalResolveBackendSelectionDetailsSchemaVersion,
				TargetInstanceID:  "launcher-instance-stale",
				TargetBackendKind: "container",
			},
		},
		SignedApprovalRequest:  *requestEnv,
		SignedApprovalDecision: *decisionEnv,
	}
	_, errResp := s.HandleApprovalResolve(context.Background(), resolveReq, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_approval_state_invalid" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
	if !strings.Contains(errResp.Error.Message, "instance_id") {
		t.Fatalf("error message = %q, want instance binding mismatch signal", errResp.Error.Message)
	}
}
