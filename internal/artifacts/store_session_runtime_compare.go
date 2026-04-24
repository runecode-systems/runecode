package artifacts

import (
	"maps"
	"slices"
)

func sessionDurableStateComparable(state SessionDurableState) sessionDurableStateCompare {
	out := newSessionDurableStateCompare(state)
	out.ExecutionTriggers = sessionExecutionTriggerComparables(state.ExecutionTriggers)
	out.TurnExecutions = sessionTurnExecutionComparables(state.TurnExecutions)
	out.TranscriptTurns = sessionTranscriptTurnComparables(state.TranscriptTurns)
	out.IdempotencyByKey = sessionIdempotencyComparables(state.IdempotencyByKey)
	out.ExecutionTriggerIdempotencyByKey = sessionExecutionTriggerIdempotencyComparables(state.ExecutionTriggerIdempotencyByKey)
	return out
}

func newSessionDurableStateCompare(state SessionDurableState) sessionDurableStateCompare {
	linkedRunIDs := append([]string{}, state.LinkedRunIDs...)
	slices.Sort(linkedRunIDs)
	return sessionDurableStateCompare{
		SessionID:                        state.SessionID,
		WorkspaceID:                      state.WorkspaceID,
		CreatedAtUnixNano:                state.CreatedAt.UnixNano(),
		CreatedByRunID:                   state.CreatedByRunID,
		UpdatedAtUnixNano:                state.UpdatedAt.UnixNano(),
		Status:                           state.Status,
		WorkPosture:                      state.WorkPosture,
		WorkPostureReason:                state.WorkPostureReason,
		LastActivityUnixNano:             state.LastActivityAt.UnixNano(),
		LastActivityKind:                 state.LastActivityKind,
		LastActivityPreview:              state.LastActivityPreview,
		LastInteractionSequence:          state.LastInteractionSequence,
		TurnCount:                        state.TurnCount,
		HasIncompleteTurn:                state.HasIncompleteTurn,
		LinkedRunIDs:                     linkedRunIDs,
		ExecutionTriggers:                make([]sessionExecutionTriggerDurableStateCompare, 0, len(state.ExecutionTriggers)),
		TurnExecutions:                   make([]sessionTurnExecutionDurableStateCompare, 0, len(state.TurnExecutions)),
		TranscriptTurns:                  make([]sessionTranscriptTurnDurableStateCompare, 0, len(state.TranscriptTurns)),
		IdempotencyByKey:                 make(map[string]sessionIdempotencyRecordCompare, len(state.IdempotencyByKey)),
		ExecutionTriggerIdempotencyByKey: make(map[string]sessionExecutionTriggerIdempotencyRecordCompare, len(state.ExecutionTriggerIdempotencyByKey)),
	}
}

func sessionExecutionTriggerComparables(triggers []SessionExecutionTriggerDurableState) []sessionExecutionTriggerDurableStateCompare {
	out := make([]sessionExecutionTriggerDurableStateCompare, 0, len(triggers))
	for _, trigger := range triggers {
		out = append(out, sessionExecutionTriggerComparable(trigger))
	}
	return out
}

func sessionTurnExecutionComparables(executions []SessionTurnExecutionDurableState) []sessionTurnExecutionDurableStateCompare {
	out := make([]sessionTurnExecutionDurableStateCompare, 0, len(executions))
	for _, execution := range executions {
		out = append(out, sessionTurnExecutionComparable(execution))
	}
	return out
}

func sessionTranscriptTurnComparables(turns []SessionTranscriptTurnDurableState) []sessionTranscriptTurnDurableStateCompare {
	out := make([]sessionTranscriptTurnDurableStateCompare, 0, len(turns))
	for _, turn := range turns {
		out = append(out, sessionTranscriptTurnComparable(turn))
	}
	return out
}

func sessionIdempotencyComparables(records map[string]SessionIdempotencyRecord) map[string]sessionIdempotencyRecordCompare {
	out := make(map[string]sessionIdempotencyRecordCompare, len(records))
	for key := range maps.Keys(records) {
		record := records[key]
		out[key] = sessionIdempotencyRecordCompare{RequestHash: record.RequestHash, TurnID: record.TurnID, MessageID: record.MessageID, Seq: record.Seq}
	}
	return out
}

func sessionExecutionTriggerIdempotencyComparables(records map[string]SessionExecutionTriggerIdempotencyRecord) map[string]sessionExecutionTriggerIdempotencyRecordCompare {
	out := make(map[string]sessionExecutionTriggerIdempotencyRecordCompare, len(records))
	for key := range maps.Keys(records) {
		record := records[key]
		out[key] = sessionExecutionTriggerIdempotencyRecordCompare{RequestHash: record.RequestHash, TriggerID: record.TriggerID, Seq: record.Seq}
	}
	return out
}

func sessionTurnExecutionComparable(execution SessionTurnExecutionDurableState) sessionTurnExecutionDurableStateCompare {
	out := sessionTurnExecutionDurableStateCompare{
		TurnID:                               execution.TurnID,
		SessionID:                            execution.SessionID,
		ExecutionIndex:                       execution.ExecutionIndex,
		TriggerID:                            execution.TriggerID,
		TriggerSource:                        execution.TriggerSource,
		RequestedOperation:                   execution.RequestedOperation,
		ExecutionState:                       execution.ExecutionState,
		WaitKind:                             execution.WaitKind,
		WaitState:                            execution.WaitState,
		ApprovalProfile:                      execution.ApprovalProfile,
		AutonomyPosture:                      execution.AutonomyPosture,
		PrimaryRunID:                         execution.PrimaryRunID,
		PendingApprovalID:                    execution.PendingApprovalID,
		LinkedRunIDs:                         sortedStringsCopy(execution.LinkedRunIDs),
		LinkedApprovalIDs:                    sortedStringsCopy(execution.LinkedApprovalIDs),
		LinkedArtifactDigests:                sortedStringsCopy(execution.LinkedArtifactDigests),
		LinkedAuditRecordDigests:             sortedStringsCopy(execution.LinkedAuditRecordDigests),
		BoundValidatedProjectSubstrateDigest: execution.BoundValidatedProjectSubstrateDigest,
		BlockedReasonCode:                    execution.BlockedReasonCode,
		TerminalOutcome:                      execution.TerminalOutcome,
		CreatedAtUnixNano:                    execution.CreatedAt.UnixNano(),
		UpdatedAtUnixNano:                    execution.UpdatedAt.UnixNano(),
	}
	return out
}

func sessionExecutionTriggerComparable(trigger SessionExecutionTriggerDurableState) sessionExecutionTriggerDurableStateCompare {
	return sessionExecutionTriggerDurableStateCompare{
		TriggerID:              trigger.TriggerID,
		SessionID:              trigger.SessionID,
		TriggerIndex:           trigger.TriggerIndex,
		TriggerSource:          trigger.TriggerSource,
		RequestedOperation:     trigger.RequestedOperation,
		UserMessageContentText: trigger.UserMessageContentText,
		CreatedAtUnixNano:      trigger.CreatedAt.UnixNano(),
	}
}

func sessionTranscriptTurnComparable(turn SessionTranscriptTurnDurableState) sessionTranscriptTurnDurableStateCompare {
	completedAt := int64(0)
	if turn.CompletedAt != nil {
		completedAt = turn.CompletedAt.UnixNano()
	}
	out := sessionTranscriptTurnDurableStateCompare{
		TurnID:              turn.TurnID,
		SessionID:           turn.SessionID,
		TurnIndex:           turn.TurnIndex,
		StartedAtUnixNano:   turn.StartedAt.UnixNano(),
		CompletedAtUnixNano: completedAt,
		Status:              turn.Status,
		Messages:            make([]sessionTranscriptMessageDurableStateCompare, 0, len(turn.Messages)),
	}
	for _, message := range turn.Messages {
		out.Messages = append(out.Messages, sessionTranscriptMessageComparable(message))
	}
	return out
}

func sessionTranscriptMessageComparable(message SessionTranscriptMessageDurableState) sessionTranscriptMessageDurableStateCompare {
	return sessionTranscriptMessageDurableStateCompare{
		MessageID:         message.MessageID,
		TurnID:            message.TurnID,
		SessionID:         message.SessionID,
		MessageIndex:      message.MessageIndex,
		Role:              message.Role,
		CreatedAtUnixNano: message.CreatedAt.UnixNano(),
		ContentText:       message.ContentText,
		RelatedLinks: sessionTranscriptLinksDurableStateCompare{
			RunIDs:             sortedStringsCopy(message.RelatedLinks.RunIDs),
			ApprovalIDs:        sortedStringsCopy(message.RelatedLinks.ApprovalIDs),
			ArtifactDigests:    sortedStringsCopy(message.RelatedLinks.ArtifactDigests),
			AuditRecordDigests: sortedStringsCopy(message.RelatedLinks.AuditRecordDigests),
		},
	}
}

func sortedStringsCopy(values []string) []string {
	out := append([]string{}, values...)
	slices.Sort(out)
	return out
}

type sessionDurableStateCompare struct {
	SessionID                        string
	WorkspaceID                      string
	CreatedAtUnixNano                int64
	CreatedByRunID                   string
	UpdatedAtUnixNano                int64
	Status                           string
	WorkPosture                      string
	WorkPostureReason                string
	LastActivityUnixNano             int64
	LastActivityKind                 string
	LastActivityPreview              string
	LastInteractionSequence          int64
	TurnCount                        int
	HasIncompleteTurn                bool
	ExecutionTriggers                []sessionExecutionTriggerDurableStateCompare
	TurnExecutions                   []sessionTurnExecutionDurableStateCompare
	TranscriptTurns                  []sessionTranscriptTurnDurableStateCompare
	IdempotencyByKey                 map[string]sessionIdempotencyRecordCompare
	ExecutionTriggerIdempotencyByKey map[string]sessionExecutionTriggerIdempotencyRecordCompare
	LinkedRunIDs                     []string
}

type sessionExecutionTriggerDurableStateCompare struct {
	TriggerID              string
	SessionID              string
	TriggerIndex           int
	TriggerSource          string
	RequestedOperation     string
	UserMessageContentText string
	CreatedAtUnixNano      int64
}

type sessionTurnExecutionDurableStateCompare struct {
	TurnID                               string
	SessionID                            string
	ExecutionIndex                       int
	TriggerID                            string
	TriggerSource                        string
	RequestedOperation                   string
	ExecutionState                       string
	WaitKind                             string
	WaitState                            string
	ApprovalProfile                      string
	AutonomyPosture                      string
	PrimaryRunID                         string
	PendingApprovalID                    string
	LinkedRunIDs                         []string
	LinkedApprovalIDs                    []string
	LinkedArtifactDigests                []string
	LinkedAuditRecordDigests             []string
	BoundValidatedProjectSubstrateDigest string
	BlockedReasonCode                    string
	TerminalOutcome                      string
	CreatedAtUnixNano                    int64
	UpdatedAtUnixNano                    int64
}

type sessionTranscriptTurnDurableStateCompare struct {
	TurnID              string
	SessionID           string
	TurnIndex           int
	StartedAtUnixNano   int64
	CompletedAtUnixNano int64
	Status              string
	Messages            []sessionTranscriptMessageDurableStateCompare
}

type sessionTranscriptMessageDurableStateCompare struct {
	MessageID         string
	TurnID            string
	SessionID         string
	MessageIndex      int
	Role              string
	CreatedAtUnixNano int64
	ContentText       string
	RelatedLinks      sessionTranscriptLinksDurableStateCompare
}

type sessionTranscriptLinksDurableStateCompare struct {
	RunIDs             []string
	ApprovalIDs        []string
	ArtifactDigests    []string
	AuditRecordDigests []string
}

type sessionIdempotencyRecordCompare struct {
	RequestHash string
	TurnID      string
	MessageID   string
	Seq         int64
}

type sessionExecutionTriggerIdempotencyRecordCompare struct {
	RequestHash string
	TriggerID   string
	Seq         int64
}
