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
	approvalsBySession := map[string]map[string]struct{}{}
	for _, approval := range s.listApprovals() {
		sessionID, ok := s.sessionIDForRun(approval.BoundScope.RunID)
		if !ok {
			continue
		}
		if _, exists := approvalsBySession[sessionID]; !exists {
			approvalsBySession[sessionID] = map[string]struct{}{}
		}
		approvalsBySession[sessionID][approval.ApprovalID] = struct{}{}
	}
	return approvalsBySession
}

func (s *Service) recordsBySession() map[string][]artifacts.ArtifactRecord {
	byID := map[string][]artifacts.ArtifactRecord{}
	for _, rec := range s.List() {
		sessionID, ok := s.sessionIDForRun(rec.RunID)
		if !ok {
			continue
		}
		byID[sessionID] = append(byID[sessionID], rec)
	}
	return byID
}

func (s *Service) sessionIDForRun(runID string) (string, bool) {
	if strings.TrimSpace(runID) == "" {
		return "", false
	}
	runtime := s.RuntimeFacts(runID)
	sessionID := strings.TrimSpace(runtime.LaunchReceipt.Normalized().SessionID)
	return sessionID, sessionID != ""
}

func (s *Service) sessionLinkedObjects(sessionID string) (map[string]struct{}, map[string]struct{}, map[string]struct{}) {
	runs := map[string]struct{}{}
	approvals := map[string]struct{}{}
	artifactsByDigest := map[string]struct{}{}
	for _, rec := range s.List() {
		if linkedSessionID, ok := s.sessionIDForRun(rec.RunID); ok && linkedSessionID == sessionID {
			runs[rec.RunID] = struct{}{}
			artifactsByDigest[rec.Reference.Digest] = struct{}{}
		}
	}
	for _, approval := range s.listApprovals() {
		if linkedSessionID, ok := s.sessionIDForRun(approval.BoundScope.RunID); ok && linkedSessionID == sessionID {
			approvals[approval.ApprovalID] = struct{}{}
		}
	}
	return runs, approvals, artifactsByDigest
}
