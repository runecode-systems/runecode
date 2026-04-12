package brokerapi

func filterRunWatchSummaries(all []RunSummary, req RunWatchRequest) []RunSummary {
	filtered := make([]RunSummary, 0, len(all))
	for _, run := range all {
		if req.RunID != "" && run.RunID != req.RunID {
			continue
		}
		if req.WorkspaceID != "" && run.WorkspaceID != req.WorkspaceID {
			continue
		}
		if req.LifecycleState != "" && run.LifecycleState != req.LifecycleState {
			continue
		}
		filtered = append(filtered, run)
	}
	return filtered
}

func filterApprovalWatchSummaries(all []ApprovalSummary, req ApprovalWatchRequest) []ApprovalSummary {
	filtered := make([]ApprovalSummary, 0, len(all))
	for _, approval := range all {
		if !matchesApprovalWatchRequest(approval, req) {
			continue
		}
		filtered = append(filtered, approval)
	}
	return filtered
}

func filterSessionWatchSummaries(all []SessionSummary, req SessionWatchRequest) []SessionSummary {
	filtered := make([]SessionSummary, 0, len(all))
	for _, session := range all {
		if !matchesSessionWatchRequest(session, req) {
			continue
		}
		filtered = append(filtered, session)
	}
	return filtered
}

func matchesApprovalWatchRequest(approval ApprovalSummary, req ApprovalWatchRequest) bool {
	if req.ApprovalID != "" && approval.ApprovalID != req.ApprovalID {
		return false
	}
	if req.RunID != "" && approval.BoundScope.RunID != req.RunID {
		return false
	}
	if req.WorkspaceID != "" && approval.BoundScope.WorkspaceID != req.WorkspaceID {
		return false
	}
	if req.Status != "" && approval.Status != req.Status {
		return false
	}
	return true
}

func matchesSessionWatchRequest(session SessionSummary, req SessionWatchRequest) bool {
	if req.SessionID != "" && session.Identity.SessionID != req.SessionID {
		return false
	}
	if req.WorkspaceID != "" && session.Identity.WorkspaceID != req.WorkspaceID {
		return false
	}
	if req.Status != "" && session.Status != req.Status {
		return false
	}
	if req.LastActivityKind != "" && session.LastActivityKind != req.LastActivityKind {
		return false
	}
	return true
}
