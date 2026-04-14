package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	approvalResolveDetailsSchemaID                      = "runecode.protocol.v0.ApprovalResolveDetails"
	approvalResolveDetailsSchemaVersion                 = "0.1.0"
	approvalResolvePromotionDetailsSchemaID             = "runecode.protocol.v0.ApprovalResolvePromotionDetails"
	approvalResolvePromotionDetailsSchemaVersion        = "0.1.0"
	approvalResolveBackendSelectionDetailsSchemaID      = "runecode.protocol.v0.ApprovalResolveBackendPostureSelectionDetail"
	approvalResolveBackendSelectionDetailsSchemaVersion = "0.1.0"
)

func (req ApprovalResolveRequest) normalizedResolutionDetails() ApprovalResolveDetails {
	details := req.ResolutionDetails
	details = normalizeResolveDetailsEnvelope(details)
	if details.Promotion == nil && req.BoundScope.ActionKind != "backend_posture_change" && req.hasLegacyPromotionResolutionFields() {
		details.Promotion = legacyPromotionResolutionDetails(req)
	}
	details.Promotion = normalizePromotionResolutionDetails(details.Promotion)
	details.BackendPostureSelection = normalizeBackendSelectionResolutionDetails(details.BackendPostureSelection)
	if details.BackendPostureSelection == nil && req.BoundScope.ActionKind == "backend_posture_change" {
		if selection, ok := backendResolveSelectionFromSignedRequest(req.SignedApprovalRequest); ok {
			details.BackendPostureSelection = &selection
		}
	}
	return details
}

func normalizeResolveDetailsEnvelope(details ApprovalResolveDetails) ApprovalResolveDetails {
	if strings.TrimSpace(details.SchemaID) == "" {
		details.SchemaID = approvalResolveDetailsSchemaID
	}
	if strings.TrimSpace(details.SchemaVersion) == "" {
		details.SchemaVersion = approvalResolveDetailsSchemaVersion
	}
	return details
}

func legacyPromotionResolutionDetails(req ApprovalResolveRequest) *ApprovalResolvePromotionDetails {
	return &ApprovalResolvePromotionDetails{
		SchemaID:              approvalResolvePromotionDetailsSchemaID,
		SchemaVersion:         approvalResolvePromotionDetailsSchemaVersion,
		UnapprovedDigest:      req.UnapprovedDigest,
		Approver:              req.Approver,
		RepoPath:              req.RepoPath,
		Commit:                req.Commit,
		ExtractorToolVersion:  req.ExtractorToolVersion,
		FullContentVisible:    req.FullContentVisible,
		ExplicitViewFull:      req.ExplicitViewFull,
		BulkRequest:           req.BulkRequest,
		BulkApprovalConfirmed: req.BulkApprovalConfirmed,
	}
}

func normalizePromotionResolutionDetails(details *ApprovalResolvePromotionDetails) *ApprovalResolvePromotionDetails {
	if details == nil {
		return nil
	}
	if strings.TrimSpace(details.SchemaID) == "" {
		details.SchemaID = approvalResolvePromotionDetailsSchemaID
	}
	if strings.TrimSpace(details.SchemaVersion) == "" {
		details.SchemaVersion = approvalResolvePromotionDetailsSchemaVersion
	}
	return details
}

func normalizeBackendSelectionResolutionDetails(details *ApprovalResolveBackendPostureSelectionDetail) *ApprovalResolveBackendPostureSelectionDetail {
	if details == nil {
		return nil
	}
	if strings.TrimSpace(details.SchemaID) == "" {
		details.SchemaID = approvalResolveBackendSelectionDetailsSchemaID
	}
	if strings.TrimSpace(details.SchemaVersion) == "" {
		details.SchemaVersion = approvalResolveBackendSelectionDetailsSchemaVersion
	}
	return details
}

func (req ApprovalResolveRequest) hasLegacyPromotionResolutionFields() bool {
	if strings.TrimSpace(req.UnapprovedDigest) != "" {
		return true
	}
	if strings.TrimSpace(req.RepoPath) != "" || strings.TrimSpace(req.Commit) != "" || strings.TrimSpace(req.ExtractorToolVersion) != "" {
		return true
	}
	if strings.TrimSpace(req.Approver) != "" {
		return true
	}
	if req.FullContentVisible || req.ExplicitViewFull || req.BulkRequest || req.BulkApprovalConfirmed {
		return true
	}
	return false
}

func normalizeApprovalResolveRequest(req ApprovalResolveRequest) ApprovalResolveRequest {
	req.ResolutionDetails = req.normalizedResolutionDetails()
	return req
}

func backendResolveSelectionFromSignedRequest(envelope trustpolicy.SignedObjectEnvelope) (ApprovalResolveBackendPostureSelectionDetail, bool) {
	payload, err := decodeApprovalRequestPayload(envelope)
	if err != nil {
		return ApprovalResolveBackendPostureSelectionDetail{}, false
	}
	details, _ := payload["details"].(map[string]any)
	if len(details) == 0 {
		return ApprovalResolveBackendPostureSelectionDetail{}, false
	}
	targetInstanceID := strings.TrimSpace(stringFieldOrEmpty(details, "target_instance_id"))
	targetBackendKind := strings.TrimSpace(stringFieldOrEmpty(details, "target_backend_kind"))
	if targetInstanceID == "" || targetBackendKind == "" {
		return ApprovalResolveBackendPostureSelectionDetail{}, false
	}
	return ApprovalResolveBackendPostureSelectionDetail{
		SchemaID:          approvalResolveBackendSelectionDetailsSchemaID,
		SchemaVersion:     approvalResolveBackendSelectionDetailsSchemaVersion,
		TargetInstanceID:  targetInstanceID,
		TargetBackendKind: targetBackendKind,
	}, true
}
