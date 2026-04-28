package artifacts

import (
	"fmt"
	"strings"
	"time"
)

type SessionExecutionTriggerAppendRequest struct {
	SessionID                            string
	AuthoritativeRepositoryRoot          string
	WorkspaceID                          string
	CreatedByRunID                       string
	TriggerSource                        string
	RequestedOperation                   string
	OrchestrationScopeID                 string
	DependsOnScopeIDs                    []string
	ApprovalProfile                      string
	AutonomyPosture                      string
	PrimaryRunID                         string
	PendingApprovalID                    string
	LinkedRunIDs                         []string
	LinkedApprovalIDs                    []string
	LinkedArtifactDigests                []string
	LinkedAuditRecordDigests             []string
	BoundValidatedProjectSubstrateDigest string
	WaitKind                             string
	WaitState                            string
	BlockedReasonCode                    string
	TerminalOutcome                      string
	ExecutionState                       string
	UserMessageContentText               string
	IdempotencyKey                       string
	IdempotencyHash                      string
	OccurredAt                           time.Time
}

type SessionExecutionTriggerAppendResult struct {
	Created         bool
	Trigger         SessionExecutionTriggerDurableState
	TurnExecution   SessionTurnExecutionDurableState
	Seq             int64
	IdempotencyHash string
}

type SessionTurnExecutionUpdateRequest struct {
	SessionID                            string
	TurnID                               string
	ExecutionState                       string
	WaitKind                             string
	WaitState                            string
	OrchestrationScopeID                 string
	DependsOnScopeIDs                    []string
	PrimaryRunID                         string
	PendingApprovalID                    string
	LinkedRunIDs                         []string
	LinkedApprovalIDs                    []string
	LinkedArtifactDigests                []string
	LinkedAuditRecordDigests             []string
	BlockedReasonCode                    string
	TerminalOutcome                      string
	BoundValidatedProjectSubstrateDigest string
	OccurredAt                           time.Time
}

func normalizeSessionExecutionTriggerAppendRequest(req SessionExecutionTriggerAppendRequest) (SessionExecutionTriggerAppendRequest, error) {
	normalized := req
	if err := normalizeSessionExecutionTriggerIdentityFields(&normalized, req); err != nil {
		return SessionExecutionTriggerAppendRequest{}, err
	}
	normalized.AuthoritativeRepositoryRoot = strings.TrimSpace(req.AuthoritativeRepositoryRoot)
	normalized.OrchestrationScopeID = strings.TrimSpace(req.OrchestrationScopeID)
	if req.DependsOnScopeIDs != nil {
		normalized.DependsOnScopeIDs = uniqueSortedStrings(req.DependsOnScopeIDs)
	}
	normalizeSessionExecutionTriggerControlFields(&normalized, req)
	normalized.BoundValidatedProjectSubstrateDigest = strings.TrimSpace(req.BoundValidatedProjectSubstrateDigest)
	normalized.WaitKind = strings.TrimSpace(req.WaitKind)
	normalized.WaitState = strings.TrimSpace(req.WaitState)
	normalized.PrimaryRunID = strings.TrimSpace(req.PrimaryRunID)
	normalized.PendingApprovalID = strings.TrimSpace(req.PendingApprovalID)
	normalized.LinkedRunIDs = uniqueSortedStrings(req.LinkedRunIDs)
	normalized.LinkedApprovalIDs = uniqueSortedStrings(req.LinkedApprovalIDs)
	normalized.LinkedArtifactDigests = uniqueSortedStrings(req.LinkedArtifactDigests)
	normalized.LinkedAuditRecordDigests = uniqueSortedStrings(req.LinkedAuditRecordDigests)
	normalized.BlockedReasonCode = strings.TrimSpace(req.BlockedReasonCode)
	normalized.TerminalOutcome = strings.TrimSpace(req.TerminalOutcome)
	normalized.ExecutionState = defaultSessionExecutionState(req.ExecutionState)
	normalized.UserMessageContentText = strings.TrimSpace(req.UserMessageContentText)
	normalized.IdempotencyKey, normalized.IdempotencyHash = normalizeSessionExecutionTriggerIdempotency(req)
	if normalized.IdempotencyKey != "" && normalized.IdempotencyHash == "" {
		return SessionExecutionTriggerAppendRequest{}, fmt.Errorf("idempotency hash is required when idempotency key is set")
	}
	normalized.OccurredAt = normalizedOccurredAt(req.OccurredAt)
	return normalized, nil
}

func normalizeSessionExecutionTriggerIdentityFields(normalized *SessionExecutionTriggerAppendRequest, req SessionExecutionTriggerAppendRequest) error {
	normalized.SessionID = strings.TrimSpace(req.SessionID)
	if normalized.SessionID == "" {
		return fmt.Errorf("session id is required")
	}
	normalized.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	if normalized.WorkspaceID == "" {
		normalized.WorkspaceID = "workspace-local"
	}
	normalized.CreatedByRunID = strings.TrimSpace(req.CreatedByRunID)
	normalized.TriggerSource = strings.TrimSpace(req.TriggerSource)
	if normalized.TriggerSource == "" {
		return fmt.Errorf("trigger source is required")
	}
	normalized.RequestedOperation = strings.TrimSpace(req.RequestedOperation)
	if normalized.RequestedOperation == "" {
		return fmt.Errorf("requested operation is required")
	}
	return nil
}

func normalizeSessionExecutionTriggerControlFields(normalized *SessionExecutionTriggerAppendRequest, req SessionExecutionTriggerAppendRequest) {
	normalized.ApprovalProfile = defaultSessionApprovalProfile(req.ApprovalProfile)
	normalized.AutonomyPosture = defaultSessionAutonomyPosture(req.AutonomyPosture)
	normalized.PrimaryRunID = strings.TrimSpace(req.PrimaryRunID)
	normalized.PendingApprovalID = strings.TrimSpace(req.PendingApprovalID)
	normalized.LinkedRunIDs, normalized.LinkedApprovalIDs, normalized.LinkedArtifactDigests, normalized.LinkedAuditRecordDigests = normalizeSessionExecutionTriggerLinks(req)
}

func defaultSessionApprovalProfile(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "moderate"
	}
	return trimmed
}

func defaultSessionAutonomyPosture(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "operator_guided"
	}
	return trimmed
}

func defaultSessionExecutionState(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "queued"
	}
	return trimmed
}

func normalizeSessionExecutionTriggerLinks(req SessionExecutionTriggerAppendRequest) ([]string, []string, []string, []string) {
	return uniqueSortedStrings(req.LinkedRunIDs), uniqueSortedStrings(req.LinkedApprovalIDs), uniqueSortedStrings(req.LinkedArtifactDigests), uniqueSortedStrings(req.LinkedAuditRecordDigests)
}

func normalizeSessionExecutionTriggerIdempotency(req SessionExecutionTriggerAppendRequest) (string, string) {
	return strings.TrimSpace(req.IdempotencyKey), strings.TrimSpace(req.IdempotencyHash)
}

func normalizeSessionTurnExecutionUpdateRequest(req SessionTurnExecutionUpdateRequest) (SessionTurnExecutionUpdateRequest, error) {
	normalized := req
	normalized.SessionID = strings.TrimSpace(req.SessionID)
	if normalized.SessionID == "" {
		return SessionTurnExecutionUpdateRequest{}, fmt.Errorf("session id is required")
	}
	normalized.TurnID = strings.TrimSpace(req.TurnID)
	if normalized.TurnID == "" {
		return SessionTurnExecutionUpdateRequest{}, fmt.Errorf("turn id is required")
	}
	normalized.ExecutionState = strings.TrimSpace(req.ExecutionState)
	if normalized.ExecutionState == "" {
		return SessionTurnExecutionUpdateRequest{}, fmt.Errorf("execution state is required")
	}
	normalized.WaitKind = strings.TrimSpace(req.WaitKind)
	normalized.WaitState = strings.TrimSpace(req.WaitState)
	normalized.OrchestrationScopeID = strings.TrimSpace(req.OrchestrationScopeID)
	if req.DependsOnScopeIDs != nil {
		normalized.DependsOnScopeIDs = uniqueSortedStrings(req.DependsOnScopeIDs)
	}
	normalized.PrimaryRunID = strings.TrimSpace(req.PrimaryRunID)
	normalized.PendingApprovalID = strings.TrimSpace(req.PendingApprovalID)
	normalized.LinkedRunIDs = uniqueSortedStrings(req.LinkedRunIDs)
	normalized.LinkedApprovalIDs = uniqueSortedStrings(req.LinkedApprovalIDs)
	normalized.LinkedArtifactDigests = uniqueSortedStrings(req.LinkedArtifactDigests)
	normalized.LinkedAuditRecordDigests = uniqueSortedStrings(req.LinkedAuditRecordDigests)
	normalized.BlockedReasonCode = strings.TrimSpace(req.BlockedReasonCode)
	normalized.TerminalOutcome = strings.TrimSpace(req.TerminalOutcome)
	normalized.BoundValidatedProjectSubstrateDigest = strings.TrimSpace(req.BoundValidatedProjectSubstrateDigest)
	normalized.OccurredAt = normalizedOccurredAt(req.OccurredAt)
	return normalized, nil
}

func normalizedOccurredAt(occurredAt time.Time) time.Time {
	if occurredAt.IsZero() {
		return time.Now().UTC()
	}
	return occurredAt.UTC()
}

func loadSessionForExecutionTriggerAppend(states map[string]SessionDurableState, req SessionExecutionTriggerAppendRequest) SessionDurableState {
	session := states[req.SessionID]
	if session.SessionID == "" {
		return newSessionStateFromExecutionTriggerAppendRequest(req)
	}
	return session
}

func newSessionStateFromExecutionTriggerAppendRequest(req SessionExecutionTriggerAppendRequest) SessionDurableState {
	return SessionDurableState{
		SessionID:                        req.SessionID,
		WorkspaceID:                      req.WorkspaceID,
		CreatedAt:                        req.OccurredAt,
		CreatedByRunID:                   req.CreatedByRunID,
		UpdatedAt:                        req.OccurredAt,
		Status:                           "active",
		WorkPosture:                      "running",
		LastActivityAt:                   req.OccurredAt,
		LastActivityKind:                 "execution_trigger_submitted",
		LastInteractionSequence:          0,
		HasIncompleteTurn:                false,
		IdempotencyByKey:                 map[string]SessionIdempotencyRecord{},
		ExecutionTriggerIdempotencyByKey: map[string]SessionExecutionTriggerIdempotencyRecord{},
		TurnExecutions:                   []SessionTurnExecutionDurableState{},
	}
}
