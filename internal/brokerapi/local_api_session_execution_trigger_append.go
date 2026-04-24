package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
)

func (s *Service) buildSessionExecutionAppendRequest(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState) (artifacts.SessionExecutionTriggerAppendRequest, *ErrorResponse) {
	project, errResp := s.requireSupportedProjectSubstrateForSessionExecution(requestID)
	if errResp != nil {
		return artifacts.SessionExecutionTriggerAppendRequest{}, errResp
	}
	return s.newSessionExecutionAppendRequest(requestID, req, session, project)
}

func (s *Service) newSessionExecutionAppendRequest(requestID string, req SessionExecutionTriggerRequest, session artifacts.SessionDurableState, project projectsubstrate.DiscoveryResult) (artifacts.SessionExecutionTriggerAppendRequest, *ErrorResponse) {
	controls := sessionExecutionTriggerControls(req)
	orchestrationScopeID, dependencyScopeIDs := sessionExecutionScopeForTrigger(session, req)
	idempotencyHash, errResp := s.sessionExecutionTriggerIdempotencyHash(requestID, req, controls)
	if errResp != nil {
		return artifacts.SessionExecutionTriggerAppendRequest{}, errResp
	}
	links := sessionExecutionLinksFromSessionState(session)
	executionState, waitKind, waitState := sessionExecutionInitialState(req.TriggerSource, controls.autonomyPosture)
	return artifacts.SessionExecutionTriggerAppendRequest{
		SessionID:                            req.SessionID,
		WorkspaceID:                          session.WorkspaceID,
		CreatedByRunID:                       session.CreatedByRunID,
		TriggerSource:                        req.TriggerSource,
		RequestedOperation:                   req.RequestedOperation,
		OrchestrationScopeID:                 orchestrationScopeID,
		DependsOnScopeIDs:                    dependencyScopeIDs,
		ApprovalProfile:                      controls.approvalProfile,
		AutonomyPosture:                      controls.autonomyPosture,
		PrimaryRunID:                         initialSessionExecutionPrimaryRunID(session),
		LinkedRunIDs:                         links.runIDs,
		LinkedApprovalIDs:                    links.approvalIDs,
		LinkedArtifactDigests:                links.artifactDigests,
		LinkedAuditRecordDigests:             links.auditRecordDigests,
		BoundValidatedProjectSubstrateDigest: sessionExecutionBoundDigest(project),
		ExecutionState:                       executionState,
		WaitKind:                             waitKind,
		WaitState:                            waitState,
		UserMessageContentText:               strings.TrimSpace(req.UserMessageContentText),
		IdempotencyKey:                       strings.TrimSpace(req.IdempotencyKey),
		IdempotencyHash:                      idempotencyHash,
		OccurredAt:                           s.currentTimestamp(),
	}, nil
}

func (s *Service) sessionExecutionTriggerIdempotencyHash(requestID string, req SessionExecutionTriggerRequest, controls sessionExecutionTriggerControlValues) (string, *ErrorResponse) {
	idempotencyHash, err := artifacts.SessionExecutionTriggerIdempotencyHash(req.SessionID, req.TriggerSource, req.RequestedOperation, controls.approvalProfile, controls.autonomyPosture, req.UserMessageContentText)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return "", &errOut
	}
	return idempotencyHash, nil
}

func initialSessionExecutionPrimaryRunID(session artifacts.SessionDurableState) string {
	if len(session.TurnExecutions) > 0 {
		return ""
	}
	return strings.TrimSpace(session.CreatedByRunID)
}
