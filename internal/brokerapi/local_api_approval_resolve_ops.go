package brokerapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type approvalResolutionInput struct {
	approvalID     string
	decisionDigest string
	requestPayload map[string]any
	decision       trustpolicy.ApprovalDecision
	outcome        string
}

type approvalResumeResult struct {
	statusOverride     string
	resolutionReason   string
	supersededByID     string
	approvedArtifact   *ArtifactSummary
	notYetSupportedFor string
}

func (s *Service) resolveApprovalInput(requestID string, req ApprovalResolveRequest) (approvalResolutionInput, *ErrorResponse) {
	approvalID, errResp := s.resolveApprovalRequestIdentity(requestID, req)
	if errResp != nil {
		return approvalResolutionInput{}, errResp
	}
	requestPayload, errResp := s.resolveApprovalRequestPayload(requestID, req)
	if errResp != nil {
		return approvalResolutionInput{}, errResp
	}
	decisionDigest, decision, errResp := s.resolveApprovalDecision(requestID, req)
	if errResp != nil {
		return approvalResolutionInput{}, errResp
	}
	if errResp := s.verifyApprovalBindingAndOutcome(requestID, req, decision); errResp != nil {
		return approvalResolutionInput{}, errResp
	}
	return approvalResolutionInput{
		approvalID:     approvalID,
		decisionDigest: decisionDigest,
		requestPayload: requestPayload,
		decision:       decision,
		outcome:        decision.DecisionOutcome,
	}, nil
}

func (s *Service) resolveApprovalRequestIdentity(requestID string, req ApprovalResolveRequest) (string, *ErrorResponse) {
	approvalID, err := approvalIDFromRequest(req.SignedApprovalRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", &errOut
	}
	if req.ApprovalID != "" && req.ApprovalID != approvalID {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_id does not match signed_approval_request")
		return "", &errOut
	}
	return approvalID, nil
}

func (s *Service) resolveApprovalRequestPayload(requestID string, req ApprovalResolveRequest) (map[string]any, *ErrorResponse) {
	if err := s.verifySignedApprovalRequestEnvelope(req.SignedApprovalRequest); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return nil, &errOut
	}
	requestPayload, err := decodeApprovalRequestPayload(req.SignedApprovalRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return nil, &errOut
	}
	return requestPayload, nil
}

func (s *Service) resolveApprovalDecision(requestID string, req ApprovalResolveRequest) (string, trustpolicy.ApprovalDecision, *ErrorResponse) {
	decisionDigest, err := signedEnvelopeDigest(req.SignedApprovalDecision)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", trustpolicy.ApprovalDecision{}, &errOut
	}
	decision, err := decodeApprovalDecision(req.SignedApprovalDecision)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", trustpolicy.ApprovalDecision{}, &errOut
	}
	if err := s.verifySignedApprovalDecisionEnvelope(req.SignedApprovalDecision); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", trustpolicy.ApprovalDecision{}, &errOut
	}
	if err := s.verifyApprovalDecisionApproverBinding(req, decision); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", trustpolicy.ApprovalDecision{}, &errOut
	}
	return decisionDigest, decision, nil
}

func (s *Service) verifyApprovalBindingAndOutcome(requestID string, req ApprovalResolveRequest, decision trustpolicy.ApprovalDecision) *ErrorResponse {
	if err := verifyApprovalDecisionBinding(req.SignedApprovalRequest, decision); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return &errOut
	}
	if _, ok := approvalStatusForDecisionOutcome(decision.DecisionOutcome); !ok {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("unsupported decision_outcome %q", decision.DecisionOutcome))
		return &errOut
	}
	return nil
}

func (s *Service) promoteAndHeadResolvedArtifact(requestID string, req ApprovalResolveRequest) (artifacts.ArtifactRecord, *ErrorResponse) {
	promotion := req.promotionResolveDetails()
	ref, promoteErr := s.PromoteApprovedExcerpt(artifacts.PromotionRequest{
		UnapprovedDigest:      promotion.UnapprovedDigest,
		Approver:              promotion.Approver,
		ApprovalRequest:       &req.SignedApprovalRequest,
		ApprovalDecision:      &req.SignedApprovalDecision,
		RepoPath:              promotion.RepoPath,
		Commit:                promotion.Commit,
		ExtractorToolVersion:  promotion.ExtractorToolVersion,
		FullContentVisible:    promotion.FullContentVisible,
		ExplicitViewFull:      promotion.ExplicitViewFull,
		BulkRequest:           promotion.BulkRequest,
		BulkApprovalConfirmed: promotion.BulkApprovalConfirmed,
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

func buildResolvedApprovalRecordForOutcome(req ApprovalResolveRequest, prior approvalRecord, approvalID, decisionDigest, status, supersededBy string, resolvedAt time.Time) (approvalRecord, error) {
	now := resolvedAt.UTC().Format(time.RFC3339)
	if strings.TrimSpace(status) == "" {
		return approvalRecord{}, fmt.Errorf("approval status is required")
	}
	resolved := resolvedApprovalEvidence(req, prior, approvalID, decisionDigest, status, now)
	return approvalRecord{
		Summary: ApprovalSummary{
			SchemaID:               "runecode.protocol.v0.ApprovalSummary",
			SchemaVersion:          "0.1.0",
			ApprovalID:             approvalID,
			Status:                 status,
			RequestedAt:            resolved.requestedAt,
			ExpiresAt:              prior.Summary.ExpiresAt,
			DecidedAt:              now,
			ApprovalTriggerCode:    prior.Summary.ApprovalTriggerCode,
			ChangesIfApproved:      resolved.changesIfApproved,
			ApprovalAssuranceLevel: decodeDecisionString(req.SignedApprovalDecision.Payload, "approval_assurance_level", "reauthenticated"),
			PresenceMode:           decodeDecisionString(req.SignedApprovalDecision.Payload, "presence_mode", "hardware_touch"),
			BoundScope:             prior.Summary.BoundScope,
			PolicyDecisionHash:     prior.Summary.PolicyDecisionHash,
			SupersededByApprovalID: supersededBy,
			RequestDigest:          prior.Summary.RequestDigest,
			DecisionDigest:         decisionDigest,
			ScopeDigest:            resolved.scopeDigest,
			ArtifactSetDigest:      resolved.artifactSetDigest,
			DiffDigest:             resolved.diffDigest,
			SummaryPreviewDigest:   resolved.summaryPreviewDigest,
			ConsumedActionHash:     resolved.consumedActionHash,
			ConsumedArtifactDigest: resolved.consumedArtifactDigest,
			ConsumptionLinkDigest:  resolved.consumptionLinkDigest,
		},
		RequestEnvelope:        &req.SignedApprovalRequest,
		DecisionEnvelope:       &req.SignedApprovalDecision,
		SourceDigest:           prior.SourceDigest,
		ManifestHash:           prior.ManifestHash,
		ActionRequestHash:      prior.ActionRequestHash,
		RelevantArtifactHashes: append([]string{}, prior.RelevantArtifactHashes...),
	}, nil
}

func (s *Service) verifyApprovalDecisionApproverBinding(req ApprovalResolveRequest, decision trustpolicy.ApprovalDecision) error {
	if strings.TrimSpace(decision.Approver.PrincipalID) == "" {
		return fmt.Errorf("approval decision approver principal is required")
	}
	if requestApprover := strings.TrimSpace(req.requestApprover()); requestApprover != "" && requestApprover != strings.TrimSpace(decision.Approver.PrincipalID) {
		return fmt.Errorf("approval decision approver does not match request approver")
	}
	verifiers, err := s.trustedApprovalVerifiersForEnvelope(req.SignedApprovalDecision)
	if err != nil {
		return err
	}
	for _, verifier := range verifiers {
		if samePrincipalIdentity(verifier.OwnerPrincipal, decision.Approver) {
			return nil
		}
	}
	return fmt.Errorf("approval decision approver does not match verifier owner identity")
}

func (req ApprovalResolveRequest) promotionResolveDetails() ApprovalResolvePromotionDetails {
	details := req.normalizedResolutionDetails()
	if details.Promotion != nil {
		return *details.Promotion
	}
	return ApprovalResolvePromotionDetails{}
}

func (req ApprovalResolveRequest) requestApprover() string {
	promotion := req.promotionResolveDetails()
	if strings.TrimSpace(promotion.Approver) != "" {
		return promotion.Approver
	}
	return req.Approver
}

func samePrincipalIdentity(left trustpolicy.PrincipalIdentity, right trustpolicy.PrincipalIdentity) bool {
	if left.SchemaID != right.SchemaID {
		return false
	}
	if left.SchemaVersion != right.SchemaVersion {
		return false
	}
	if left.ActorKind != right.ActorKind {
		return false
	}
	if left.PrincipalID != right.PrincipalID {
		return false
	}
	if left.InstanceID != right.InstanceID {
		return false
	}
	return true
}

func buildApprovalResolveResponseNoArtifact(requestID string, record approvalRecord, approvedArtifact *ArtifactSummary, reasonOverride string) ApprovalResolveResponse {
	reasonCode := ""
	if record.Summary.Status != "approved" {
		reasonCode = resolutionReasonCodeForApprovalStatus(record.Summary.Status)
	}
	if strings.TrimSpace(reasonOverride) != "" {
		reasonCode = strings.TrimSpace(reasonOverride)
	}
	resolutionStatus := "resolved"
	if reasonCode == "approval_superseded" {
		resolutionStatus = "no_change"
	}
	return ApprovalResolveResponse{
		SchemaID:             "runecode.protocol.v0.ApprovalResolveResponse",
		SchemaVersion:        "0.1.0",
		RequestID:            requestID,
		ResolutionStatus:     resolutionStatus,
		ResolutionReasonCode: reasonCode,
		Approval:             record.Summary,
		ApprovedArtifact:     approvedArtifact,
	}
}
