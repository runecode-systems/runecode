package artifacts

func copySessionDurableState(in SessionDurableState) SessionDurableState {
	out := in
	out.LinkedRunIDs = append([]string{}, in.LinkedRunIDs...)
	out.ExecutionTriggers = make([]SessionExecutionTriggerDurableState, 0, len(in.ExecutionTriggers))
	for _, trigger := range in.ExecutionTriggers {
		out.ExecutionTriggers = append(out.ExecutionTriggers, copySessionExecutionTriggerDurableState(trigger))
	}
	out.TurnExecutions = make([]SessionTurnExecutionDurableState, 0, len(in.TurnExecutions))
	for _, execution := range in.TurnExecutions {
		out.TurnExecutions = append(out.TurnExecutions, copySessionTurnExecutionDurableState(execution))
	}
	out.TranscriptTurns = make([]SessionTranscriptTurnDurableState, 0, len(in.TranscriptTurns))
	for _, turn := range in.TranscriptTurns {
		out.TranscriptTurns = append(out.TranscriptTurns, copySessionTurnDurableState(turn))
	}
	if in.IdempotencyByKey != nil {
		out.IdempotencyByKey = make(map[string]SessionIdempotencyRecord, len(in.IdempotencyByKey))
		for key, value := range in.IdempotencyByKey {
			out.IdempotencyByKey[key] = value
		}
	}
	if in.ExecutionTriggerIdempotencyByKey != nil {
		out.ExecutionTriggerIdempotencyByKey = make(map[string]SessionExecutionTriggerIdempotencyRecord, len(in.ExecutionTriggerIdempotencyByKey))
		for key, value := range in.ExecutionTriggerIdempotencyByKey {
			out.ExecutionTriggerIdempotencyByKey[key] = value
		}
	}
	return out
}

func copySessionExecutionTriggerDurableState(in SessionExecutionTriggerDurableState) SessionExecutionTriggerDurableState {
	out := in
	out.CreatedAt = in.CreatedAt.UTC()
	return out
}

func copySessionTurnExecutionDurableState(in SessionTurnExecutionDurableState) SessionTurnExecutionDurableState {
	out := in
	out.LinkedRunIDs = append([]string{}, in.LinkedRunIDs...)
	out.LinkedApprovalIDs = append([]string{}, in.LinkedApprovalIDs...)
	out.LinkedArtifactDigests = append([]string{}, in.LinkedArtifactDigests...)
	out.LinkedAuditRecordDigests = append([]string{}, in.LinkedAuditRecordDigests...)
	out.CreatedAt = in.CreatedAt.UTC()
	out.UpdatedAt = in.UpdatedAt.UTC()
	return out
}

func copySessionTurnDurableState(in SessionTranscriptTurnDurableState) SessionTranscriptTurnDurableState {
	out := in
	if in.CompletedAt != nil {
		completedAt := in.CompletedAt.UTC()
		out.CompletedAt = &completedAt
	}
	out.Messages = make([]SessionTranscriptMessageDurableState, 0, len(in.Messages))
	for _, msg := range in.Messages {
		out.Messages = append(out.Messages, copySessionMessageDurableState(msg))
	}
	return out
}

func copySessionMessageDurableState(in SessionTranscriptMessageDurableState) SessionTranscriptMessageDurableState {
	out := in
	out.RelatedLinks = SessionTranscriptLinksDurableState{
		RunIDs:             append([]string{}, in.RelatedLinks.RunIDs...),
		ApprovalIDs:        append([]string{}, in.RelatedLinks.ApprovalIDs...),
		ArtifactDigests:    append([]string{}, in.RelatedLinks.ArtifactDigests...),
		AuditRecordDigests: append([]string{}, in.RelatedLinks.AuditRecordDigests...),
	}
	return out
}
