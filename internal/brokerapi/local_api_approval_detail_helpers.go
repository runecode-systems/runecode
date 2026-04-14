package brokerapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) approvalDetailFromRecord(record approvalRecord) (ApprovalDetail, error) {
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	detail := ApprovalDetail{
		SchemaID:              "runecode.protocol.v0.ApprovalDetail",
		SchemaVersion:         "0.1.0",
		ApprovalID:            record.Summary.ApprovalID,
		LifecycleDetail:       approvalLifecycleDetailFromRecord(record, now),
		WhatChangesIfApproved: approvalWhatChangesIfApprovedFromRecord(record),
		BlockedWorkScope:      approvalBlockedWorkScopeFromRecord(record),
	}
	if policyRef := strings.TrimSpace(record.Summary.PolicyDecisionHash); policyRef != "" {
		if decision, ok := s.PolicyDecisionGet(policyRef); ok {
			detail.PolicyReasonCode = strings.TrimSpace(decision.PolicyReasonCode)
		}
	}
	if strings.TrimSpace(record.Summary.BoundScope.ActionKind) == policyengine.ActionKindBackendPosture {
		if selection, ok := approvalBackendPostureSelectionFromRecord(record); ok {
			detail.BackendPostureSelection = &selection
		}
	}
	if strings.TrimSpace(record.Summary.BoundScope.ActionKind) == "stage_summary_sign_off" {
		return approvalStageSignOffDetail(record, detail)
	}
	return approvalExactActionDetail(record, detail)
}

func approvalBackendPostureSelectionFromRecord(record approvalRecord) (ApprovalBackendPostureSelection, bool) {
	if record.RequestEnvelope == nil {
		return ApprovalBackendPostureSelection{}, false
	}
	requestPayload, err := decodeApprovalRequestPayload(*record.RequestEnvelope)
	if err != nil {
		return ApprovalBackendPostureSelection{}, false
	}
	details, _ := requestPayload["details"].(map[string]any)
	if len(details) == 0 {
		return ApprovalBackendPostureSelection{}, false
	}
	requestedPosture := strings.TrimSpace(stringFieldOrEmpty(details, "requested_posture"))
	if requestedPosture == "" {
		requestedPosture = strings.TrimSpace(stringFieldOrEmpty(details, "selection_mode"))
	}
	selection := ApprovalBackendPostureSelection{
		SchemaID:                     "runecode.protocol.v0.ApprovalBackendPostureSelection",
		SchemaVersion:                "0.1.0",
		TargetBackendKind:            valueOrUnknown(strings.TrimSpace(stringFieldOrEmpty(details, "target_backend_kind")), "unknown"),
		SelectionMode:                valueOrUnknown(strings.TrimSpace(stringFieldOrEmpty(details, "selection_mode")), "explicit_selection"),
		ChangeKind:                   valueOrUnknown(strings.TrimSpace(stringFieldOrEmpty(details, "change_kind")), "unknown"),
		RequestedPosture:             valueOrUnknown(requestedPosture, "unknown"),
		AssuranceChangeKind:          valueOrUnknown(strings.TrimSpace(stringFieldOrEmpty(details, "assurance_change_kind")), "unknown"),
		OptInKind:                    valueOrUnknown(strings.TrimSpace(stringFieldOrEmpty(details, "opt_in_kind")), "exact_action_approval"),
		ReducedAssuranceAcknowledged: boolFieldOrFalse(details, "reduced_assurance_acknowledged"),
	}
	return selection, true
}

func stringFieldOrEmpty(object map[string]any, field string) string {
	value, _ := object[field].(string)
	return value
}

func boolFieldOrFalse(object map[string]any, field string) bool {
	value, ok := object[field].(bool)
	if !ok {
		return false
	}
	return value
}

func valueOrUnknown(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	if fallback != "" {
		return fallback
	}
	return "unknown"
}

func approvalStageSignOffDetail(record approvalRecord, detail ApprovalDetail) (ApprovalDetail, error) {
	detail.BindingKind = "stage_sign_off"
	detail.BoundStageSummaryHash = stageSummaryHashForApprovalDetail(record)
	if detail.BoundStageSummaryHash == "" {
		return ApprovalDetail{}, fmt.Errorf("stage sign-off approval missing bound stage_summary_hash")
	}
	detail.BoundIdentity = approvalBoundIdentityFromRecord(record, detail.BindingKind, "", detail.BoundStageSummaryHash)
	return detail, nil
}

func approvalExactActionDetail(record approvalRecord, detail ApprovalDetail) (ApprovalDetail, error) {
	detail.BindingKind = "exact_action"
	detail.BoundActionHash = record.ActionRequestHash
	if strings.TrimSpace(detail.BoundActionHash) == "" {
		return ApprovalDetail{}, fmt.Errorf("exact-action approval missing bound action hash")
	}
	detail.BoundIdentity = approvalBoundIdentityFromRecord(record, detail.BindingKind, detail.BoundActionHash, "")
	return detail, nil
}

func approvalLifecycleDetailFromRecord(record approvalRecord, now time.Time) ApprovalLifecycleDetail {
	state := strings.TrimSpace(record.Summary.Status)
	detail := ApprovalLifecycleDetail{
		SchemaID:            "runecode.protocol.v0.ApprovalLifecycleDetail",
		SchemaVersion:       "0.1.0",
		LifecycleState:      state,
		LifecycleReasonCode: lifecycleReasonCodeForApprovalSummary(record.Summary),
		Stale:               approvalIsStale(record, now),
		StaleReasonCode:     staleReasonCodeForApprovalRecord(record, now),
		ExpiresAt:           strings.TrimSpace(record.Summary.ExpiresAt),
		DecidedAt:           strings.TrimSpace(record.Summary.DecidedAt),
		ConsumedAt:          strings.TrimSpace(record.Summary.ConsumedAt),
	}
	if state == "superseded" {
		detail.SupersededByApprovalID = strings.TrimSpace(record.Summary.SupersededByApprovalID)
	}
	if !detail.Stale {
		detail.StaleReasonCode = ""
	}
	return detail
}

func lifecycleReasonCodeForApprovalSummary(summary ApprovalSummary) string {
	switch strings.TrimSpace(summary.Status) {
	case "pending":
		return "approval_pending"
	case "approved":
		return "approval_approved"
	case "denied":
		return "approval_denied"
	case "expired":
		return "approval_expired"
	case "cancelled":
		return "approval_cancelled"
	case "superseded":
		return "approval_superseded"
	case "consumed":
		return "approval_consumed"
	default:
		return "approval_lifecycle_unknown"
	}
}

func approvalIsStale(record approvalRecord, now time.Time) bool {
	state := strings.TrimSpace(record.Summary.Status)
	return state == "superseded" || state == "expired" || staleReasonCodeForApprovalRecord(record, now) != ""
}

func staleReasonCodeForApprovalRecord(record approvalRecord, now time.Time) string {
	if strings.TrimSpace(record.Summary.Status) == "superseded" {
		return "approval_superseded"
	}
	if strings.TrimSpace(record.Summary.Status) == "expired" {
		return "approval_expired"
	}
	if isApprovalExpiredAt(record, now) {
		return "approval_expired"
	}
	return ""
}

func isApprovalExpiredAt(record approvalRecord, at time.Time) bool {
	expiresAt, ok := parseRFC3339(strings.TrimSpace(record.Summary.ExpiresAt))
	if !ok {
		return false
	}
	return !at.Before(expiresAt)
}

func approvalWhatChangesIfApprovedFromRecord(record approvalRecord) ApprovalWhatChangesIfApproved {
	summary := strings.TrimSpace(record.Summary.ChangesIfApproved)
	if summary == "" {
		summary = approvalChangesIfApprovedDefault
	}
	return ApprovalWhatChangesIfApproved{
		SchemaID:      "runecode.protocol.v0.ApprovalWhatChangesIfApproved",
		SchemaVersion: "0.1.0",
		Summary:       summary,
		EffectKind:    approvalEffectKindForActionKind(record.Summary.BoundScope.ActionKind),
	}
}

func approvalBlockedWorkScopeFromRecord(record approvalRecord) ApprovalBlockedWorkScope {
	bound := record.Summary.BoundScope
	return ApprovalBlockedWorkScope{
		SchemaID:       "runecode.protocol.v0.ApprovalBlockedWorkScope",
		SchemaVersion:  "0.1.0",
		ScopeKind:      approvalBlockedScopeKind(bound),
		WorkspaceID:    bound.WorkspaceID,
		RunID:          bound.RunID,
		StageID:        bound.StageID,
		StepID:         bound.StepID,
		RoleInstanceID: bound.RoleInstanceID,
		ActionKind:     bound.ActionKind,
	}
}
