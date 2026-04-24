package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

type sessionExecutionTriggerControlValues struct {
	approvalProfile string
	autonomyPosture string
}

func sessionExecutionTriggerControls(req SessionExecutionTriggerRequest) sessionExecutionTriggerControlValues {
	return sessionExecutionTriggerControlValues{
		approvalProfile: normalizeSessionTriggerApprovalProfile(req.ApprovalProfile),
		autonomyPosture: normalizeSessionTriggerAutonomyPosture(req.AutonomyPosture),
	}
}

func sessionExecutionBoundDigest(project projectsubstrate.DiscoveryResult) string {
	boundDigest := strings.TrimSpace(project.Snapshot.ValidatedSnapshotDigest)
	if boundDigest == "" {
		boundDigest = strings.TrimSpace(project.Snapshot.ProjectContextIdentityDigest)
	}
	return boundDigest
}

func sessionExecutionInitialState(triggerSource, autonomyPosture string) (string, string, string) {
	if triggerSource != "autonomous_background" {
		return "running", "", ""
	}
	if autonomyPosture == "operator_guided" {
		return "waiting", "operator_input", "waiting_operator_input"
	}
	return "running", "", ""
}

func sessionExecutionScopeForTrigger(session artifacts.SessionDurableState, req SessionExecutionTriggerRequest) (string, []string) {
	if strings.TrimSpace(req.RequestedOperation) == "continue" {
		if target, ok := currentOrResumableTurnExecution(session.TurnExecutions, strings.TrimSpace(req.TurnID)); ok {
			scopeID := strings.TrimSpace(target.OrchestrationScopeID)
			if scopeID == "" {
				scopeID = sessionExecutionScopeIDFromTurn(target.TurnID, target.ExecutionIndex)
			}
			return scopeID, append([]string{}, target.DependsOnScopeIDs...)
		}
	}
	nextIndex := len(session.ExecutionTriggers) + 1
	scopeID := fmt.Sprintf("%s.scope.%06d", strings.TrimSpace(session.SessionID), nextIndex)
	return scopeID, []string{}
}

func sessionExecutionScopeIDFromTurn(turnID string, executionIndex int) string {
	trimmedTurnID := strings.TrimSpace(turnID)
	if trimmedTurnID != "" {
		return strings.Replace(trimmedTurnID, ".exec.", ".scope.", 1)
	}
	return fmt.Sprintf("scope.%06d", executionIndex)
}
