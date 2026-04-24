package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const sessionExecutionStartCheckpointIdempotencyPrefix = "session-execution-start-checkpoint-"

func (s *Service) appendSessionExecutionTranscriptCheckpoint(sessionID, role, content string, links artifacts.SessionTranscriptLinksDurableState) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(content) == "" {
		return nil
	}
	session, ok := s.SessionState(sessionID)
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = "system"
	}
	_, err := s.AppendSessionMessage(artifacts.SessionMessageAppendRequest{
		SessionID:      sessionID,
		WorkspaceID:    session.WorkspaceID,
		CreatedByRunID: session.CreatedByRunID,
		Role:           role,
		ContentText:    strings.TrimSpace(content),
		RelatedLinks:   links,
		OccurredAt:     s.currentTimestamp(),
	})
	if err != nil {
		return err
	}
	return s.restoreSessionExecutionSummary(sessionID)
}

func (s *Service) appendSessionExecutionStartCheckpoint(sessionID, triggerID, runID, input string) error {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil
	}
	triggerID = strings.TrimSpace(triggerID)
	if triggerID == "" {
		return fmt.Errorf("trigger id is required")
	}
	session, ok := s.SessionState(sessionID)
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}
	linkRuns := []string{}
	if strings.TrimSpace(runID) != "" {
		linkRuns = []string{runID}
	}
	_, err := s.AppendSessionMessage(artifacts.SessionMessageAppendRequest{
		SessionID:       sessionID,
		WorkspaceID:     session.WorkspaceID,
		CreatedByRunID:  session.CreatedByRunID,
		Role:            "user",
		ContentText:     text,
		RelatedLinks:    artifacts.SessionTranscriptLinksDurableState{RunIDs: uniqueSortedStrings(linkRuns)},
		IdempotencyKey:  sessionExecutionStartCheckpointIdempotencyPrefix + triggerID,
		IdempotencyHash: shaDigestIdentity(sessionID + "\n" + triggerID + "\n" + text),
		OccurredAt:      s.currentTimestamp(),
	})
	if err != nil {
		return err
	}
	return s.restoreSessionExecutionSummary(sessionID)
}

func (s *Service) appendApprovalResolutionExecutionCheckpoint(runID, approvalStatus, approvalID string) error {
	if strings.TrimSpace(runID) == "" {
		return nil
	}
	status := strings.TrimSpace(approvalStatus)
	if status == "" {
		status = "resolved"
	}
	for _, session := range s.SessionStates() {
		if !sessionHasRunLink(session, runID) {
			continue
		}
		text := "approval wait " + status
		if strings.TrimSpace(approvalID) != "" {
			text = text + " (" + strings.TrimSpace(approvalID) + ")"
		}
		if err := s.appendSessionExecutionTranscriptCheckpoint(session.SessionID, "system", text, artifacts.SessionTranscriptLinksDurableState{RunIDs: uniqueSortedStrings([]string{runID}), ApprovalIDs: uniqueSortedStrings([]string{approvalID})}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) appendRunnerCheckpointExecutionCheckpoint(runID, checkpointCode string) error {
	message := ""
	switch strings.TrimSpace(checkpointCode) {
	case "approval_wait_entered":
		message = "approval wait entered"
	case "approval_wait_cleared":
		message = "approval wait cleared"
	case "run_started":
		message = "execution started"
	}
	if message == "" {
		return nil
	}
	for _, session := range s.SessionStates() {
		if !sessionHasRunLink(session, runID) {
			continue
		}
		if err := s.appendSessionExecutionTranscriptCheckpoint(session.SessionID, "system", message, artifacts.SessionTranscriptLinksDurableState{RunIDs: uniqueSortedStrings([]string{runID})}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) appendRunnerResultExecutionCheckpoint(runID, resultCode, lifecycleState string) error {
	resultCode = strings.TrimSpace(resultCode)
	lifecycleState = strings.TrimSpace(lifecycleState)
	if resultCode == "" && lifecycleState == "" {
		return nil
	}
	message := "execution terminal"
	if lifecycleState != "" {
		message = "execution " + lifecycleState
	}
	if resultCode != "" {
		message = message + " (" + resultCode + ")"
	}
	for _, session := range s.SessionStates() {
		if !sessionHasRunLink(session, runID) {
			continue
		}
		if err := s.appendSessionExecutionTranscriptCheckpoint(session.SessionID, "system", message, artifacts.SessionTranscriptLinksDurableState{RunIDs: uniqueSortedStrings([]string{runID})}); err != nil {
			return err
		}
	}
	return nil
}

func sessionHasRunLink(session artifacts.SessionDurableState, runID string) bool {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return false
	}
	if sessionRunLinksContain(session.CreatedByRunID, session.LinkedRunIDs, runID) {
		return true
	}
	for _, exec := range session.TurnExecutions {
		if sessionRunLinksContain(exec.PrimaryRunID, exec.LinkedRunIDs, runID) {
			return true
		}
	}
	return false
}

func sessionRunLinksContain(primary string, linked []string, runID string) bool {
	if strings.TrimSpace(primary) == runID {
		return true
	}
	for _, candidate := range linked {
		if strings.TrimSpace(candidate) == runID {
			return true
		}
	}
	return false
}

func (s *Service) restoreSessionExecutionSummary(sessionID string) error {
	session, ok := s.SessionState(sessionID)
	if !ok {
		return nil
	}
	if len(session.TurnExecutions) == 0 {
		return nil
	}
	latest := session.TurnExecutions[len(session.TurnExecutions)-1]
	_, err := s.UpdateSessionTurnExecution(artifacts.SessionTurnExecutionUpdateRequest{
		SessionID:                            sessionID,
		TurnID:                               latest.TurnID,
		ExecutionState:                       latest.ExecutionState,
		WaitKind:                             latest.WaitKind,
		WaitState:                            latest.WaitState,
		PrimaryRunID:                         latest.PrimaryRunID,
		PendingApprovalID:                    latest.PendingApprovalID,
		LinkedRunIDs:                         append([]string{}, latest.LinkedRunIDs...),
		LinkedApprovalIDs:                    append([]string{}, latest.LinkedApprovalIDs...),
		LinkedArtifactDigests:                append([]string{}, latest.LinkedArtifactDigests...),
		LinkedAuditRecordDigests:             append([]string{}, latest.LinkedAuditRecordDigests...),
		BlockedReasonCode:                    latest.BlockedReasonCode,
		TerminalOutcome:                      latest.TerminalOutcome,
		BoundValidatedProjectSubstrateDigest: latest.BoundValidatedProjectSubstrateDigest,
		OccurredAt:                           s.currentTimestamp(),
	})
	return err
}
