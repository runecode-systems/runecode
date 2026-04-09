package brokerapi

import (
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func decodeApprovalRequestPayload(envelope trustpolicy.SignedObjectEnvelope) (map[string]any, error) {
	request := map[string]any{}
	if err := json.Unmarshal(envelope.Payload, &request); err != nil {
		return nil, fmt.Errorf("decode signed approval request payload: %w", err)
	}
	return request, nil
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
