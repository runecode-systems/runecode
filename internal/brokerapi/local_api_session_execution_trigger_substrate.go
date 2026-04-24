package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) requireCurrentSessionExecutionDigest(requestID string, session artifacts.SessionDurableState, target artifacts.SessionTurnExecutionDurableState) (string, *ErrorResponse) {
	project, errResp := s.requireSupportedProjectSubstrateForSessionExecution(requestID)
	if errResp != nil {
		reason := firstReasonCodeFromMessage(errResp.Error.Message, sessionExecutionBlockedReasonProjectSubstratePosture)
		_ = s.markTurnExecutionProjectBlocked(session.SessionID, target, reason, s.currentTimestamp())
		return "", errResp
	}
	currentDigest := validatedProjectSubstrateDigest(project)
	boundDigest := strings.TrimSpace(target.BoundValidatedProjectSubstrateDigest)
	if boundDigest != "" && !strings.EqualFold(boundDigest, currentDigest) {
		_ = s.markTurnExecutionProjectBlocked(session.SessionID, target, sessionExecutionBlockedReasonProjectSubstrateDrift, s.currentTimestamp())
		errOut := s.makeError(requestID, "broker_session_execution_project_context_drift", "policy", false, "validated project substrate digest drift blocks session execution continuation")
		return "", &errOut
	}
	return currentDigest, nil
}

func validatedProjectSubstrateDigest(project projectsubstrate.DiscoveryResult) string {
	digest := strings.TrimSpace(project.Snapshot.ValidatedSnapshotDigest)
	if digest == "" {
		digest = strings.TrimSpace(project.Snapshot.ProjectContextIdentityDigest)
	}
	return digest
}

func projectBlockedReason(project projectsubstrate.DiscoveryResult) string {
	if len(project.Compatibility.BlockedReasonCodes) > 0 {
		reason := strings.TrimSpace(project.Compatibility.BlockedReasonCodes[0])
		if reason != "" {
			return reason
		}
	}
	posture := strings.TrimSpace(project.Compatibility.Posture)
	if posture == "" {
		return sessionExecutionBlockedReasonProjectSubstratePosture
	}
	if strings.HasPrefix(posture, "project_substrate_") {
		return posture
	}
	return "project_substrate_" + strings.ReplaceAll(posture, " ", "_")
}

func firstReasonCodeFromMessage(message string, fallback string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return fallback
	}
	idx := strings.LastIndex(trimmed, ":")
	if idx < 0 || idx == len(trimmed)-1 {
		return fallback
	}
	reason := strings.TrimSpace(trimmed[idx+1:])
	if reason == "" {
		return fallback
	}
	return strings.ReplaceAll(reason, ",", "_")
}

func (s *Service) requireSupportedProjectSubstrateForSessionExecution(requestID string) (projectsubstrate.DiscoveryResult, *ErrorResponse) {
	project, err := s.discoverProjectSubstrate()
	if err != nil {
		errOut := s.makeError(requestID, "project_substrate_operation_blocked", "policy", false, "project substrate discovery failed for execution")
		return projectsubstrate.DiscoveryResult{}, &errOut
	}
	if !project.Compatibility.NormalOperationAllowed {
		errOut := s.makeError(requestID, "project_substrate_operation_blocked", "policy", false, "project substrate posture blocks session execution: "+projectBlockedReason(project))
		return projectsubstrate.DiscoveryResult{}, &errOut
	}
	if strings.TrimSpace(project.Snapshot.ValidatedSnapshotDigest) == "" {
		errOut := s.makeError(requestID, "project_substrate_operation_blocked", "policy", false, "validated project substrate snapshot digest missing")
		return projectsubstrate.DiscoveryResult{}, &errOut
	}
	return project, nil
}
