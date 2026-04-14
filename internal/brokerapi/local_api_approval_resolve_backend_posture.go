package brokerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) validateBackendPostureBinding(current approvalRecord, req ApprovalResolveRequest) error {
	if current.RequestEnvelope == nil {
		return fmt.Errorf("backend posture approval has no stored signed request")
	}
	targetInstanceID, targetBackendKind, err := backendPostureTargetFromRequestEnvelope(*current.RequestEnvelope)
	if err != nil {
		return err
	}
	resolvedDetails := req.normalizedResolutionDetails()
	if resolvedDetails.BackendPostureSelection == nil {
		return fmt.Errorf("resolution_details.backend_posture_selection is required for backend posture approvals")
	}
	if err := validateResolvedBackendSelection(resolvedDetails.BackendPostureSelection, targetInstanceID, targetBackendKind); err != nil {
		return err
	}
	if strings.TrimSpace(current.Summary.BoundScope.InstanceID) != "" && strings.TrimSpace(current.Summary.BoundScope.InstanceID) != targetInstanceID {
		return fmt.Errorf("approval bound scope instance_id does not match signed approval request target_instance_id")
	}
	if strings.TrimSpace(req.BoundScope.InstanceID) != "" && strings.TrimSpace(req.BoundScope.InstanceID) != targetInstanceID {
		return fmt.Errorf("request bound_scope.instance_id does not match signed approval request target_instance_id")
	}
	return nil
}

func (s *Service) applyResolvedBackendPosture(current approvalRecord, req ApprovalResolveRequest) error {
	resolvedDetails := req.normalizedResolutionDetails()
	selection := resolvedDetails.BackendPostureSelection
	if selection == nil {
		return fmt.Errorf("resolution_details.backend_posture_selection is required")
	}
	targetBackendKind := normalizeBackendKindForResolve(selection.TargetBackendKind)
	if targetBackendKind == "" {
		return fmt.Errorf("resolution_details.backend_posture_selection.target_backend_kind is invalid")
	}
	targetInstanceID := strings.TrimSpace(selection.TargetInstanceID)
	if targetInstanceID == "" {
		return fmt.Errorf("resolution_details.backend_posture_selection.target_instance_id is required")
	}
	posture := s.currentInstanceBackendPosture()
	if strings.TrimSpace(posture.InstanceID) == "" {
		return fmt.Errorf("launcher instance identity unavailable")
	}
	if posture.InstanceID != targetInstanceID {
		return fmt.Errorf("backend posture approval target instance is stale")
	}
	if err := s.applyInstanceBackendPosture(context.Background(), targetInstanceID, targetBackendKind); err != nil {
		return err
	}
	if err := s.emitBackendPostureAppliedAuditEvent(targetInstanceID, targetBackendKind, current); err != nil {
		return err
	}
	return nil
}

func normalizeBackendKindForResolve(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case launcherbackend.BackendKindMicroVM:
		return launcherbackend.BackendKindMicroVM
	case launcherbackend.BackendKindContainer:
		return launcherbackend.BackendKindContainer
	default:
		return ""
	}
}

func backendPostureTargetFromRequestEnvelope(requestEnv trustpolicy.SignedObjectEnvelope) (string, string, error) {
	requestPayload, err := decodeApprovalRequestPayload(requestEnv)
	if err != nil {
		return "", "", err
	}
	details, _ := requestPayload["details"].(map[string]any)
	if len(details) == 0 {
		return "", "", fmt.Errorf("backend posture approval request missing details payload")
	}
	targetInstanceID := strings.TrimSpace(stringFieldOrEmpty(details, "target_instance_id"))
	if targetInstanceID == "" {
		return "", "", fmt.Errorf("backend posture approval request missing details.target_instance_id")
	}
	targetBackendKind := normalizeBackendKindForResolve(strings.TrimSpace(stringFieldOrEmpty(details, "target_backend_kind")))
	if targetBackendKind == "" {
		return "", "", fmt.Errorf("backend posture approval request missing details.target_backend_kind")
	}
	return targetInstanceID, targetBackendKind, nil
}

func validateResolvedBackendSelection(selection *ApprovalResolveBackendPostureSelectionDetail, targetInstanceID, targetBackendKind string) error {
	if strings.TrimSpace(selection.TargetInstanceID) != targetInstanceID {
		return fmt.Errorf("resolution_details.backend_posture_selection.target_instance_id does not match signed approval request")
	}
	if normalizeBackendKindForResolve(selection.TargetBackendKind) != targetBackendKind {
		return fmt.Errorf("resolution_details.backend_posture_selection.target_backend_kind does not match signed approval request")
	}
	return nil
}

func (s *Service) emitBackendPostureAppliedAuditEvent(targetInstanceID, targetBackendKind string, current approvalRecord) error {
	details := map[string]interface{}{
		"instance_id":            targetInstanceID,
		"target_backend_kind":    targetBackendKind,
		"action_request_hash":    current.ActionRequestHash,
		"policy_decision_hash":   current.Summary.PolicyDecisionHash,
		"approval_request_hash":  current.Summary.RequestDigest,
		"approval_decision_hash": current.Summary.DecisionDigest,
	}
	if strings.TrimSpace(current.Summary.BoundScope.ActionKind) == policyengine.ActionKindBackendPosture {
		details["action_kind"] = current.Summary.BoundScope.ActionKind
	}
	return s.AppendTrustedAuditEvent("backend_posture_applied", "brokerapi", details)
}
