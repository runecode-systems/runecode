package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

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

func (s *Service) verifySignedApprovalRequestEnvelope(envelope trustpolicy.SignedObjectEnvelope) error {
	if isPlaceholderApprovalRequestEnvelope(envelope) {
		return fmt.Errorf("verify signed approval request: placeholder approval request envelope is not verifiable")
	}
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
		ExpectedPayloadSchemaID:   trustpolicy.ApprovalRequestSchemaID,
		ExpectedPayloadVersion:    trustpolicy.ApprovalRequestSchemaVersion,
	}); err != nil {
		return fmt.Errorf("verify signed approval request: %w", err)
	}
	return nil
}

func isPlaceholderApprovalRequestEnvelope(envelope trustpolicy.SignedObjectEnvelope) bool {
	if envelope.Signature.KeyID != trustpolicy.KeyIDProfile {
		return false
	}
	if envelope.Signature.KeyIDValue != strings.Repeat("0", 64) {
		return false
	}
	return envelope.Signature.Signature == "cGVuZGluZw=="
}
