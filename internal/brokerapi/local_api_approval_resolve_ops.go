package brokerapi

import (
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) resolveApprovalDigests(requestID string, req ApprovalResolveRequest) (string, string, *ErrorResponse) {
	approvalID, err := approvalIDFromRequest(req.SignedApprovalRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", &errOut
	}
	if req.ApprovalID != "" && req.ApprovalID != approvalID {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_id does not match signed_approval_request")
		return "", "", &errOut
	}
	decisionDigest, err := signedEnvelopeDigest(req.SignedApprovalDecision)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", &errOut
	}
	return approvalID, decisionDigest, nil
}

func (s *Service) promoteAndHeadResolvedArtifact(requestID string, req ApprovalResolveRequest) (artifacts.ArtifactRecord, *ErrorResponse) {
	ref, promoteErr := s.PromoteApprovedExcerpt(artifacts.PromotionRequest{
		UnapprovedDigest:      req.UnapprovedDigest,
		Approver:              req.Approver,
		ApprovalRequest:       &req.SignedApprovalRequest,
		ApprovalDecision:      &req.SignedApprovalDecision,
		RepoPath:              req.RepoPath,
		Commit:                req.Commit,
		ExtractorToolVersion:  req.ExtractorToolVersion,
		FullContentVisible:    req.FullContentVisible,
		ExplicitViewFull:      req.ExplicitViewFull,
		BulkRequest:           req.BulkRequest,
		BulkApprovalConfirmed: req.BulkApprovalConfirmed,
	})
	if promoteErr != nil {
		errOut := s.errorFromStore(requestID, promoteErr)
		return artifacts.ArtifactRecord{}, &errOut
	}
	head, err := s.Head(ref.Digest)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return artifacts.ArtifactRecord{}, &errOut
	}
	return head, nil
}

func buildResolvedApprovalRecord(req ApprovalResolveRequest, approvalID, decisionDigest string) approvalRecord {
	now := time.Now().UTC().Format(time.RFC3339)
	return approvalRecord{
		Summary: ApprovalSummary{
			SchemaID:               "runecode.protocol.v0.ApprovalSummary",
			SchemaVersion:          "0.1.0",
			ApprovalID:             approvalID,
			Status:                 "approved",
			RequestedAt:            now,
			DecidedAt:              now,
			ApprovalTriggerCode:    "excerpt_promotion",
			ChangesIfApproved:      "Promote reviewed file excerpts for downstream use.",
			ApprovalAssuranceLevel: decodeDecisionString(req.SignedApprovalDecision.Payload, "approval_assurance_level", "reauthenticated"),
			PresenceMode:           decodeDecisionString(req.SignedApprovalDecision.Payload, "presence_mode", "hardware_touch"),
			BoundScope:             req.BoundScope,
			PolicyDecisionHash:     decisionDigest,
			RequestDigest:          approvalID,
			DecisionDigest:         decisionDigest,
		},
		RequestEnvelope:  &req.SignedApprovalRequest,
		DecisionEnvelope: &req.SignedApprovalDecision,
	}
}

func buildApprovalResolveResponse(requestID string, record approvalRecord, head artifacts.ArtifactRecord) ApprovalResolveResponse {
	return ApprovalResolveResponse{
		SchemaID:         "runecode.protocol.v0.ApprovalResolveResponse",
		SchemaVersion:    "0.1.0",
		RequestID:        requestID,
		ResolutionStatus: "resolved",
		Approval:         record.Summary,
		ApprovedArtifact: ptrArtifactSummary(toArtifactSummary(head)),
	}
}
