package artifacts

func copySessionDurableState(in SessionDurableState) SessionDurableState {
	out := in
	out.LinkedRunIDs = append([]string{}, in.LinkedRunIDs...)
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
