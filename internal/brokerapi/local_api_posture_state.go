package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) currentBackendPostureState() BackendPostureState {
	posture := s.currentInstanceBackendPosture()
	state := BackendPostureState{
		SchemaID:             "runecode.protocol.v0.BackendPostureState",
		SchemaVersion:        "0.1.0",
		InstanceID:           strings.TrimSpace(posture.InstanceID),
		BackendKind:          strings.TrimSpace(posture.BackendKind),
		PreferredBackendKind: strings.TrimSpace(posture.PreferredBackendKind),
		Availability: []BackendPostureAvailability{
			{SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "microvm", Available: true},
			{SchemaID: "runecode.protocol.v0.BackendPostureAvailability", SchemaVersion: "0.1.0", BackendKind: "container", Available: true},
		},
	}
	state.ReducedAssuranceActive = strings.EqualFold(state.BackendKind, "container")
	approval := s.latestPendingBackendPostureApprovalForInstance(state.InstanceID)
	if approval != nil {
		state.PendingApproval = true
		state.PendingApprovalID = approval.ApprovalID
		state.LatestPolicyDecisionHash = approval.PolicyDecisionHash
		state.LatestAppliedEvidenceRef = approval.RequestDigest
	}
	if state.LatestPolicyDecisionHash == "" {
		state.LatestPolicyDecisionHash = s.latestBackendPosturePolicyDecisionHash()
	}
	if state.LatestAppliedEvidenceRef == "" {
		state.LatestAppliedEvidenceRef = s.latestBackendPostureAppliedEvidenceRef(state.InstanceID)
	}
	return state
}

func (s *Service) latestPendingBackendPostureApprovalForInstance(instanceID string) *artifacts.ApprovalRecord {
	for _, rec := range s.ApprovalList() {
		if rec.Status != "pending" || rec.ActionKind != policyengine.ActionKindBackendPosture {
			continue
		}
		if strings.TrimSpace(instanceID) == "" || strings.TrimSpace(rec.InstanceID) == strings.TrimSpace(instanceID) {
			copy := rec
			return &copy
		}
	}
	return nil
}

func (s *Service) latestBackendPosturePolicyDecisionHash() string {
	for _, rec := range s.ApprovalList() {
		if rec.ActionKind != policyengine.ActionKindBackendPosture {
			continue
		}
		if hash := strings.TrimSpace(rec.PolicyDecisionHash); hash != "" {
			return hash
		}
	}
	return ""
}

func (s *Service) latestBackendPostureAppliedEvidenceRef(instanceID string) string {
	events, err := s.ReadAuditEvents()
	if err != nil {
		return ""
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if !matchesBackendPostureAppliedEvent(event, instanceID) {
			continue
		}
		if hash := approvalRequestHashFromAuditEvent(event); hash != "" {
			return hash
		}
	}
	return ""
}

func matchesBackendPostureAppliedEvent(event artifacts.AuditEvent, instanceID string) bool {
	if event.Type != "backend_posture_applied" {
		return false
	}
	if strings.TrimSpace(instanceID) == "" {
		return true
	}
	got, _ := event.Details["instance_id"].(string)
	return strings.TrimSpace(got) == strings.TrimSpace(instanceID)
}

func approvalRequestHashFromAuditEvent(event artifacts.AuditEvent) string {
	hash, _ := event.Details["approval_request_hash"].(string)
	return strings.TrimSpace(hash)
}
