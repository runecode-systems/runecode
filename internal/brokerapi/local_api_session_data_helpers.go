package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) auditRecordDigestsBySession() (map[string]map[string]struct{}, error) {
	events, err := s.ReadAuditEvents()
	if err != nil {
		return nil, err
	}
	bySession := map[string]map[string]struct{}{}
	for _, event := range events {
		details := event.Details
		if details == nil {
			continue
		}
		sessionID, ok := detailString(details, "session_id")
		if !ok {
			continue
		}
		recordDigest, ok := detailString(details, "record_digest")
		if !ok {
			continue
		}
		if _, exists := bySession[sessionID]; !exists {
			bySession[sessionID] = map[string]struct{}{}
		}
		bySession[sessionID][recordDigest] = struct{}{}
	}
	return bySession, nil
}

func detailString(details map[string]interface{}, key string) (string, bool) {
	raw, ok := details[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func (s *Service) approvalIDsBySession() map[string]map[string]struct{} {
	states := s.store.SessionDurableStates()
	runToSession := runToSessionIndex(states)
	approvalsBySession := map[string]map[string]struct{}{}
	for _, approval := range s.listApprovals() {
		sessionID, ok := s.sessionIDForRun(approval.BoundScope.RunID, runToSession)
		if !ok {
			continue
		}
		appendSessionLink(approvalsBySession, sessionID, approval.ApprovalID)
	}
	return approvalsBySession
}

func runToSessionIndex(states []artifacts.SessionDurableState) map[string]string {
	index := map[string]string{}
	for _, state := range states {
		sessionID := strings.TrimSpace(state.SessionID)
		if sessionID == "" {
			continue
		}
		for _, runID := range state.LinkedRunIDs {
			trimmed := strings.TrimSpace(runID)
			if trimmed == "" {
				continue
			}
			index[trimmed] = sessionID
		}
		if trimmed := strings.TrimSpace(state.CreatedByRunID); trimmed != "" {
			index[trimmed] = sessionID
		}
	}
	return index
}

func appendSessionLink(bySession map[string]map[string]struct{}, sessionID, value string) {
	if _, exists := bySession[sessionID]; !exists {
		bySession[sessionID] = map[string]struct{}{}
	}
	bySession[sessionID][value] = struct{}{}
}

func (s *Service) sessionArtifactDigestsBySession(runToSession map[string]string) map[string]map[string]struct{} {
	bySession := map[string]map[string]struct{}{}
	for _, rec := range s.List() {
		sessionID, ok := s.sessionIDForRun(rec.RunID, runToSession)
		if !ok {
			continue
		}
		appendSessionLink(bySession, sessionID, rec.Reference.Digest)
	}
	return bySession
}

func (s *Service) sessionRunsBySession(states []artifacts.SessionDurableState, runToSession map[string]string) map[string]map[string]struct{} {
	bySession := map[string]map[string]struct{}{}
	for _, state := range states {
		seedSessionRunIndex(bySession, runToSession, state)
	}
	for _, rec := range s.List() {
		sessionID, ok := s.sessionIDForRun(rec.RunID, runToSession)
		if !ok {
			continue
		}
		appendSessionLink(bySession, sessionID, rec.RunID)
	}
	return bySession
}

func seedSessionRunIndex(bySession map[string]map[string]struct{}, runToSession map[string]string, state artifacts.SessionDurableState) {
	sessionID := strings.TrimSpace(state.SessionID)
	if sessionID == "" {
		return
	}
	for _, runID := range state.LinkedRunIDs {
		appendRunSessionLink(bySession, runToSession, sessionID, runID)
	}
	appendRunSessionLink(bySession, runToSession, sessionID, state.CreatedByRunID)
}

func appendRunSessionLink(bySession map[string]map[string]struct{}, runToSession map[string]string, sessionID, runID string) {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return
	}
	runToSession[trimmed] = sessionID
	appendSessionLink(bySession, sessionID, trimmed)
}

func (s *Service) sessionIDForRun(runID string, runToSession map[string]string) (string, bool) {
	if strings.TrimSpace(runID) == "" {
		return "", false
	}
	if runToSession != nil {
		if sessionID, ok := runToSession[runID]; ok {
			return sessionID, true
		}
	}
	runtime := s.RuntimeFacts(runID)
	sessionID := strings.TrimSpace(runtime.LaunchReceipt.Normalized().SessionID)
	if runToSession != nil && sessionID != "" {
		runToSession[runID] = sessionID
	}
	return sessionID, sessionID != ""
}

func (s *Service) sessionLinkIndexes(states []artifacts.SessionDurableState) (map[string]map[string]struct{}, map[string]map[string]struct{}, map[string]map[string]struct{}) {
	runToSession := runToSessionIndex(states)
	runsBySession := s.sessionRunsBySession(states, runToSession)
	approvalsBySession := map[string]map[string]struct{}{}
	for _, approval := range s.listApprovals() {
		sessionID, ok := s.sessionIDForRun(approval.BoundScope.RunID, runToSession)
		if !ok {
			continue
		}
		appendSessionLink(approvalsBySession, sessionID, approval.ApprovalID)
	}
	artifactsBySession := s.sessionArtifactDigestsBySession(runToSession)
	return runsBySession, approvalsBySession, artifactsBySession
}
