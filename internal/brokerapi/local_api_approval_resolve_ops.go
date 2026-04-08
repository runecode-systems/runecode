package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) resolveApprovalDigestsAndOutcome(requestID string, req ApprovalResolveRequest) (string, string, string, *ErrorResponse) {
	approvalID, err := approvalIDFromRequest(req.SignedApprovalRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", "", &errOut
	}
	if req.ApprovalID != "" && req.ApprovalID != approvalID {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_id does not match signed_approval_request")
		return "", "", "", &errOut
	}
	decisionDigest, err := signedEnvelopeDigest(req.SignedApprovalDecision)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", "", &errOut
	}
	decision, err := decodeApprovalDecision(req.SignedApprovalDecision)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", "", &errOut
	}
	if err := s.verifySignedApprovalDecisionEnvelope(req.SignedApprovalDecision); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", "", &errOut
	}
	if err := verifyApprovalDecisionBinding(req.SignedApprovalRequest, decision); err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", "", &errOut
	}
	if _, ok := approvalStatusForDecisionOutcome(decision.DecisionOutcome); !ok {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("unsupported decision_outcome %q", decision.DecisionOutcome))
		return "", "", "", &errOut
	}
	return approvalID, decisionDigest, decision.DecisionOutcome, nil
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

func buildResolvedApprovalRecordForOutcome(req ApprovalResolveRequest, prior approvalRecord, approvalID, decisionDigest, outcome string) (approvalRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	status, ok := approvalStatusForDecisionOutcome(outcome)
	if !ok {
		return approvalRecord{}, fmt.Errorf("unsupported decision_outcome %q", outcome)
	}
	requestedAt := prior.Summary.RequestedAt
	if requestedAt == "" {
		requestedAt = now
	}
	changesIfApproved := prior.Summary.ChangesIfApproved
	if changesIfApproved == "" {
		changesIfApproved = approvalChangesIfApprovedDefault
	}
	return approvalRecord{
		Summary: ApprovalSummary{
			SchemaID:               "runecode.protocol.v0.ApprovalSummary",
			SchemaVersion:          "0.1.0",
			ApprovalID:             approvalID,
			Status:                 status,
			RequestedAt:            requestedAt,
			ExpiresAt:              prior.Summary.ExpiresAt,
			DecidedAt:              now,
			ApprovalTriggerCode:    "excerpt_promotion",
			ChangesIfApproved:      changesIfApproved,
			ApprovalAssuranceLevel: decodeDecisionString(req.SignedApprovalDecision.Payload, "approval_assurance_level", "reauthenticated"),
			PresenceMode:           decodeDecisionString(req.SignedApprovalDecision.Payload, "presence_mode", "hardware_touch"),
			BoundScope:             req.BoundScope,
			PolicyDecisionHash:     decisionDigest,
			RequestDigest:          approvalID,
			DecisionDigest:         decisionDigest,
		},
		RequestEnvelope:  &req.SignedApprovalRequest,
		DecisionEnvelope: &req.SignedApprovalDecision,
		SourceDigest:     req.UnapprovedDigest,
	}, nil
}

func buildApprovalResolveResponseNoArtifact(requestID string, record approvalRecord, approvedArtifact *ArtifactSummary) ApprovalResolveResponse {
	reasonCode := ""
	if record.Summary.Status != "approved" {
		reasonCode = resolutionReasonCodeForApprovalStatus(record.Summary.Status)
	}
	return ApprovalResolveResponse{
		SchemaID:             "runecode.protocol.v0.ApprovalResolveResponse",
		SchemaVersion:        "0.1.0",
		RequestID:            requestID,
		ResolutionStatus:     "resolved",
		ResolutionReasonCode: reasonCode,
		Approval:             record.Summary,
		ApprovedArtifact:     approvedArtifact,
	}
}

func decodeApprovalDecision(envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.ApprovalDecision, error) {
	decision := trustpolicy.ApprovalDecision{}
	if err := json.Unmarshal(envelope.Payload, &decision); err != nil {
		return trustpolicy.ApprovalDecision{}, fmt.Errorf("decode signed approval decision payload: %w", err)
	}
	return decision, nil
}

func verifyApprovalDecisionBinding(requestEnvelope trustpolicy.SignedObjectEnvelope, decision trustpolicy.ApprovalDecision) error {
	if err := trustpolicy.ValidateApprovalDecisionEvidence(decision); err != nil {
		return fmt.Errorf("validate approval decision evidence: %w", err)
	}
	requestDigest, err := approvalIDFromRequest(requestEnvelope)
	if err != nil {
		return fmt.Errorf("derive approval request digest: %w", err)
	}
	decisionRequestHash, err := decision.ApprovalRequestHash.Identity()
	if err != nil {
		return fmt.Errorf("approval_request_hash: %w", err)
	}
	if requestDigest != decisionRequestHash {
		return fmt.Errorf("approval_request_hash %q does not match signed_approval_request digest %q", decisionRequestHash, requestDigest)
	}
	return nil
}

func (s *Service) verifySignedApprovalDecisionEnvelope(envelope trustpolicy.SignedObjectEnvelope) error {
	verifiers, err := s.trustedApprovalVerifiersForEnvelope(envelope)
	if err != nil {
		return err
	}
	registry, err := trustpolicy.NewVerifierRegistry(verifiers)
	if err != nil {
		return fmt.Errorf("create verifier registry: %w", err)
	}
	if err := trustpolicy.VerifySignedEnvelope(envelope, registry, trustpolicy.EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   trustpolicy.ApprovalDecisionSchemaID,
		ExpectedPayloadVersion:    trustpolicy.ApprovalDecisionSchemaVersion,
	}); err != nil {
		return fmt.Errorf("verify signed approval decision: %w", err)
	}
	return nil
}

func (s *Service) trustedApprovalVerifiersForEnvelope(envelope trustpolicy.SignedObjectEnvelope) ([]trustpolicy.VerifierRecord, error) {
	records := make([]trustpolicy.VerifierRecord, 0)
	for _, artifactRecord := range s.List() {
		trusted, trustErr := s.isTrustedVerifierArtifact(artifactRecord)
		verifier, skip, err := selectVerifierRecordForEnvelope(artifactRecord, trusted, trustErr, envelope, s)
		if skip {
			if err != nil {
				return nil, err
			}
			continue
		}
		records = append(records, verifier)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("trusted verifier not found for signed approval decision")
	}
	return records, nil
}

func selectVerifierRecordForEnvelope(record artifacts.ArtifactRecord, trusted bool, trustErr error, envelope trustpolicy.SignedObjectEnvelope, service *Service) (trustpolicy.VerifierRecord, bool, error) {
	if trustErr != nil {
		return trustpolicy.VerifierRecord{}, true, trustErr
	}
	if !trusted {
		return trustpolicy.VerifierRecord{}, true, nil
	}
	verifier, err := service.loadVerifierRecord(record)
	if err != nil {
		return trustpolicy.VerifierRecord{}, true, err
	}
	if !matchesVerifierIdentity(verifier, envelope) {
		return trustpolicy.VerifierRecord{}, true, nil
	}
	if !isApprovalAuthorityVerifier(verifier) {
		return trustpolicy.VerifierRecord{}, true, nil
	}
	return verifier, false, nil
}

func (s *Service) loadVerifierRecord(record artifacts.ArtifactRecord) (trustpolicy.VerifierRecord, error) {
	reader, err := s.Get(record.Reference.Digest)
	if err != nil {
		return trustpolicy.VerifierRecord{}, fmt.Errorf("read trusted verifier artifact %q: %w", record.Reference.Digest, err)
	}
	blob, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		if readErr != nil {
			return trustpolicy.VerifierRecord{}, fmt.Errorf("read trusted verifier artifact %q bytes: %w", record.Reference.Digest, readErr)
		}
		return trustpolicy.VerifierRecord{}, fmt.Errorf("close trusted verifier artifact %q reader: %w", record.Reference.Digest, closeErr)
	}
	verifier := trustpolicy.VerifierRecord{}
	if err := json.Unmarshal(blob, &verifier); err != nil {
		return trustpolicy.VerifierRecord{}, fmt.Errorf("decode trusted verifier artifact %q: %w", record.Reference.Digest, err)
	}
	return verifier, nil
}

func matchesVerifierIdentity(verifier trustpolicy.VerifierRecord, envelope trustpolicy.SignedObjectEnvelope) bool {
	return verifier.KeyID == envelope.Signature.KeyID && verifier.KeyIDValue == envelope.Signature.KeyIDValue
}

func isApprovalAuthorityVerifier(verifier trustpolicy.VerifierRecord) bool {
	return verifier.LogicalPurpose == "approval_authority" && verifier.LogicalScope == "user"
}

func approvalStatusForDecisionOutcome(outcome string) (string, bool) {
	switch outcome {
	case "approve":
		return "approved", true
	case "deny":
		return "denied", true
	case "expired":
		return "expired", true
	case "cancelled":
		return "cancelled", true
	default:
		return "", false
	}
}

func resolutionReasonCodeForApprovalStatus(status string) string {
	switch status {
	case "denied":
		return "approval_denied"
	case "expired":
		return "approval_expired"
	case "cancelled":
		return "approval_cancelled"
	default:
		return "approval_resolved"
	}
}
